package postgres

import (
	pb "content/genproto/content"
	"context"
	"database/sql"
)

type ContentRepo struct {
	DB *sql.DB
}

func NewContentRepository(db *sql.DB) *ContentRepo {
	return &ContentRepo{DB: db}
}

func (c *ContentRepo) CreateStory(ctx context.Context, request *pb.CreateStoriesRequest) (*pb.CreateStoriesResponse, error) {
	// Start a new transaction
	tx, err := c.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Insert the story into the stories table
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

	// Insert the tags into the story_tags table
	tagQuery := `INSERT INTO story_tags (story_id, tag) VALUES ($1, $2)`
	for _, tag := range request.Tags {
		_, err := tx.ExecContext(ctx, tagQuery, createdStory.Id, tag)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Set the tags in the response
	createdStory.Tags = request.Tags

	return &createdStory, nil
}
