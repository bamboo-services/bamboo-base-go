//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/plugins/async/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/cron/v2"
	xGrpc "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const targetComponentKey ctxkey.Key = "sample_cron_target_component"

type cronTargetComponent struct {
	pool async.Pool
}

func (c *cronTargetComponent) Start(ctx context.Context) error {
	pool, getPoolErr := application.Get[async.Pool](ctx, async.PoolKey)
	if getPoolErr != nil {
		return getPoolErr
	}
	c.pool = pool
	return nil
}

func (c *cronTargetComponent) Run(ctx context.Context) error {
	task, err := c.pool.Submit(ctx, func(taskCtx context.Context) {
		fmt.Printf("[async] executed job under lifetime context: %s\n", time.Now().Format(time.RFC3339))
	})
	if err != nil {
		return fmt.Errorf("submit async task failed: %w", err)
	}
	if waitErr := task.Wait(); waitErr != nil {
		return fmt.Errorf("wait async task failed: %w", waitErr)
	}
	return nil
}

func (c *cronTargetComponent) Quiesce(context.Context) error {
	return nil
}

func (c *cronTargetComponent) Stop(context.Context) error {
	return nil
}

func main() {
	app := application.New(
		application.WithGraceTimeout(5*time.Second),
		application.WithShutdownTimeout(10*time.Second),
	)

	asyncInfo, asyncFac := async.New()
	if err := app.RegisterTool(asyncInfo, asyncFac); err != nil {
		log.Fatalf("register async tool failed: %v", err)
	}

	compInfo := application.PluginInfo{
		Key:      targetComponentKey,
		Name:     "sample-cron-component",
		Requires: []ctxkey.Key{async.PoolKey},
		Overlap:  application.OverlapPolicy{Kind: application.OverlapSkip},
	}
	if err := app.RegisterComponent(compInfo, func() application.RunnablePlugin {
		return &cronTargetComponent{}
	}); err != nil {
		log.Fatalf("register component failed: %v", err)
	}

	cronInfo, cronFac := cron.New(
		cron.WithActivator(app.Activator()),
		cron.WithIntervalJob(5*time.Second, targetComponentKey),
	)
	if err := app.RegisterWorker(cronInfo, cronFac); err != nil {
		log.Fatalf("register cron worker failed: %v", err)
	}

	grpcInfo, grpcFac := xGrpc.New(
		xGrpc.WithAddr("127.0.0.1:9090"),
		xGrpc.WithService(func(startCtx context.Context, s *grpc.Server) {
			healthServer := health.NewServer()
			healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
			grpc_health_v1.RegisterHealthServer(s, healthServer)
		}),
	)
	if err := app.RegisterWorker(grpcInfo, grpcFac); err != nil {
		log.Fatalf("register grpc worker failed: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("app run error: %v", err)
	}
}
