package app

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/dig"
)

type dummyWorker struct {
	started atomic.Bool
	stopped atomic.Bool
}

func (w *dummyWorker) Run(ctx context.Context) {
	w.started.Store(true)
	<-ctx.Done()
	w.stopped.Store(true)
}

func TestRegisterWorker(t *testing.T) {
	Test()

	workerInstance := &dummyWorker{}
	RegisterWorker(func() Worker {
		return workerInstance
	})

	var resolvedWorkers []Worker
	Invoke(func(p struct {
		dig.In
		Workers []Worker `group:"workers"`
	}) {
		resolvedWorkers = p.Workers
	})

	assert.Len(t, resolvedWorkers, 1)

	ctx, cancel := context.WithCancel(context.Background())
	go resolvedWorkers[0].Run(ctx)

	time.Sleep(10 * time.Millisecond)
	assert.True(t, workerInstance.started.Load())
	assert.False(t, workerInstance.stopped.Load())

	cancel()
	time.Sleep(10 * time.Millisecond)
	assert.True(t, workerInstance.stopped.Load())
}
