package workerpool

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// TaskFunc 任务函数。
type TaskFunc func(ctx context.Context) error

// Pool 工作池。
type Pool struct {
	size   int
	jobs   chan TaskFunc
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	logger zerolog.Logger
	closed bool
	mu     sync.Mutex
}

var (
	// ErrPoolClosed 表示工作池已关闭。
	ErrPoolClosed = errors.New("工作池已关闭")
	// ErrQueueFull 表示任务队列已满。
	ErrQueueFull = errors.New("任务队列已满")
)

// New 创建工作池。
func New(size, queueSize int, logger zerolog.Logger) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		size:   size,
		jobs:   make(chan TaskFunc, queueSize),
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}
}

// Start 启动工作池。
func (p *Pool) Start() {
	p.logger.Info().
		Int("size", p.size).
		Int("queue_size", cap(p.jobs)).
		Msg("工作池启动")
	for i := 0; i < p.size; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Submit 提交任务。
func (p *Pool) Submit(task TaskFunc) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		p.logger.Warn().Msg("工作池已关闭，无法提交任务")
		return ErrPoolClosed
	}
	p.mu.Unlock()

	select {
	case p.jobs <- task:
		p.logger.Debug().Int("queue_len", len(p.jobs)).Msg("任务已进入队列")
		return nil
	default:
		p.logger.Warn().Int("queue_len", len(p.jobs)).Msg("任务队列已满")
		return ErrQueueFull
	}
}

// Shutdown 关闭工作池。
func (p *Pool) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	p.logger.Info().Msg("工作池开始关闭")
	close(p.jobs)
	p.cancel()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info().Msg("工作池已关闭")
		return nil
	case <-ctx.Done():
		p.logger.Warn().Err(ctx.Err()).Msg("工作池关闭超时")
		return ctx.Err()
	}
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()
	p.logger.Debug().Int("worker", id).Msg("工作线程启动")
	for {
		select {
		case <-p.ctx.Done():
			p.logger.Debug().Int("worker", id).Msg("工作线程退出")
			return
		case task, ok := <-p.jobs:
			if !ok {
				p.logger.Debug().Int("worker", id).Msg("任务通道关闭，工作线程退出")
				return
			}
			ctx, cancel := context.WithTimeout(p.ctx, 10*time.Minute)
			start := time.Now()
			if err := task(ctx); err != nil {
				p.logger.Error().Err(err).Int("worker", id).Msg("任务执行失败")
			} else {
				p.logger.Debug().
					Int("worker", id).
					Dur("duration", time.Since(start)).
					Msg("任务执行完成")
			}
			cancel()
		}
	}
}
