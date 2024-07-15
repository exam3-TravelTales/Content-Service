package main

// import (
// 	"auth/genproto/users"
// 	"auth/service"
// 	"auth/storage/postgres"
// 	"fmt"
// 	"google.golang.org/grpc"
// 	"log"
// 	"net"
// )

// func main() {
// 	db, err := postgres.ConnectDB()
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer db.Close()
// 	fmt.Println("Starting server...")
// 	lis, err := net.Listen("tcp", ":50051")
// 	if err != nil {
// 		log.Fatalf("error while listening: %v", err)
// 	}
// 	defer lis.Close()
// 	userService := service.NewUserService(db)
// 	server := grpc.NewServer()
// 	users.RegisterUserServer(server, userService)
// 	log.Printf("server listening at %v", lis.Addr())
// 	err = server.Serve(lis)
// 	if err != nil {
// 		log.Fatalf("error while serving: %v", err)
// 	}
// }
