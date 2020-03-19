package main

import (
	"context"
	"fmt"
	"log"

	pb "github.com/DazWilkin/Golang-gRPC-CloudRun/protos"

	"go.opencensus.io/trace"
)

// Prove that Server implements pb.CalculatorServer by instantiating a Server
var _ pb.CalculatorServer = (*Server)(nil)

// Server is a struct implements the pb.CalculatorServer
type Server struct {
}

func NewServer() *Server {
	return &Server{}
}

// Calculate performs an operation on operands defined by pb.BinaryOperation returning pb.CalculationResult
func (s *Server) Calculate(ctx context.Context, r *pb.BinaryOperation) (*pb.CalculationResult, error) {
	log.Println("[server:Calculate] Started")
	if ctx.Err() == context.Canceled {
		return &pb.CalculationResult{}, fmt.Errorf("client cancelled: abandoning")
	}
	// TODO(dazwilkin) Low-value but required to capture a trace at all
	ctx, span := trace.StartSpan(ctx, "Calculate")
	defer span.End()

	switch r.GetOperation() {
	case pb.Operation_ADD:
		return &pb.CalculationResult{
			Result: r.GetFirstOperand() + r.GetSecondOperand(),
		}, nil
	case pb.Operation_SUBTRACT:
		return &pb.CalculationResult{
			Result: r.GetFirstOperand() - r.GetSecondOperand(),
		}, nil
	default:
		return &pb.CalculationResult{}, fmt.Errorf("undefined operation")
	}

}
