package postgres

import (
	pb "content/genproto/content"
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type ContentRepo struct {
	DB *sql.DB
}

func NewContentRepository(db *sql.DB) *ContentRepo {
	return &ContentRepo{DB: db}
}

func (c *ContentRepo) GetDestinations(ctx context.Context, req *pb.GetDestinationsReq) (*pb.GetDestinationsRes, error) {

	query := `
        SELECT id, name, country, description, currency
        FROM destinations
        WHERE ($1 = '' OR name ILIKE '%' || $1 || '%')
        ORDER BY name
        LIMIT $2 OFFSET $3
    `

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

	query := `
	SELECT m.id, m.content, 
	s.id AS sender_user_id, s.username AS sender_username, s.full_name AS sender_full_name,
	r.id AS recipient_user_id, r.username AS recipient_username, r.full_name AS recipient_full_name
FROM messages m
INNER JOIN users s ON m.sender_id = s.id
INNER JOIN users r ON m.recipient_id = r.id
ORDER BY m.created_at DESC
LIMIT $1 OFFSET $2

    `

	rows, err := c.DB.QueryContext(ctx, query, req.Limit, req.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %v", err)
	}
	defer rows.Close()

	var messages []*pb.Messages
	for rows.Next() {
		var message pb.Messages
		var sender, recipient pb.Author

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

	countQuery := `SELECT COUNT(*) FROM messages`
	var total int64
	if err := c.DB.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to fetch total message count: %v", err)
	}

	res := &pb.GetMessagesRes{
		Messages: messages,
		Total:    total,
		Offset:   req.Offset,
		Limit:    req.Limit,
	}

	return res, nil
}

func (c *ContentRepo) CreateTips(ctx context.Context, req *pb.CreateTipsReq) (*pb.CreateTipsRes, error) {

	authorID := req.UserId

	query := `
        INSERT INTO travel_tips (title, content, category, author_id, created_at)
        VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
        RETURNING id
    `

	var id string

	err := c.DB.QueryRowContext(ctx, query, req.Title, req.Content, req.Category, authorID).Scan(&id)
	if err != nil {
		return nil, err
	}

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
	query := `
        SELECT tt.id, tt.title, tt.category, u.id AS user_id, u.username, u.full_name
        FROM travel_tips tt
        JOIN users u ON tt.author_id = u.id
    `

	queryParams := make([]interface{}, 0)
	conditions := make([]string, 0)

	n := 1
	if req.Category != "" {
		conditions = append(conditions, fmt.Sprintf("tt.category = $%d", n))
		queryParams = append(queryParams, req.Category)
		n++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	queryParams = append(queryParams, req.Offset, req.Limit)
	query += fmt.Sprintf(" ORDER BY tt.created_at DESC OFFSET $%d LIMIT $%d", n, n+1)

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

	countQuery := `
        SELECT COUNT(*) AS total
        FROM travel_tips tt
    `
	if len(conditions) > 0 {
		countQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	countQueryParams := make([]interface{}, 0)
	if req.Category != "" {
		countQueryParams = append(countQueryParams, req.Category)
	}

	err = c.DB.QueryRowContext(ctx, countQuery, countQueryParams...).Scan(&total)
	if err != nil {
		return nil, err
	}

	res := &pb.GetTipsRes{
		Tips:   tips,
		Total:  total,
		Offset: req.Offset,
		Limit:  req.Limit,
	}

	return res, nil
}

func (c *ContentRepo) GetUserStat(ctx context.Context, req *pb.GetUserStatReq) (*pb.GetUserStatRes, error) {

	res := &pb.GetUserStatRes{
		UserId: req.UserId,
	}

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

func (c *ContentRepo) GetTopDestinations(ctx context.Context) (*pb.Answer, error) {
	query := `
        SELECT country, description, best_time_to_visit, popularity_score
        FROM destinations
        ORDER BY popularity_score DESC
        LIMIT 10
    `

	rows, err := c.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topDestinations []*pb.TopDestinationsRes
	for rows.Next() {
		var destination pb.TopDestinationsRes
		if err := rows.Scan(
			&destination.Country,
			&destination.Description,
			&destination.BestTimeToVisit,
			&destination.PopularityScore,
		); err != nil {
			return nil, err
		}
		topDestinations = append(topDestinations, &destination)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	res := &pb.Answer{
		Topdestinations: topDestinations,
	}

	return res, nil
}
