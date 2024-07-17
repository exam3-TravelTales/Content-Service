package postgres

import (
	pb "content/genproto/itineraries"
	"context"
	"database/sql"
	"fmt"
)

type ItinerariesRepo struct {
	DB *sql.DB
}

func NewItinerariesRepository(db *sql.DB) *ItinerariesRepo {
	return &ItinerariesRepo{DB: db}
}

func (c *ItinerariesRepo) Itineraries(ctx context.Context, req *pb.ItinerariesReq) (*pb.ItinerariesRes, error) {

	tx, err := c.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

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

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &itinerary, nil
}

func (c *ItinerariesRepo) UpdateItineraries(ctx context.Context, req *pb.UpdateItinerariesReq) (*pb.ItinerariesRes, error) {

	tx, err := c.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

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

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &updatedItinerary, nil
}

func (c *ItinerariesRepo) DeleteItineraries(ctx context.Context, req *pb.StoryId) error {
	tx, err := c.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

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

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (c *ItinerariesRepo) GetItineraries(ctx context.Context, req *pb.GetItinerariesReq) (*pb.GetItinerariesRes, error) {

	var total int64
	totalQuery := `SELECT COUNT(*) FROM itineraries WHERE deleted_at = 0`
	err := c.DB.QueryRowContext(ctx, totalQuery).Scan(&total)
	if err != nil {
		return nil, err
	}

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

func (c *ItinerariesRepo) GetItinerariesById(ctx context.Context, req *pb.StoryId) (*pb.GetItinerariesByIdRes, error) {

	itinerary := pb.GetItinerariesByIdRes{
		Author: &pb.Author{},
	}

	itineraryQuery := `
        SELECT i.id, i.title, i.description, i.start_date, i.end_date, u.id, u.username, u.full_name
        FROM itineraries i
        JOIN users u ON i.author_id = u.id
        WHERE i.id = $1 AND i.deleted_at = 0
    `
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
        SELECT name, start_date, end_date
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
            WHERE destination_id in (
                SELECT id
                FROM itinerary_destinations
                WHERE name = $1
            )
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

		if err = activityRows.Err(); err != nil {
			return nil, err
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

func (c *ItinerariesRepo) CommentItineraries(ctx context.Context, req *pb.CommentItinerariesReq) (*pb.CommentItinerariesRes, error) {
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
