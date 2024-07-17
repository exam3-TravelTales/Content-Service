package service

import (
	pb "content/genproto/content"
	"content/logger"
	"content/storage/postgres"
	"content/storage/redis"
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

func (u *ContentService) GetDestinations(ctx context.Context, req *pb.GetDestinationsReq) (*pb.GetDestinationsRes, error) {
	u.Log.Info("GetDestinations rpc method started")
	res, err := u.Repo.GetDestinations(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("GetDestinations rpc method finished")
	return res, nil
}
func (u *ContentService) GetDestinationsById(ctx context.Context, req *pb.GetDestinationsByIdReq) (*pb.GetDestinationsByIdRes, error) {
	u.Log.Info("GetDestinationsById rpc method started")
	res, err := u.Repo.GetDestinationsById(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
	}
	u.Log.Info("GetDestinationsById rpc method finished")
	return res, nil
}
func (u *ContentService) SendMessage(ctx context.Context, req *pb.SendMessageReq) (*pb.SendMessageRes, error) {
	u.Log.Info("SendMessage rpc method started")
	res, err := u.Repo.SendMessage(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("SendMessage rpc method finished")
	return res, nil
}
func (u *ContentService) GetMessages(ctx context.Context, req *pb.GetMessagesReq) (*pb.GetMessagesRes, error) {
	u.Log.Info("GetMessages rpc method started")
	res, err := u.Repo.GetMessages(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("GetMessages rpc method finished")
	return res, nil
}
func (u *ContentService) CreateTips(ctx context.Context, req *pb.CreateTipsReq) (*pb.CreateTipsRes, error) {
	u.Log.Info("CreateTips rpc method started")
	res, err := u.Repo.CreateTips(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("CreateTips rpc method finished")
	return res, nil
}
func (u *ContentService) GetTips(ctx context.Context, req *pb.GetTipsReq) (*pb.GetTipsRes, error) {
	u.Log.Info("GetTips rpc method started")
	res, err := u.Repo.GetTips(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("GetTips rpc method finished")
	return res, nil
}
func (u *ContentService) GetUserStat(ctx context.Context, req *pb.GetUserStatReq) (*pb.GetUserStatRes, error) {
	u.Log.Info("GetUserStat rpc method started")
	res, err := u.Repo.GetUserStat(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("GetUserStat rpc method finished")
	return res, nil
}

func (u *ContentService) TopDestinations(ctx context.Context, req *pb.Void) (*pb.Answer, error) {
	u.Log.Info("TopDestinations rpc method started")
	res, err := redis.SaveTopDestinations(ctx, u.Repo)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("TopDestinations rpc method finished")
	return res, nil
}
