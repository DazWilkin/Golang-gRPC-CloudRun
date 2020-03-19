package main

import (
	"context"

	pb "github.com/DazWilkin/Golang-gRPC-CloudRun/protos"

	"go.opencensus.io/trace"

	"google.golang.org/grpc"
)

type Client struct {
	client pb.CalculatorClient
}

func NewClient(conn *grpc.ClientConn) *Client {
	return &Client{
		client: pb.NewCalculatorClient(conn),
	}
}

func (c *Client) Calculate(ctx context.Context, r *pb.BinaryOperation) (*pb.CalculationResult, error) {
	ctx, span := trace.StartSpan(ctx, "Calculate")
	defer span.End()
	defer latencyTimer(ctx, "Calculate")()
	return c.client.Calculate(ctx, r)
}
