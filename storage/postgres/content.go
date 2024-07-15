package postgres

import (
	pb "content/genproto/content"
	"context"
	"database/sql"
	"fmt"
)

type ContentRepo struct {
	DB *sql.DB
}

func NewContentRepository(db *sql.DB) *ContentRepo {
	return &ContentRepo{DB: db}
}

func (c *ContentRepo) CreateStory(ctx context.Context, request *pb.CreateStoriesRequest) (*pb.CreateStoriesResponse, error) {
	tx, err := c.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	query := `
        INSERT INTO stories (title, content, location, author_id)
        VALUES ($1, $2, $3, $4)
        RETURNING id, title, content, location, author_id, created_at
    `

	var createdStory pb.CreateStoriesResponse
	err = tx.QueryRowContext(ctx, query, request.Title, request.Content, request.Location, request.UserId).Scan(
		&createdStory.Id, &createdStory.Title, &createdStory.Content, &createdStory.Location, &createdStory.AuthorId, &createdStory.CreatedAt)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tagQuery := `INSERT INTO story_tags (story_id, tag) VALUES ($1, $2)`
	for _, tag := range request.Tags {
		_, err := tx.ExecContext(ctx, tagQuery, createdStory.Id, tag)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	createdStory.Tags = request.Tags

	return &createdStory, nil
}

func (c *ContentRepo) UpdateStory(ctx context.Context, request *pb.UpdateStoriesReq) (*pb.UpdateStoriesRes, error) {
	tx, err := c.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	query := `
        UPDATE stories
        SET title = $1, content = $2, updated_at = CURRENT_TIMESTAMP
        WHERE id = $3 and deleted_at=0
        RETURNING id, title, content, location, author_id, updated_at
    `

	var updatedStory pb.UpdateStoriesRes
	err = tx.QueryRowContext(ctx, query, request.Title, request.Content, request.Id).Scan(
		&updatedStory.Id, &updatedStory.Title, &updatedStory.Content, &updatedStory.Location, &updatedStory.AuthorId, &updatedStory.UpdatedAt)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tagQuery := `SELECT tag FROM story_tags WHERE story_id = $1`
	rows, err := tx.QueryContext(ctx, tagQuery, updatedStory.Id)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			tx.Rollback()
			return nil, err
		}
		tags = append(tags, tag)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	updatedStory.Tags = tags

	return &updatedStory, nil
}

func (c *ContentRepo) DeleteStory(ctx context.Context, id *pb.StoryId) error {
	// Prepare the SQL query to mark the story as deleted by setting the deleted_at field
	query := `
        UPDATE stories
        SET deleted_at = date_part('epoch', current_timestamp)::INT
        WHERE id = $1 and deleted_at = 0
    `
	// Execute the query
	_, err := c.DB.ExecContext(ctx, query, id.Id)
	if err != nil {
		return err
	}

	return nil
}

