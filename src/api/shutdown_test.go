package main

import (
	"context"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// fakeScheduler records that Stop was called. It satisfies services.Scheduler
// via the embedded interface; only Stop is exercised here.
type fakeScheduler struct {
	services.Scheduler
	mu      sync.Mutex
	stopped bool
}

func (f *fakeScheduler) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = true
}

func (f *fakeScheduler) wasStopped() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stopped
}

func TestShutdownRuntimeStopsSchedulersAndBackgroundWork(t *testing.T) {
	gin.SetMode(gin.TestMode)

	scheduler := &fakeScheduler{}
	registry := &SchedulerRegistry{}
	registry.Register(scheduler)

	backgroundCtx, cancelBackground := context.WithCancel(context.Background())
	defer cancelBackground()

	srv := &http.Server{Handler: gin.New()}
	runtime := serverRuntime{
		logger:           services.NewLogger(10),
		schedulers:       registry,
		cancelBackground: cancelBackground,
	}

	shutdownRuntime(runtime, srv)

	if !scheduler.wasStopped() {
		t.Fatal("expected registered schedulers to be stopped")
	}
	select {
	case <-backgroundCtx.Done():
	default:
		t.Fatal("expected the background context to be cancelled")
	}
}

// A request already being served when the signal arrives must be allowed to
// finish rather than having its connection severed.
func TestShutdownRuntimeDrainsInFlightRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	released := make(chan struct{})
	started := make(chan struct{})
	router := gin.New()
	router.GET("/slow", func(c *gin.Context) {
		close(started)
		<-released
		c.String(http.StatusOK, "finished")
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	srv := &http.Server{Handler: router}
	go func() { _ = srv.Serve(listener) }()

	type result struct {
		status int
		err    error
	}
	responses := make(chan result, 1)
	go func() {
		resp, err := http.Get("http://" + listener.Addr().String() + "/slow")
		if err != nil {
			responses <- result{err: err}
			return
		}
		defer resp.Body.Close()
		responses <- result{status: resp.StatusCode}
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("request never reached the handler")
	}

	shutdownDone := make(chan struct{})
	go func() {
		shutdownRuntime(serverRuntime{logger: services.NewLogger(10)}, srv)
		close(shutdownDone)
	}()

	// Let the handler complete only after shutdown has begun draining.
	time.Sleep(100 * time.Millisecond)
	close(released)

	select {
	case got := <-responses:
		if got.err != nil {
			t.Fatalf("in-flight request was severed by shutdown: %v", got.err)
		}
		if got.status != http.StatusOK {
			t.Fatalf("in-flight request status = %d, want 200", got.status)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("in-flight request never completed")
	}

	select {
	case <-shutdownDone:
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown did not return")
	}
}
