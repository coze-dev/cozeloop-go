// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

func newConcurrentTestQueue(t *testing.T, opts batchQueueManagerOptions) *BatchQueueManager {
	t.Helper()
	if opts.maxQueueLength == 0 {
		opts.maxQueueLength = 32
	}
	if opts.maxExportBatchLength == 0 {
		opts.maxExportBatchLength = 10
	}
	if opts.maxExportBatchByteSize == 0 {
		opts.maxExportBatchByteSize = 1024
	}
	if opts.batchTimeout == 0 {
		opts.batchTimeout = time.Hour
	}
	opts.exportConcurrency = 2
	queue := newBatchQueueManager(opts)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := queue.Shutdown(ctx); err != nil {
			t.Error(err)
		}
	})
	return queue
}

func TestProcessConcurrentQueue(t *testing.T) {
	for _, tc := range []struct {
		name      string
		batchSize int
		byteLimit int
		timeout   time.Duration
	}{
		{name: "batch_count", batchSize: 2, byteLimit: 1024, timeout: time.Hour},
		{name: "batch_bytes", batchSize: 10, byteLimit: 6, timeout: time.Hour},
		{name: "batch_timeout", batchSize: 10, byteLimit: 1024, timeout: 10 * time.Millisecond},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exported := make(chan []interface{}, 2)
			queue := newConcurrentTestQueue(t, batchQueueManagerOptions{
				maxExportBatchLength:   tc.batchSize,
				maxExportBatchByteSize: tc.byteLimit,
				batchTimeout:           tc.timeout,
				exportFunc: func(ctx context.Context, batch []interface{}) {
					exported <- batch
				},
			})
			// Each round must independently reach its count/byte/time trigger.
			for round := 0; round < 2; round++ {
				want := []interface{}{round * 2, round*2 + 1}
				if tc.name == "batch_timeout" {
					want = []interface{}{round}
				}
				for _, item := range want {
					queue.Enqueue(context.Background(), item, 3)
				}
				select {
				case got := <-exported:
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("exported %v, want %v", got, want)
					}
				case <-time.After(5 * time.Second):
					t.Fatal("batch trigger did not export queued items")
				}
			}
		})
	}

	t.Run("workers_survive_exporter_panic", func(t *testing.T) {
		var attempts, completed int32
		queue := newConcurrentTestQueue(t, batchQueueManagerOptions{
			maxExportBatchLength: 1,
			exportFunc: func(ctx context.Context, batch []interface{}) {
				if atomic.AddInt32(&attempts, 1) <= 4 {
					panic("exporter failure")
				}
				atomic.AddInt32(&completed, 1)
			},
		})
		for i := 0; i < 12; i++ {
			queue.Enqueue(context.Background(), i, 1)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := queue.ForceFlush(ctx); err != nil {
			t.Fatal(err)
		}
		if atomic.LoadInt32(&attempts) != 12 || atomic.LoadInt32(&completed) != 8 {
			t.Fatalf("attempts=%d completed=%d, want 12 and 8", attempts, completed)
		}
	})

	t.Run("nil_exporter", func(t *testing.T) {
		queue := newConcurrentTestQueue(t, batchQueueManagerOptions{})
		queue.Enqueue(context.Background(), 1, 1)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := queue.ForceFlush(ctx); err != nil {
			t.Fatal(err)
		}
	})
}

