package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	pb "github.com/DazWilkin/Golang-gRPC-CloudRun/protos"

	"google.golang.org/grpc"
)

const (
	serviceName = "grpc-cloudrun-server"
)

func main() {
	log.Printf("Starting: %s", serviceName)
	defer func() {
		log.Printf("Stopping:%s", serviceName)
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	grpcEndpoint := fmt.Sprintf(":%s", port)
	log.Printf("gRPC endpoint [%s]", grpcEndpoint)

	grpcServer := grpc.NewServer()
	pb.RegisterCalculatorServer(grpcServer, NewServer())

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	// gRPC Server
	wg.Add(1)
	go func() {
		defer wg.Done()
		listen, err := net.Listen("tcp", grpcEndpoint)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Starting: gRPC Listener [%s]\n", grpcEndpoint)
		log.Fatal(grpcServer.Serve(listen))
	}()
	wg.Wait()
}
