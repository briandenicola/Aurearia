package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/services"
)

type synchronizedWorkerStopper struct {
	entered chan<- struct{}
	release <-chan struct{}
}

func (w synchronizedWorkerStopper) StopWorkers(ctx context.Context) error {
	w.entered <- struct{}{}
	select {
	case <-w.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestShutdownRuntimeStopsWorkerPoolsTogetherAndWaits(t *testing.T) {
	entered, release, done := make(chan struct{}, 3), make(chan struct{}), make(chan struct{})
	worker := synchronizedWorkerStopper{entered: entered, release: release}
	go func() {
		shutdownRuntime(serverRuntime{logger: services.NewLogger(10), workers: []workerStopper{worker, worker, worker}}, &http.Server{})
		close(done)
	}()
	defer func() {
		close(release)
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("shutdown did not finish after workers exited")
		}
	}()
	for i := 0; i < 3; i++ {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("workers were not stopped concurrently")
		}
	}
	select {
	case <-done:
		t.Fatal("shutdown did not wait for workers")
	default:
	}
}
