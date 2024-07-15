package service

import (
	pb "content/genproto/content"
	"content/logger"
	"content/storage/postgres"
	"context"
	"database/sql"
	"log/slog"
)

type ContentService struct {
	pb.UnimplementedContentServer
	Repo *postgres.ContentRepo
	Log  *slog.Logger
}

func NewContentService(db *sql.DB) *ContentService {
	return &ContentService{
		Repo: postgres.NewContentRepository(db),
		Log:  logger.NewLogger(),
	}
}
func (u *ContentService) CreateStories(ctx context.Context, req *pb.CreateStoriesRequest) (*pb.CreateStoriesResponse, error) {
	u.Log.Info("CreateStories rpc method started")
	res, err := u.Repo.CreateStory(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("CreateStories rpc method finished")
	return res, nil
}
func (u *ContentService) UpdateStories(ctx context.Context, req *pb.UpdateStoriesReq) (*pb.UpdateStoriesRes, error) {
	u.Log.Info("UpdateStories rpc method started")
	res, err := u.Repo.UpdateStory(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("UpdateStories rpc method finished")
	return res, nil
}

func (u *ContentService) DeleteStories(ctx context.Context, req *pb.StoryId) (*pb.Void, error) {
	u.Log.Info("DeleteStories rpc method started")
	err := u.Repo.DeleteStory(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("DeleteStories rpc method finished")
	return &pb.Void{}, nil
}

func (u *ContentService) GetAllStories(ctx context.Context, req *pb.GetAllStoriesReq) (*pb.GetAllStoriesRes, error) {
	u.Log.Info("GetAllStories rpc method started")
	res, err := u.Repo.GetAllStory(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("GetAllStories rpc method finished")
	return res, nil
}

func (u *ContentService) GetStory(ctx context.Context, req *pb.StoryId) (*pb.GetStoryRes, error) {
	u.Log.Info("GetStory rpc method started")
	res, err := u.Repo.GetStoryById(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("GetStory rpc method finished")
	return res, nil
}

func (u *ContentService) CommentStory(ctx context.Context, req *pb.CommentStoryReq) (*pb.CommentStoryRes, error) {
	u.Log.Info("CommentStory rpc method started")
	res, err := u.Repo.CommentToStory(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("CommentStory rpc method finished")
	return res, nil
}

func (u *ContentService) GetCommentsOfStory(ctx context.Context, req *pb.StoryId) (*pb.GetCommentsOfStoryRes, error) {
	u.Log.Info("GetCommentsOfStory rpc method started")
	res, err := u.Repo.GetCommentsOfStory(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("GetCommentsOfStory rpc method finished")
	return res, nil
}

func (u *ContentService) Like(ctx context.Context, req *pb.LikeReq) (*pb.LikeRes, error) {
	u.Log.Info("Like rpc method started")
	res, err := u.Repo.Like(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("Like rpc method finished")
	return res, nil
}

func (u *ContentService) Itineraries(ctx context.Context, req *pb.ItinerariesReq) (*pb.ItinerariesRes, error) {
	u.Log.Info("Itineraries rpc method started")
	res, err := u.Repo.Itineraries(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("Itineraries rpc method finished")
	return res, nil
}

func (u *ContentService) UpdateItineraries(ctx context.Context, req *pb.UpdateItinerariesReq) (*pb.ItinerariesRes, error) {
	u.Log.Info("UpdateItineraries rpc method started")
	res, err := u.Repo.UpdateItineraries(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("UpdateItineraries rpc method finished")
	return res, nil
}
