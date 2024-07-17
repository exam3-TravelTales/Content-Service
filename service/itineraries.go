package service

import (
	pb "content/genproto/itineraries"
	"content/logger"
	"content/storage/postgres"
	"context"
	"database/sql"
	"log/slog"
)

type ItinerariesService struct {
	pb.UnimplementedItinerariesServer
	Repo *postgres.ItinerariesRepo
	Log  *slog.Logger
}

func NewItinerariesService(db *sql.DB) *ItinerariesService {
	return &ItinerariesService{
		Repo: postgres.NewItinerariesRepository(db),
		Log:  logger.NewLogger(),
	}
}

func (u *ItinerariesService) Itineraries(ctx context.Context, req *pb.ItinerariesReq) (*pb.ItinerariesRes, error) {
	u.Log.Info("Itineraries rpc method started")
	res, err := u.Repo.Itineraries(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("Itineraries rpc method finished")
	return res, nil
}

func (u *ItinerariesService) UpdateItineraries(ctx context.Context, req *pb.UpdateItinerariesReq) (*pb.ItinerariesRes, error) {
	u.Log.Info("UpdateItineraries rpc method started")
	res, err := u.Repo.UpdateItineraries(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("UpdateItineraries rpc method finished")
	return res, nil
}

func (u *ItinerariesService) DeleteItineraries(ctx context.Context, req *pb.StoryId) (*pb.Void, error) {
	u.Log.Info("DeleteItineraries rpc method started")
	err := u.Repo.DeleteItineraries(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("DeleteItineraries rpc method finished")
	return &pb.Void{}, nil
}
func (u *ItinerariesService) GetItineraries(ctx context.Context, req *pb.GetItinerariesReq) (*pb.GetItinerariesRes, error) {
	u.Log.Info("GetItineraries rpc method started")
	res, err := u.Repo.GetItineraries(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("GetItineraries rpc method finished")
	return res, nil
}
func (u *ItinerariesService) GetItinerariesById(ctx context.Context, req *pb.StoryId) (*pb.GetItinerariesByIdRes, error) {
	u.Log.Info("GetItinerariesById rpc method started")
	res, err := u.Repo.GetItinerariesById(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("GetItinerariesById rpc method finished")
	return res, nil
}
func (u *ItinerariesService) CommentItineraries(ctx context.Context, req *pb.CommentItinerariesReq) (*pb.CommentItinerariesRes, error) {
	u.Log.Info("CommentItineraries rpc method started")
	res, err := u.Repo.CommentItineraries(ctx, req)
	if err != nil {
		u.Log.Error(err.Error())
		return nil, err
	}
	u.Log.Info("CommentItineraries rpc method finished")
	return res, nil
}
