package server

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi/pkg/pi"
	"github.com/sllt/pi/pkg/pi/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockingTask struct {
	once             sync.Once
	started, stopped chan struct{}
	active           atomic.Int32
	maxActive        atomic.Int32
}

func (t *blockingTask) CheckUser(ctx context.Context) error {
	cur := t.active.Add(1)
	for {
		prev := t.maxActive.Load()
		if cur <= prev || t.maxActive.CompareAndSwap(prev, cur) {
			break
		}
	}
	defer t.active.Add(-1)

	t.once.Do(func() {
		close(t.started)
		<-ctx.Done()
		close(t.stopped)
	})
	return ctx.Err()
}

func TestSchedulerStopsBeforeOwnedDependencies(t *testing.T) {
	task := &blockingTask{started: make(chan struct{}), stopped: make(chan struct{})}
	closed := make(chan struct{})
	a := testkit.New(
		t,
		pi.WithConfig(map[string]string{"LOG_LEVEL": "ERROR"}),
		pi.WithResource(pi.Resource{
			Name:      "dependency",
			Ownership: pi.Owned,
			Stop: func(context.Context) error {
				select {
				case <-task.stopped:
				default:
					t.Error("dependency closed before scheduled job stopped")
				}
				close(closed)
				return nil
			},
		}),
	)
	require.NoError(t, RegisterTaskServer(a, log.NewLogger(a.Logger()), task))

	// Construction must not trigger the scheduled task before Start.
	select {
	case <-task.started:
		t.Fatal("scheduled job started before App.Start")
	default:
	}

	require.NoError(t, a.Start(t.Context()))
	select {
	case <-task.started:
	case <-time.After(5 * time.Second):
		t.Fatal("scheduled job did not start")
	}

	require.NoError(t, a.Stop(t.Context()))
	<-closed
	assert.Equal(t, int32(1), task.maxActive.Load(), "overlapping ticks must be skipped while CheckUser is in flight")
}
