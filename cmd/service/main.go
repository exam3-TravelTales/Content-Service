package main

import (
	"content/genproto/content"

	"content/service"
	"content/storage/postgres"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
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
	Service := service.NewContentService(db)
	server := grpc.NewServer()
	content.RegisterContentServer(server, Service)
	log.Printf("server listening at %v", lis.Addr())
	err = server.Serve(lis)
	if err != nil {
		log.Fatalf("error while serving: %v", err)
	}
}
