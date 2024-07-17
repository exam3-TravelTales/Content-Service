package main

import (
	"content/genproto/content"
	"content/genproto/itineraries"
	"content/genproto/story"

	"content/service"
	"content/storage/postgres"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
)

func main() {
	db, err := postgres.ConnectDB()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	fmt.Println("Starting server...")
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("error while listening: %v", err)
	}
	defer lis.Close()

	Servicecn := service.NewContentService(db)
	Servicest := service.NewStoryService(db)
	Serviceit := service.NewItinerariesService(db)

	server := grpc.NewServer()

	content.RegisterContentServer(server, Servicecn)
	story.RegisterStoryServer(server, Servicest)
	itineraries.RegisterItinerariesServer(server, Serviceit)

	log.Printf("server listening at %v", lis.Addr())
	err = server.Serve(lis)
	if err != nil {
		log.Fatalf("error while serving: %v", err)
	}
}