func TestConcurrentQueueForceFlush(t *testing.T) {
	t.Run("multiple_callers", func(t *testing.T) {
		var exported int32
		queue := newConcurrentTestQueue(t, batchQueueManagerOptions{
			exportFunc: func(ctx context.Context, batch []interface{}) {
				atomic.AddInt32(&exported, int32(len(batch)))
			},
		})
		for i := 0; i < 7; i++ {
			queue.Enqueue(context.Background(), i, 1)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		results := make(chan error, 8)
		for i := 0; i < cap(results); i++ {
			go func() { results <- queue.ForceFlush(ctx) }()
		}
		for i := 0; i < cap(results); i++ {
			select {
			case err := <-results:
				if err != nil {
					t.Fatal(err)
				}
				if got := atomic.LoadInt32(&exported); got != 7 {
					t.Fatalf("flush returned with %d items exported, want 7", got)
				}
			case <-ctx.Done():
				t.Fatal("concurrent flush did not complete")
			}
		}
	})

	t.Run("canceled_context_and_closed_queue", func(t *testing.T) {
		queue := newConcurrentTestQueue(t, batchQueueManagerOptions{})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := queue.ForceFlush(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled flush returned %v", err)
		}
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := queue.Shutdown(shutdownCtx); err != nil {
			t.Fatal(err)
		}
		queue.Enqueue(context.Background(), 1, 1)
		if err := queue.ForceFlush(shutdownCtx); err != nil {
			t.Fatal(err)
		}
		if len(queue.queue) != 0 {
			t.Fatal("closed queue accepted new data")
		}
	})
}

func TestQueueExportConcurrency(t *testing.T) {
	for _, concurrency := range []int{-1, 0, 1, 3} {
		t.Run(fmt.Sprint(concurrency), func(t *testing.T) {
			const count = 12
			limit := concurrency
			if limit <= 1 {
				limit = 1
			}
			started := make(chan struct{}, count)
			release := make(chan struct{})
			var releaseOnce sync.Once
			var mu sync.Mutex
			active, peak := 0, 0
			seen := make(map[int]int)
			queue := newBatchQueueManager(batchQueueManagerOptions{
				maxQueueLength:         count,
				maxExportBatchLength:   1,
				maxExportBatchByteSize: 1024,
				batchTimeout:           time.Hour,
				exportConcurrency:      concurrency,
				exportFunc: func(ctx context.Context, batch []interface{}) {
					mu.Lock()
					active++
					if active > peak {
						peak = active
					}
					mu.Unlock()
					started <- struct{}{}
					<-release
					// Read after blocking to detect reused batch backing arrays.
					mu.Lock()
					for _, item := range batch {
						seen[item.(int)]++
					}
					active--
					mu.Unlock()
				},
			})
			t.Cleanup(func() {
				releaseOnce.Do(func() { close(release) })
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := queue.Shutdown(ctx); err != nil {
					t.Error(err)
				}
			})
			for i := 0; i < count; i++ {
				queue.Enqueue(context.Background(), i, 1)
			}
			for i := 0; i < limit; i++ {
				select {
				case <-started:
				case <-time.After(5 * time.Second):
					t.Fatal("configured concurrency was not reached")
				}
			}
			releaseOnce.Do(func() { close(release) })
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := queue.Shutdown(ctx); err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			defer mu.Unlock()
			if peak != limit {
				t.Errorf("peak concurrency = %d, want %d", peak, limit)
			}
			for i := 0; i < count; i++ {
				if seen[i] != 1 {
					t.Errorf("item %d exported %d times, want once", i, seen[i])
				}
			}
		})
	}
}

func TestConcurrentQueueFlushAndShutdownWait(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		t.Run(fmt.Sprintf("shutdown=%v", shutdown), func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			var releaseOnce sync.Once
			queue := newBatchQueueManager(batchQueueManagerOptions{
				maxQueueLength:         16,
				maxExportBatchLength:   10,
				maxExportBatchByteSize: 1024,
				batchTimeout:           time.Hour,
				exportConcurrency:      2,
				exportFunc: func(ctx context.Context, batch []interface{}) {
					close(started)
					<-release
				},
			})
			t.Cleanup(func() {
				releaseOnce.Do(func() { close(release) })
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = queue.Shutdown(ctx)
			})
			queue.Enqueue(context.Background(), 1, 1)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() {
				if shutdown {
					result <- queue.Shutdown(ctx)
				} else {
					result <- queue.ForceFlush(ctx)
				}
			}()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("partial batch was not exported")
			}
			// The export is blocked: neither operation may report success.
			cancel()
			select {
			case err := <-result:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("got %v, want context.Canceled while export is blocked", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("operation ignored context cancellation")
			}
			releaseOnce.Do(func() { close(release) })
			finished := make(chan struct{})
			go func() {
				if shutdown {
					queue.stopWait.Wait()
				} else {
					flushCtx, flushCancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer flushCancel()
					if err := queue.ForceFlush(flushCtx); err != nil {
						t.Error(err)
					}
				}
				close(finished)
			}()
			select {
			case <-finished:
			case <-time.After(5 * time.Second):
				t.Fatal("queue did not finish after exports were released")
			}
		})
	}
}

// retryTrackingExporter fails each span and file once to exercise the complete
// span -> span retry -> file -> file retry chain during flush and shutdown.
type retryTrackingExporter struct {
	mu           sync.Mutex
	spanAttempts map[string]int
	fileAttempts map[string]int
}

func (e *retryTrackingExporter) ExportSpans(ctx context.Context, spans []*entity.UploadSpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	var err error
	for _, span := range spans {
		e.spanAttempts[span.SpanID]++
		if e.spanAttempts[span.SpanID] == 1 {
			err = errors.New("retry span")
		}
	}
	return err
}

func (e *retryTrackingExporter) ExportFiles(ctx context.Context, files []*entity.UploadFile) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	var err error
	for _, file := range files {
		e.fileAttempts[file.TosKey]++
		if e.fileAttempts[file.TosKey] == 1 {
			err = errors.New("retry file")
		}
	}
	return err
}

func TestConcurrentProcessorDrainsRetryAndFileQueues(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		t.Run(fmt.Sprintf("shutdown=%v", shutdown), func(t *testing.T) {
			exporter := &retryTrackingExporter{
				spanAttempts: make(map[string]int),
				fileAttempts: make(map[string]int),
			}
			processor := NewBatchSpanProcessor(exporter, nil, nil, nil, &QueueConf{
				SpanMaxExportBatchLength: 1,
				ExportConcurrency:        3,
			})
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := processor.Shutdown(ctx); err != nil {
					t.Error(err)
				}
			})
			const count = 4
			input := strings.Repeat("x", consts.MaxBytesOfOneTagValueOfInputOutput+1)
			for i := 0; i < count; i++ {
				processor.OnSpanEnd(context.Background(), &Span{
					SpanContext:      SpanContext{SpanID: fmt.Sprint(i), TraceID: "trace"},
					TagMap:           map[string]interface{}{tracespec.Input: input},
					ultraLargeReport: true,
				})
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var err error
			if shutdown {
				err = processor.Shutdown(ctx)
			} else {
				err = processor.ForceFlush(ctx)
			}
			if err != nil {
				t.Fatal(err)
			}
			exporter.mu.Lock()
			defer exporter.mu.Unlock()
			for kind, attempts := range map[string]map[string]int{
				"span": exporter.spanAttempts, "file": exporter.fileAttempts,
			} {
				if len(attempts) != count {
					t.Errorf("%s count = %d, want %d", kind, len(attempts), count)
				}
				for id, n := range attempts {
					if n != 2 {
						t.Errorf("%s %s attempted %d times, want 2", kind, id, n)
					}
				}
			}
		})
	}
}
