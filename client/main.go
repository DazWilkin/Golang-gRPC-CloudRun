package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"

	"contrib.go.opencensus.io/exporter/ocagent"
	pb "github.com/DazWilkin/Golang-gRPC-CloudRun/protos"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
	"go.opencensus.io/zpages"
	"google.golang.org/grpc"
)

const (
	serviceName = "grpc-cloudrun-server"
)

var (
	grpcEndpoint = flag.String("gprc_endpoint", ":50051", "The gRPC Endpoint of the Server")
	cnssEndpoint = flag.String("cnss_endpoint", "", "Endpoint of the OpenCensus Agent")
	zpgzEndpoint = flag.String("zpgz_endpoint", ":9997", "Endpoint of the zPages exporter")
)

var (
	mLatencyMs *stats.Float64Measure
	keyClient  tag.Key
	keyMethod  tag.Key
)

func zPages(endpoint string) {
	zPagesMux := http.NewServeMux()
	zpages.Handle(zPagesMux, "/debug")
	zpgzServer := &http.Server{
		Addr:    *zpgzEndpoint,
		Handler: zPagesMux,
	}
	listen, err := net.Listen("tcp", *zpgzEndpoint)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Starting zPages Listener [%s]\n", *zpgzEndpoint)
	log.Printf("zPages RPC Stats %s/deubg/rpcz\n", *zpgzEndpoint)
	log.Printf("zPages Trace Spans %s/debug/tracez\n", *zpgzEndpoint)
	log.Fatal(zpgzServer.Serve(listen))
}

func main() {
	log.Printf("[main] Starting: %s", serviceName)
	defer func() {
		log.Printf("[main] Stopping: %s", serviceName)
	}()

	flag.Parse()
	if *grpcEndpoint == "" {
		log.Fatal("[main] unable to start client without gRPC endpoint to server")
	}

	if err := view.Register(ocgrpc.DefaultClientViews...); err != nil {
		log.Fatalf("Failed to register ocgrpc client views: %v", err)
	}

	go zPages(*zpgzEndpoint)

	log.Printf("Starting: OpenCensus Agent exporter [%s]\n", *cnssEndpoint)
	oc, err := ocagent.NewExporter(
		ocagent.WithAddress(*cnssEndpoint),
		ocagent.WithInsecure(),
		ocagent.WithReconnectionPeriod(10*time.Second),
		ocagent.WithServiceName(serviceName),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer oc.Stop()

	view.RegisterExporter(oc)
	trace.RegisterExporter(oc)

	// TODO(dazwilkin) Reduce Trace Sample Frequency
	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.AlwaysSample(),
	})

	// For want of something more interesting, a simple latency measurement
	mLatencyMs = stats.Float64("latency", "The latency in milliseconds.", "ms")
	keyClient, _ = tag.NewKey("client")
	keyMethod, _ = tag.NewKey("method")
	views := []*view.View{
		{
			Name:        fmt.Sprintf("%s/latency", serviceName),
			Description: "The latencies of the method",
			Measure:     mLatencyMs,
			Aggregation: view.Distribution(10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000),
			TagKeys:     []tag.Key{keyClient, keyMethod},
		},
	}
	if err := view.Register(views...); err != nil {
		log.Fatal(err)
	}

	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithStatsHandler(new(ocgrpc.ClientHandler)),
	}
	log.Printf("Connecting to gRPC Service [%s]", *grpcEndpoint)
	conn, err := grpc.Dial(*grpcEndpoint, opts...)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := NewClient(conn)
	ctx := context.Background()

	// Loop indefinitely
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	for {
		o1 := r1.Float32()
		o2 := r1.Float32()
		rqst := &pb.BinaryOperation{
			FirstOperand:  o1,
			SecondOperand: o2,
			Operation:     pb.Operation_ADD,
		}
		resp, err := client.Calculate(ctx, rqst)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("[main] %0.3f+%0.3f=%0.3f", o1, o2, resp.GetResult())
		time.Sleep(15 * time.Second)
	}

}
