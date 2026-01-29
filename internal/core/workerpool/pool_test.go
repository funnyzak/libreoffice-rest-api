package workerpool

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestPoolSubmitAndShutdown(t *testing.T) {
	t.Parallel()

	logger := zerolog.New(io.Discard)
	pool := New(2, 2, logger)
	pool.Start()

	var count atomic.Int32
	var wg sync.WaitGroup
	wg.Add(2)

	_ = pool.Submit(func(ctx context.Context) error {
		defer wg.Done()
		count.Add(1)
		return nil
	})
	_ = pool.Submit(func(ctx context.Context) error {
		defer wg.Done()
		count.Add(1)
		return nil
	})

	wg.Wait()

	if count.Load() != 2 {
		t.Fatalf("期望完成 2 个任务")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := pool.Shutdown(ctx); err != nil {
		t.Fatalf("关闭失败: %v", err)
	}
}

func TestPoolQueueFull(t *testing.T) {
	t.Parallel()

	logger := zerolog.New(io.Discard)
	pool := New(1, 0, logger)
	pool.Start()

	block := make(chan struct{})
	_ = pool.Submit(func(ctx context.Context) error {
		<-block
		return nil
	})

	if err := pool.Submit(func(ctx context.Context) error { return nil }); err != ErrQueueFull {
		t.Fatalf("期望队列已满错误")
	}

	close(block)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = pool.Shutdown(ctx)
}