func (c *ContentRepo) GetAllStory(ctx context.Context, request *pb.GetAllStoriesReq) (*pb.GetAllStoriesRes, error) {
	query := `
        SELECT s.id, s.title, s.location, s.likes_count, s.comments_count, u.id, u.username, u.full_name
        FROM stories s
        JOIN users u ON s.author_id = u.id
        WHERE s.deleted_at = 0
        LIMIT $1 OFFSET $2
    `

	rows, err := c.DB.QueryContext(ctx, query, request.Limit, request.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stories []*pb.Stories
	for rows.Next() {
		var story pb.Stories
		var author pb.Author

		err := rows.Scan(
			&story.StoryId,
			&story.Title,
			&story.Location,
			&story.LikesCount,
			&story.CommentsCount,
			&author.UserId,
			&author.Username,
			&author.FullName,
		)
		if err != nil {
			return nil, err
		}

		story.Author = &author
		stories = append(stories, &story)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Count total number of stories
	countQuery := `SELECT COUNT(*) FROM stories WHERE deleted_at = 0`
	var total int64
	err = c.DB.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, err
	}

	response := &pb.GetAllStoriesRes{
		Stories: stories,
		Total:   total,
		Offset:  request.Offset,
		Limit:   request.Limit,
	}

	return response, nil
}

func (c *ContentRepo) GetStoryById(ctx context.Context, id *pb.StoryId) (*pb.GetStoryRes, error) {
	// Query to get the story details
	storyQuery := `
        SELECT s.id, s.title, s.content, s.location, s.likes_count, s.comments_count, s.created_at, s.updated_at,
               u.id, u.username, u.full_name
        FROM stories s
        JOIN users u ON s.author_id = u.id
        WHERE s.id = $1 AND s.deleted_at = 0
    `

	var story pb.GetStoryRes
	var author pb.Author

	err := c.DB.QueryRowContext(ctx, storyQuery, id.Id).Scan(
		&story.Id,
		&story.Title,
		&story.Content,
		&story.Location,
		&story.LikesCount,
		&story.CommentsCount,
		&story.CreatedAt,
		&story.UpdatedAt,
		&author.UserId,
		&author.Username,
		&author.FullName,
	)
	if err != nil {
		return nil, err
	}

	// Assign the author to the story
	story.Author = &author

	// Query to get the story tags
	tagQuery := `SELECT tag FROM story_tags WHERE story_id = $1`
	rows, err := c.DB.QueryContext(ctx, tagQuery, story.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	// Assign the tags to the story
	story.Tags = tags

	return &story, nil
}

func (c *ContentRepo) CommentToStory(ctx context.Context, req *pb.CommentStoryReq) (*pb.CommentStoryRes, error) {
	// Query to insert a new comment
	query := `
        INSERT INTO comments (id, content, author_id, story_id, created_at)
        VALUES (gen_random_uuid(), $1, $2, $3, CURRENT_TIMESTAMP)
        RETURNING id, content, author_id, story_id, created_at
    `

	var comment pb.CommentStoryRes

	// Execute the query and scan the result into the response struct
	err := c.DB.QueryRowContext(ctx, query, req.Content, req.AuthorId, req.StoryId).Scan(
		&comment.Id,
		&comment.Content,
		&comment.AuthorId,
		&comment.StoryId,
		&comment.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &comment, nil
}

func (c *ContentRepo) GetCommentsOfStory(ctx context.Context, req *pb.GetCommentsOfStoryReq) (*pb.GetCommentsOfStoryRes, error) {
	// Initialize response structure
	res := &pb.GetCommentsOfStoryRes{
		Offset: req.Offset,
		Limit:  req.Limit,
	}

	// Query to get the total number of comments for the given story
	totalQuery := `
        SELECT COUNT(*)
        FROM comments
        WHERE story_id = $1
    `
	var totalComments int64
	err := c.DB.QueryRowContext(ctx, totalQuery, req.StoryId).Scan(&totalComments)
	if err != nil {
		return nil, err
	}
	res.Total = totalComments

	// Query to get the comments for the given story with pagination
	commentsQuery := `
        SELECT c.id, c.content, c.created_at, u.id, u.username, u.full_name
        FROM comments c
        JOIN users u ON c.author_id = u.id
        WHERE c.story_id = $1
        ORDER BY c.created_at DESC
        OFFSET $2 LIMIT $3
    `
	rows, err := c.DB.QueryContext(ctx, commentsQuery, req.StoryId, req.Offset, req.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*pb.Comments
	for rows.Next() {
		var comment pb.Comments
		var author pb.Author
		err := rows.Scan(&comment.Id, &comment.Content, &comment.CreatedAt, &author.UserId, &author.Username, &author.FullName)
		if err != nil {
			return nil, err
		}
		comment.Author = &author
		comments = append(comments, &comment)
	}

	res.Comments = comments

	return res, nil
}

func (c *ContentRepo) Like(ctx context.Context, req *pb.LikeReq) (*pb.LikeRes, error) {
	// Insert the like into the database
	query := `
        INSERT INTO likes (user_id, story_id, created_at)
        VALUES ($1, $2, CURRENT_TIMESTAMP)
        ON CONFLICT (user_id, story_id) DO NOTHING
        RETURNING created_at
    `

	var likedAt string
	err := c.DB.QueryRowContext(ctx, query, req.UserId, req.StoryId).Scan(&likedAt)
	if err != nil {
		return nil, err
	}

	// Return the LikeRes message
	res := &pb.LikeRes{
		UserId:  req.UserId,
		StoryId: req.StoryId,
		LikedAt: likedAt,
	}

	return res, nil
}

func (c *ContentRepo) Itineraries(ctx context.Context, req *pb.ItinerariesReq) (*pb.ItinerariesRes, error) {
	// Begin a transaction
	tx, err := c.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Insert the itinerary
	itineraryQuery := `
        INSERT INTO itineraries (title, description, start_date, end_date, author_id, created_at)
        VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
        RETURNING id, title, description, start_date, end_date, author_id, created_at
    `
	var itinerary pb.ItinerariesRes
	err = tx.QueryRowContext(ctx, itineraryQuery, req.Title, req.Description, req.StartDate, req.EndDate, req.UserId).Scan(
		&itinerary.Id, &itinerary.Title, &itinerary.Description, &itinerary.StartDate, &itinerary.EndDate, &itinerary.UserId, &itinerary.CreatedAt)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Insert the destinations
	destinationQuery := `
        INSERT INTO itinerary_destinations (itinerary_id, name, start_date, end_date)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `
	for _, dest := range req.Destinations {
		var destinationID string
		err = tx.QueryRowContext(ctx, destinationQuery, itinerary.Id, dest.Name, dest.StartDate, dest.EndDate).Scan(&destinationID)
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		// Insert the activities for the destination
		activityQuery := `
            INSERT INTO itinerary_activities (destination_id, activity)
            VALUES ($1, $2)
        `
		for _, activity := range dest.Activities {
			_, err = tx.ExecContext(ctx, activityQuery, destinationID, activity.Text)
			if err != nil {
				tx.Rollback()
				return nil, err
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &itinerary, nil
}

func (c *ContentRepo) UpdateItineraries(ctx context.Context, req *pb.UpdateItinerariesReq) (*pb.ItinerariesRes, error) {
	// Begin a transaction
	tx, err := c.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Update the itinerary details
	query := `
        UPDATE itineraries
        SET title = $1, description = $2, updated_at = CURRENT_TIMESTAMP
        WHERE id = $3 AND deleted_at = 0
        RETURNING id, title, description, start_date, end_date, author_id, created_at
    `
	var updatedItinerary pb.ItinerariesRes
	err = tx.QueryRowContext(ctx, query, req.Title, req.Description, req.Id).Scan(
		&updatedItinerary.Id, &updatedItinerary.Title, &updatedItinerary.Description,
		&updatedItinerary.StartDate, &updatedItinerary.EndDate, &updatedItinerary.UserId,
		&updatedItinerary.CreatedAt)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &updatedItinerary, nil
}

func (c *ContentRepo) DeleteItineraries(ctx context.Context, req *pb.StoryId) error {
	tx, err := c.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Mark the itinerary as deleted by updating the deleted_at field
	query := `
        UPDATE itineraries
        SET deleted_at = date_part('epoch', current_timestamp)::INT
        WHERE id = $1 AND deleted_at = 0
    `
	_, err = tx.ExecContext(ctx, query, req.Id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (c *ContentRepo) GetItineraries(ctx context.Context, req *pb.GetItinerariesReq) (*pb.GetItinerariesRes, error) {
	// Query to get the total number of itineraries
	var total int64
	totalQuery := `SELECT COUNT(*) FROM itineraries WHERE deleted_at = 0`
	err := c.DB.QueryRowContext(ctx, totalQuery).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Query to get the itineraries with pagination
	itinerariesQuery := `
        SELECT id, title, description, start_date, end_date, author_id, created_at
        FROM itineraries
        WHERE deleted_at = 0
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `
	rows, err := c.DB.QueryContext(ctx, itinerariesQuery, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var itineraries []*pb.ItinerariesRes
	for rows.Next() {
		var itinerary pb.ItinerariesRes
		err := rows.Scan(
			&itinerary.Id,
			&itinerary.Title,
			&itinerary.Description,
			&itinerary.StartDate,
			&itinerary.EndDate,
			&itinerary.UserId,
			&itinerary.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		itineraries = append(itineraries, &itinerary)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	response := &pb.GetItinerariesRes{
		Itineraries: itineraries,
		Total:       total,
		Offset:      req.Offset,
		Limit:       req.Limit,
	}

	return response, nil
}

func (c *ContentRepo) GetItinerariesById(ctx context.Context, req *pb.StoryId) (*pb.GetItinerariesByIdRes, error) {

	itineraryQuery := `
        SELECT i.id, i.title, i.description, i.start_date, i.end_date, u.id, u.username, u.full_name
        FROM itineraries i
        JOIN users u ON i.author_id = u.id
        WHERE i.id = $1 AND i.deleted_at = 0
    `
	var itinerary pb.GetItinerariesByIdRes
	err := c.DB.QueryRowContext(ctx, itineraryQuery, req.Id).Scan(
		&itinerary.Id,
		&itinerary.Title,
		&itinerary.Description,
		&itinerary.StartDate,
		&itinerary.EndDate,
		&itinerary.Author.UserId,
		&itinerary.Author.Username,
		&itinerary.Author.FullName,
	)
	if err != nil {
		return nil, err
	}

	destinationsQuery := `
        SELECT id, name, start_date, end_date
        FROM itinerary_destinations
        WHERE itinerary_id = $1
    `
	rows, err := c.DB.QueryContext(ctx, destinationsQuery, itinerary.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var destinations []*pb.Destination
	for rows.Next() {
		var destination pb.Destination
		err := rows.Scan(
			&destination.Name,
			&destination.StartDate,
			&destination.EndDate,
		)
		if err != nil {
			return nil, err
		}

		activitiesQuery := `
            SELECT activity
            FROM itinerary_activities
            WHERE destination_id = $1
        `
		activityRows, err := c.DB.QueryContext(ctx, activitiesQuery, destination.Name)
		if err != nil {
			return nil, err
		}
		defer activityRows.Close()

		var activities []*pb.Activities
		for activityRows.Next() {
			var activity pb.Activities
			err := activityRows.Scan(&activity.Text)
			if err != nil {
				return nil, err
			}
			activities = append(activities, &activity)
		}

		destination.Activities = activities
		destinations = append(destinations, &destination)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	itinerary.Destination = destinations

	return &itinerary, nil
}

func (c *ContentRepo) CommentItineraries(ctx context.Context, req *pb.CommentItinerariesReq) (*pb.CommentItinerariesRes, error) {
	query := `
        INSERT INTO comment (id, content, author_id, itinerary_id, created_at)
        VALUES (gen_random_uuid(), $1, $2, $3, CURRENT_TIMESTAMP)
        RETURNING id, author_id, content, itinerary_id, created_at
    `

	var comment pb.CommentItinerariesRes
	err := c.DB.QueryRowContext(ctx, query, req.Content, req.AuthorId, req.ItineraryId).Scan(
		&comment.Id,
		&comment.AuthorId,
		&comment.Content,
		&comment.ItineraryId,
		&comment.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert comment: %v", err)
	}

	return &comment, nil
}

func (c *ContentRepo) GetDestinations(ctx context.Context, req *pb.GetDestinationsReq) (*pb.GetDestinationsRes, error) {
	// Prepare the SQL query to fetch destinations
	query := `
        SELECT id, name, country, description, currency
        FROM destinations
        WHERE ($1 = '' OR name ILIKE '%' || $1 || '%')
        ORDER BY name
        LIMIT $2 OFFSET $3
    `

	// Execute the query
	rows, err := c.DB.QueryContext(ctx, query, req.Name, req.Limit, req.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch destinations: %v", err)
	}
	defer rows.Close()

	var destinations []*pb.Destinations
	for rows.Next() {
		var destination pb.Destinations
		if err := rows.Scan(
			&destination.Id,
			&destination.Name,
			&destination.Country,
			&destination.Description,
			&destination.Currency,
		); err != nil {
			return nil, fmt.Errorf("failed to scan destination row: %v", err)
		}
		destinations = append(destinations, &destination)
	}

	// Count total number of destinations
	countQuery := `
        SELECT COUNT(*)
        FROM destinations
        WHERE ($1 = '' OR name ILIKE '%' || $1 || '%')
    `
	var total int64
	err = c.DB.QueryRowContext(ctx, countQuery, req.Name).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch total count of destinations: %v", err)
	}

	// Create response
	res := &pb.GetDestinationsRes{
		Destination: destinations,
		Total:       total,
		Offset:      req.Offset,
		Limit:       req.Limit,
	}

	return res, nil
}

func (c *ContentRepo) GetDestinationsById(ctx context.Context, req *pb.GetDestinationsByIdReq) (*pb.GetDestinationsByIdRes, error) {

	query := `
        SELECT id, name, country, description, best_time_to_visit, average_cost_per_day, currency, language
        FROM destinations
        WHERE id = $1
    `

	var destination pb.GetDestinationsByIdRes
	err := c.DB.QueryRowContext(ctx, query, req.Id).Scan(
		&destination.Id,
		&destination.Name,
		&destination.Country,
		&destination.Description,
		&destination.BestTimeToVisit,
		&destination.AverageCostPerDay,
		&destination.Currency,
		&destination.Language,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch destination by ID: %v", err)
	}

	return &destination, nil
}

func (c *ContentRepo) SendMessage(ctx context.Context, req *pb.SendMessageReq) (*pb.SendMessageRes, error) {
	// Insert the message into the database
	query := `
        INSERT INTO messages (id, sender_id, recipient_id, content, created_at)
        VALUES (gen_random_uuid(), $1, $2, $3, CURRENT_TIMESTAMP)
        RETURNING id, sender_id, recipient_id, content
    `

	var message pb.SendMessageRes
	err := c.DB.QueryRowContext(ctx, query, req.UserId, req.RecipientId, req.Content).Scan(
		&message.Id,
		&message.UserId,
		&message.RecipientId,
		&message.Content,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %v", err)
	}

	return &message, nil
}

func (c *ContentRepo) GetMessages(ctx context.Context, req *pb.GetMessagesReq) (*pb.GetMessagesRes, error) {
	// Query to fetch messages
	query := `
        SELECT m.id, m.content, 
               s.user_id AS sender_user_id, s.username AS sender_username, s.full_name AS sender_full_name,
               r.user_id AS recipient_user_id, r.username AS recipient_username, r.full_name AS recipient_full_name
        FROM messages m
        INNER JOIN users s ON m.sender_id = s.id
        INNER JOIN users r ON m.recipient_id = r.id
        ORDER BY m.created_at DESC
        LIMIT $1 OFFSET $2
    `

	// Execute the query
	rows, err := c.DB.QueryContext(ctx, query, req.Limit, req.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %v", err)
	}
	defer rows.Close()

	var messages []*pb.Messages
	for rows.Next() {
		var message pb.Messages
		var sender, recipient pb.Author

		// Scan row into message and author structs
		if err := rows.Scan(
			&message.Id, &message.Content,
			&sender.UserId, &sender.Username, &sender.FullName,
			&recipient.UserId, &recipient.Username, &recipient.FullName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %v", err)
		}

		message.Sender = &sender
		message.Recipient = &recipient
		messages = append(messages, &message)
	}

	// Count total number of messages
	countQuery := `SELECT COUNT(*) FROM messages`
	var total int64
	if err := c.DB.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to fetch total message count: %v", err)
	}

	// Prepare response
	res := &pb.GetMessagesRes{
		Messages: messages,
		Total:    total,
		Offset:   req.Offset,
		Limit:    req.Limit,
	}

	return res, nil
}

func (c *ContentRepo) CreateTips(ctx context.Context, req *pb.CreateTipsReq) (*pb.CreateTipsRes, error) {
	// Extract user_id from the request
	authorID := req.UserId

	// SQL query to insert a new tip into the database
	query := `
        INSERT INTO travel_tips (title, content, category, author_id, created_at)
        VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
        RETURNING id
    `

	var id string
	// Execute the query and retrieve the inserted tip's ID
	err := c.DB.QueryRowContext(ctx, query, req.Title, req.Content, req.Category, authorID).Scan(&id)
	if err != nil {
		return nil, err
	}

	// Prepare the response message
	res := &pb.CreateTipsRes{
		Id:       id,
		Title:    req.Title,
		Content:  req.Content,
		Category: req.Category,
		AuthorId: authorID,
	}

	return res, nil
}

func (c *ContentRepo) GetTips(ctx context.Context, req *pb.GetTipsReq) (*pb.GetTipsRes, error) {
	// Query to retrieve tips with pagination
	query := `
        SELECT tt.id, tt.title, tt.category, u.id AS user_id, u.username, u.full_name
        FROM travel_tips tt
        JOIN users u ON tt.author_id = u.id
        WHERE tt.deleted_at = 0
    `

	// Prepare parameters and conditions based on the request
	queryParams := make([]interface{}, 0)
	conditions := make([]string, 0)

	// Filter by category if provided
	if req.Category != "" {
		conditions = append(conditions, "tt.category = $1")
		queryParams = append(queryParams, req.Category)
	}

	// Offset and limit for pagination
	queryParams = append(queryParams, req.Limit, req.Offset)
	query += " ORDER BY tt.created_at DESC OFFSET $2 LIMIT $3"

	// Execute the query
	rows, err := c.DB.QueryContext(ctx, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tips []*pb.Tips
	for rows.Next() {
		var tipID, title, category, userID, username, fullName string
		if err := rows.Scan(&tipID, &title, &category, &userID, &username, &fullName); err != nil {
			return nil, err
		}

		author := &pb.Author{
			UserId:   userID,
			Username: username,
			FullName: fullName,
		}

		tip := &pb.Tips{
			Id:       tipID,
			Title:    title,
			Category: category,
			Author:   author,
		}

		tips = append(tips, tip)
	}

	// Count total number of tips
	countQuery := `
        SELECT COUNT(*) AS total
        FROM travel_tips tt
        WHERE tt.deleted_at = 0
    `
	if req.Category != "" {
		countQuery += " AND tt.category = $1"
	}

	var total int64
	err = c.DB.QueryRowContext(ctx, countQuery, queryParams...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Prepare response
	res := &pb.GetTipsRes{
		Tips:   tips,
		Total:  total,
		Offset: req.Offset,
		Limit:  req.Limit,
	}

	return res, nil
}

func (c *ContentRepo) GetUserStat(ctx context.Context, req *pb.GetUserStatReq) (*pb.GetUserStatRes, error) {
	// Initialize response structure
	res := &pb.GetUserStatRes{
		UserId: req.UserId,
	}

	// Query to get total stories by user
	storyQuery := `
        SELECT COUNT(*) AS total_stories
        FROM stories
        WHERE author_id = $1 AND deleted_at = 0
    `
	var totalStories int64
	err := c.DB.QueryRowContext(ctx, storyQuery, req.UserId).Scan(&totalStories)
	if err != nil {
		return nil, err
	}
	res.TotalStories = fmt.Sprintf("%d", totalStories)

	// Query to get total itineraries by user
	itineraryQuery := `
        SELECT COUNT(*) AS total_itineraries
        FROM itineraries
        WHERE author_id = $1 AND deleted_at = 0
    `
	var totalItineraries int64
	err = c.DB.QueryRowContext(ctx, itineraryQuery, req.UserId).Scan(&totalItineraries)
	if err != nil {
		return nil, err
	}
	res.TotalItineraries = fmt.Sprintf("%d", totalItineraries)

	// Query to get total countries visited by user
	countriesQuery := `
        SELECT countries_visited
        FROM users
        WHERE id = $1
    `
	var totalCountries int64
	err = c.DB.QueryRowContext(ctx, countriesQuery, req.UserId).Scan(&totalCountries)
	if err != nil {
		return nil, err
	}
	res.TotalCountriesVisited = fmt.Sprintf("%d", totalCountries)

	// Query to get total likes received by user's stories and itineraries
	likesQuery := `
        SELECT SUM(likes_count) AS total_likes_received
        FROM (
            SELECT likes_count
            FROM stories
            WHERE author_id = $1 AND deleted_at = 0
            UNION ALL
            SELECT likes_count
            FROM itineraries
            WHERE author_id = $1 AND deleted_at = 0
        ) AS combined_likes
    `
	var totalLikesReceived sql.NullInt64
	err = c.DB.QueryRowContext(ctx, likesQuery, req.UserId).Scan(&totalLikesReceived)
	if err != nil {
		return nil, err
	}
	if totalLikesReceived.Valid {
		res.TotalLikesReceived = fmt.Sprintf("%d", totalLikesReceived.Int64)
	} else {
		res.TotalLikesReceived = "0"
	}

	// Query to get total comments received by user's stories and itineraries
	commentsQuery := `
        SELECT SUM(comments_count) AS total_comments_received
        FROM (
            SELECT comments_count
            FROM stories
            WHERE author_id = $1 AND deleted_at = 0
            UNION ALL
            SELECT comments_count
            FROM itineraries
            WHERE author_id = $1 AND deleted_at = 0
        ) AS combined_comments
    `
	var totalCommentsReceived sql.NullInt64
	err = c.DB.QueryRowContext(ctx, commentsQuery, req.UserId).Scan(&totalCommentsReceived)
	if err != nil {
		return nil, err
	}
	if totalCommentsReceived.Valid {
		res.TotalCommentsReceived = fmt.Sprintf("%d", totalCommentsReceived.Int64)
	} else {
		res.TotalCommentsReceived = "0"
	}

	// Query to get the most popular story by likes
	popularStoryQuery := `
        SELECT id, title, likes_count
        FROM stories
        WHERE author_id = $1 AND deleted_at = 0
        ORDER BY likes_count DESC
        LIMIT 1
    `
	var popularStory pb.PopularStory
	err = c.DB.QueryRowContext(ctx, popularStoryQuery, req.UserId).Scan(&popularStory.Id, &popularStory.Title, &popularStory.LikesCount)
	if err != nil {
		if err == sql.ErrNoRows {
			popularStory.Id = ""
			popularStory.Title = "No popular story found"
			popularStory.LikesCount = "0"
		} else {
			return nil, err
		}
	}
	res.MostPopularStory = &popularStory

	// Query to get the most popular itinerary by likes
	popularItineraryQuery := `
        SELECT id, title, likes_count
        FROM itineraries
        WHERE author_id = $1 AND deleted_at = 0
        ORDER BY likes_count DESC
        LIMIT 1
    `
	var popularItinerary pb.PopularItinerary
	err = c.DB.QueryRowContext(ctx, popularItineraryQuery, req.UserId).Scan(&popularItinerary.Id, &popularItinerary.Title, &popularItinerary.LikesCount)
	if err != nil {
		if err == sql.ErrNoRows {
			popularItinerary.Id = ""
			popularItinerary.Title = "No popular itinerary found"
			popularItinerary.LikesCount = "0"
		} else {
			return nil, err
		}
	}
	res.MostPopularItinerary = &popularItinerary

	return res, nil
}
