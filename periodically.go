package periodically

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

var ErrWorkersStoppedForcibly = errors.New("workers were stopped forcibly")
var ErrCallMethodEveryAfterStart = errors.New("unable to register a worker after calling <Start> method")

type worker struct {
	fn                      func(ctx context.Context)
	interval                time.Duration
	prevOperationInProgress atomic.Bool
}

func (w *worker) Start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !w.prevOperationInProgress.CompareAndSwap(false, true) {
					continue
				}
				w.fn(ctx)
				w.prevOperationInProgress.Store(false)
			}
		}
	}()
}

type Manager struct {
	log               *slog.Logger
	ctx               context.Context
	stop              context.CancelFunc
	started           atomic.Bool
	workers           []*worker
	allWorkersStopped chan struct{}
}

func NewManager(log *slog.Logger) *Manager {

	if log == nil {
		log = slog.Default()
	}

	ctx, stop := context.WithCancel(context.Background())

	return &Manager{
		ctx:               ctx,
		stop:              stop,
		log:               log,
		allWorkersStopped: make(chan struct{}),
	}
}

func (m *Manager) Start() {

	if !m.started.CompareAndSwap(false, true) {
		return
	}

	wg := &sync.WaitGroup{}

	for _, worker := range m.workers {
		worker.Start(m.ctx, wg)
	}

	go func() {
		wg.Wait()
		close(m.allWorkersStopped)
	}()

	m.log.Info("periodically workers are running")

}

func (m *Manager) Stop(ctx context.Context) error {

	const op = "periodically.Stop"

	if !m.started.Load() {
		m.log.Warn("You can not call <Stop> method before <Start>")
		return nil
	}

	m.stop()

	select {
	case <-m.allWorkersStopped:
		m.log.Info("all periodically workers stopped successfully")
		return nil
	case <-ctx.Done():
		m.log.Info("periodically workers stopped forcibly")
		return fmt.Errorf("%s: %w", op, ErrWorkersStoppedForcibly)
	}

}

func (m *Manager) Every(interval time.Duration, fn func(ctx context.Context)) error {

	if m.started.Load() {
		return ErrCallMethodEveryAfterStart
	}

	worker := &worker{
		fn:       fn,
		interval: interval,
	}
	m.workers = append(m.workers, worker)

	return nil

}
