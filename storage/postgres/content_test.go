package postgres

import (
	pb "content/genproto/content"
	"context"
	"fmt"
	"testing"
)

func TestCreateStory(t *testing.T) {
	db, err := ConnectDB()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	con := NewContentRepository(db)
	req, err := con.CreateStory(context.Background(), &pb.CreateStoriesRequest{
		Title:    "new",
		Content:  "new",
		Location: "new",
		Tags:     []string{"new1", "new2", "new3"},
		UserId:   "d1e7d01d-443e-4580-a330-49fa8ebe168f",
	})
	if err != nil {
		fmt.Println(err)
	}
	res := pb.CreateStoriesResponse{
		Id:        "8bbe7c12-a359-41d0-b8e8-f6c825a0e838",
		Title:     "new",
		Content:   "new",
		Location:  "new",
		Tags:      []string{"new1", "new2", "new3"},
		AuthorId:  "d1e7d01d-443e-4580-a330-49fa8ebe168f",
		CreatedAt: "2024-07-15T12:31:04.038994+05:00",
	}

	fmt.Println(req.CreatedAt, req.Id)

	if req.Title != res.Title || req.Content != res.Content || req.Location != res.Location || req.AuthorId != res.AuthorId {
		t.Errorf("CreateStoriesRequest returned %+v, want %+v", req, &res)
	}
}
