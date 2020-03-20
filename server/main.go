package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
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
	// Cloud Run requires ability to set service endpoint with PORT environment variable
	port = os.Getenv("PORT")
)

var (
	// If set, grpcEndpoint overrides PORT environment value
	grpcEndpoint = flag.String("grpc_endpoint", "", "The gRPC endpoint to listen on.")
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
	if *grpcEndpoint == "" {
		if port == "" {
			log.Fatal("service requires either `--grpcEndpoint` or `PORT` environment value to be set")
		}
		// if port has a value and grpcEndpoint does not, set grpcEndpoint to the value of port
		log.Printf("Assigning gRPC Endpoint using `PORT` [%s]", port)
		*grpcEndpoint = fmt.Sprintf(":%s", port)
	}
	log.Printf("gRPC endpoint [%s]", *grpcEndpoint)

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
		log.Printf("Starting: gRPC Listener [%s]\n", *grpcEndpoint)
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
