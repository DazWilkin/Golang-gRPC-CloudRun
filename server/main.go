package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"contrib.go.opencensus.io/exporter/ocagent"

	pb "github.com/DazWilkin/Golang-gRPC-CloudRun/protos"

	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.opencensus.io/zpages"

	"google.golang.org/grpc"
)

const (
	serviceName = "grpc-cloudrun-server"
)

var (
	grpcEndpoint = flag.String("grpc_endpoint", ":50051", "The gRPC endpoint to listen on.")
	// httpEndpoint = flag.String("http_endpoint", "", "The HTTP endpoint to listen to.")
	cnssEndpoint = flag.String("cnss_endpoint", "", "The gRPC endpoint of the OpenCensus Agent.")
	zpgzEndpoint = flag.String("zpgz_endpoint", ":9998", "The port to export zPages.")
	tLogEndpoint = flag.String("tlog_endpoint", "", "The gRPC endpoint of the Trillian Log Server.")
	tLogID       = flag.Int64("tlog_id", 0, "Trillian Log ID")
	packageCache = flag.String("package_cache", "", "Path to local PyPi cache of packages.")
)

func main() {
	log.Printf("Starting: %s", serviceName)
	defer func() {
		log.Printf("Stopping:%s", serviceName)
	}()

	flag.Parse()

	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		log.Fatal(err)
	}

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
	view.SetReportingPeriod(60 * time.Second)

	trace.RegisterExporter(oc)

	// TODO(dazwilkin) Reduce Trace Sample Frequency
	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.AlwaysSample(),
	})

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
		listen, err := net.Listen("tcp", *grpcEndpoint)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Starting gRPC Listener [%s]\n", *grpcEndpoint)
		log.Fatal(grpcServer.Serve(listen))
	}()
	// zPages
	wg.Add(1)
	go func() {
		defer wg.Done()
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
		log.Printf("Starting: zPages Listener [%s]\n", *zpgzEndpoint)
		log.Printf("zPages RPC Stats %s/debug/rpcz\n", *zpgzEndpoint)
		log.Printf("zPages Trace Spans %s/debug/tracez\n", *zpgzEndpoint)
		log.Fatal(zpgzServer.Serve(listen))
	}()
	wg.Wait()
}