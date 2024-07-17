package postgres

import (
	pb "content/genproto/story"
	"context"
	"database/sql"
)

type StoryRepo struct {
	DB *sql.DB
}

func NewStoryRepository(db *sql.DB) *StoryRepo {
	return &StoryRepo{DB: db}
}

func (c *StoryRepo) CreateStory(ctx context.Context, request *pb.CreateStoriesRequest) (*pb.CreateStoriesResponse, error) {
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

func (c *StoryRepo) UpdateStory(ctx context.Context, request *pb.UpdateStoriesReq) (*pb.UpdateStoriesRes, error) {
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

func (c *StoryRepo) DeleteStory(ctx context.Context, id *pb.StoryId) error {

	query := `
        UPDATE stories
        SET deleted_at = date_part('epoch', current_timestamp)::INT
        WHERE id = $1 and deleted_at = 0
    `

	_, err := c.DB.ExecContext(ctx, query, id.Id)
	if err != nil {
		return err
	}

	return nil
}

func (c *StoryRepo) GetAllStory(ctx context.Context, request *pb.GetAllStoriesReq) (*pb.GetAllStoriesRes, error) {
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

func (c *StoryRepo) GetStoryById(ctx context.Context, id *pb.StoryId) (*pb.GetStoryRes, error) {

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

	story.Author = &author

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

	story.Tags = tags

	return &story, nil
}

func (c *StoryRepo) CommentToStory(ctx context.Context, req *pb.CommentStoryReq) (*pb.CommentStoryRes, error) {

	query := `
        INSERT INTO comments (id, content, author_id, story_id, created_at)
        VALUES (gen_random_uuid(), $1, $2, $3, CURRENT_TIMESTAMP)
        RETURNING id, content, author_id, story_id, created_at
    `

	var comment pb.CommentStoryRes

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
	updatequery := `
	UPDATE stories SET comments_count = comments_count + 1 WHERE id = $1
	`
	_, err = c.DB.ExecContext(ctx, updatequery, req.StoryId)
	if err != nil {
		return nil, err
	}

	return &comment, nil
}

func (c *StoryRepo) GetCommentsOfStory(ctx context.Context, req *pb.GetCommentsOfStoryReq) (*pb.GetCommentsOfStoryRes, error) {

	res := &pb.GetCommentsOfStoryRes{
		Offset: req.Offset,
		Limit:  req.Limit,
	}

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

func (c *StoryRepo) Like(ctx context.Context, req *pb.LikeReq) (*pb.LikeRes, error) {

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

	res := &pb.LikeRes{
		UserId:  req.UserId,
		StoryId: req.StoryId,
		LikedAt: likedAt,
	}

	updatequery := `
	UPDATE stories SET likes_count = likes_count + 1 WHERE id = $1
	`
	_, err = c.DB.ExecContext(ctx, updatequery, req.StoryId)
	if err != nil {
		return nil, err
	}

	return res, nil
}
