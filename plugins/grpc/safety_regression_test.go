package xGrpc_test

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"context"
	"google.golang.org/grpc"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	xGrpc "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2"
)

func TestSafetyRegression_StartRegistrarPanic_ReleasesListener(t *testing.T) {
	app := application.New(
		application.WithGraceTimeout(10*time.Millisecond),
		application.WithCleanupTimeout(100*time.Millisecond),
		application.WithShutdownTimeout(1*time.Second),
	)

	addrCh := make(chan string, 1)
	var (
		capturedPlugin *xGrpc.Plugin
		capturedMu     sync.Mutex
	)

	info, originalFactory := xGrpc.New(
		xGrpc.WithAddr("127.0.0.1:0"),
		xGrpc.WithService(func(ctx context.Context, s *grpc.Server) {
			capturedMu.Lock()
			p := capturedPlugin
			capturedMu.Unlock()
			if p != nil {
				addrCh <- p.Addr()
			}
			panic("fixture private error")
		}),
	)

	wrappedFactory := func() application.RunnablePlugin {
		raw := originalFactory()
		if p, ok := raw.(*xGrpc.Plugin); ok {
			capturedMu.Lock()
			capturedPlugin = p
			capturedMu.Unlock()
		}
		return raw
	}

	if err := app.RegisterWorker(info, wrappedFactory); err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	runDone := make(chan struct{})
	var (
		runErr   error
		runErrMu sync.Mutex
	)

	go func() {
		defer close(runDone)
		err := app.Run()
		runErrMu.Lock()
		runErr = err
		runErrMu.Unlock()
	}()

	t.Cleanup(func() {
		_ = app.Shutdown()
		select {
		case <-runDone:
		case <-time.After(2 * time.Second):
			t.Error("failfast: timed out waiting for app.Run to finish in cleanup")
		}
	})

	var capturedAddr string
	select {
	case addr := <-addrCh:
		capturedAddr = addr
	case <-time.After(2 * time.Second):
		t.Fatal("failfast: timed out waiting for capturedAddr")
	}

	select {
	case <-runDone:
	case <-time.After(2 * time.Second):
		t.Fatal("failfast: timed out waiting for app.Run")
	}

	runErrMu.Lock()
	finalRunErr := runErr
	runErrMu.Unlock()

	if finalRunErr == nil {
		t.Fatal("expected app.Run to fail with panic error, got nil")
	}

	var hookPanic *application.HookPanicError
	if !errors.As(finalRunErr, &hookPanic) {
		t.Fatalf("expected HookPanicError, got %T: %v", finalRunErr, finalRunErr)
	}

	if err := app.Shutdown(); err != nil {
		t.Logf("Shutdown returned: %v", err)
	}

	l, err := net.Listen("tcp", capturedAddr)
	if err != nil {
		t.Fatalf("expected port %s to be released and rebindable, got: %v", capturedAddr, err)
	}
	_ = l.Close()
}
