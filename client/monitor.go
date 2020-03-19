package main

import (
	"context"
	"log"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

// Inspiration: https://stackoverflow.com/a/45766707/609290
func latencyTimer(ctx context.Context, name string) func() {
	ctx, _ = tag.New(ctx, tag.Insert(keyClient, "golang"), tag.Insert(keyMethod, name))
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	startTime := time.Now()

	return func() {
		latencyMs := float64(time.Since(startTime)) / 1e6
		stats.Record(ctx, mLatencyMs.M(latencyMs))
		log.Printf("[%s] Latency: %f", name, latencyMs)
	}
}
