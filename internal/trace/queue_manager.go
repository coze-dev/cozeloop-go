// Copyright The OpenTelemetry Authors
// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: Apache-2.0
//
// This file has been modified by Bytedance Ltd. and/or its affiliates on 2025
//
// Original file was released under Apache-2.0, with the full license text
// available at https://github.com/open-telemetry/opentelemetry-go/blob/main/sdk/trace/batch_span_processor.go.
//
// This modified file is released under the same license.

package trace

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/internal/util"
)

type exportFunc func(ctx context.Context, s []interface{})

// QueueManager is a queue that batches spans and exports them
type QueueManager interface {
	Enqueue(ctx context.Context, s interface{}, byteSize int64)
	Shutdown(ctx context.Context) error
	ForceFlush(ctx context.Context) error
}

type batchQueueManagerOptions struct {
	queueName              string
	maxQueueLength         int
	batchTimeout           time.Duration
	maxExportBatchLength   int
	maxExportBatchByteSize int

	exportFunc exportFunc
}

func newBatchQueueManager(o batchQueueManagerOptions) *BatchQueueManager {
	bsp := &BatchQueueManager{
		o:          o,
		queue:      make(chan interface{}, o.maxQueueLength),
		dropped:    0,
		batch:      make([]interface{}, 0, o.maxExportBatchLength),
		batchMutex: sync.Mutex{},
		timer:      time.NewTimer(o.batchTimeout),
		exportFunc: o.exportFunc,
		stopWait:   sync.WaitGroup{},
		stopOnce:   sync.Once{},
		stopCh:     make(chan struct{}),
		stopped:    0,
	}

	util.GoSafe(context.Background(), func() {
		bsp.stopWait.Add(1)
		defer bsp.stopWait.Done()
		bsp.processQueue()
		bsp.drainQueue(context.Background())
	})

	return bsp
}

// BatchQueueManager four queue: span, span retry, file, file retry
type BatchQueueManager struct {
	o batchQueueManagerOptions

	queue   chan interface{}
	dropped uint32

	batch         []interface{}
	batchByteSize int64
	batchMutex    sync.Mutex
	timer         *time.Timer

	exportFunc func(ctx context.Context, s []interface{})

	stopWait sync.WaitGroup
	stopOnce sync.Once
	stopCh   chan struct{}
	stopped  int32
}

func (b *BatchQueueManager) processQueue() {
	defer b.timer.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case <-b.stopCh:
			return
		case <-b.timer.C:
			if len(b.batch) > 0 {
				logger.CtxDebugf(ctx, "%s time out, span length: %d, queue length: %d", b.o.queueName, len(b.batch), len(b.queue))
			}
			b.doExport(ctx)
		case sd := <-b.queue:
			if ffs, ok := sd.(forceFlushSpan); ok {
				close(ffs.flushed)
				continue
			}
			b.batchMutex.Lock()
			b.batch = append(b.batch, sd)
			shouldExport := b.isShouldExport()
			b.batchMutex.Unlock()
			if shouldExport {
				if !b.timer.Stop() { // timer reset, need stop first
					select {
					case <-b.timer.C:
					default:
					}
				}
				logger.CtxDebugf(ctx, "%s batch out, span length: %d, queue length: %d", b.o.queueName, len(b.batch), len(b.queue))

				b.doExport(ctx)
			}
		}
	}
}

func (b *BatchQueueManager) isShouldExport() bool {
	if len(b.batch) >= b.o.maxExportBatchLength {
		return true
	}
	if b.batchByteSize >= int64(b.o.maxExportBatchByteSize) {
		return true
	}

	return false
}

func (b *BatchQueueManager) drainQueue(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for {
		select {
		case sd := <-b.queue:
			if _, ok := sd.(forceFlushSpan); ok {
				continue
			}
			b.batchMutex.Lock()
			b.batch = append(b.batch, sd)
			shouldExport := len(b.batch) == b.o.maxExportBatchLength
			b.batchMutex.Unlock()

			if shouldExport {
				b.doExport(ctx)
			}
		case <-ctx.Done():
			return
		default:
			// There are no more enqueued spans. Make final export.
			b.doExport(ctx)
			return
		}
	}
}

func (b *BatchQueueManager) doExport(ctx context.Context) {
	b.timer.Reset(b.o.batchTimeout)
	b.batchMutex.Lock()
	defer b.batchMutex.Unlock()

	//logger.CtxDebugf(ctx, "%s batch length before export: %d", b.o.queueName, len(b.batch))
	if len(b.batch) > 0 {
		if b.exportFunc != nil {
			b.exportFunc(ctx, b.batch)
		}
		// delete the batch
		b.batch = b.batch[:0]
		b.batchByteSize = 0
	}
}

func (b *BatchQueueManager) Enqueue(ctx context.Context, sd interface{}, byteSize int64) {
	// Do not enqueue spans after Shutdown.
	if atomic.LoadInt32(&b.stopped) != 0 {
		return
	}

	select {
	case b.queue <- sd:
		b.batchMutex.Lock()
		b.batchByteSize += byteSize
		b.batchMutex.Unlock()
		//logger.CtxDebugf(ctx, "%s queue length: %d", b.o.queueName, len(b.queue))
		return
	default: // queue is full, not block, drop
		logger.CtxErrorf(ctx, "%s queue is full, dropped item: %v", b.o.queueName, util.ToJSON(sd))
		atomic.AddUint32(&b.dropped, 1)
	}
	return
}

func (b *BatchQueueManager) enqueueBlockOnQueueFull(ctx context.Context, sd interface{}, byteSize int64) {
	// Do not enqueue spans after Shutdown.
	if atomic.LoadInt32(&b.stopped) != 0 {
		return
	}

	select {
	case b.queue <- sd:
		return
	case <-ctx.Done():
		return
	}
}

func (b *BatchQueueManager) Shutdown(ctx context.Context) error {
	var err error
	b.stopOnce.Do(func() {
		atomic.StoreInt32(&b.stopped, 1)
		wait := make(chan struct{})
		go func() {
			close(b.stopCh)
			b.stopWait.Wait()
			close(wait)
		}()
		// Wait until the wait group is done or the context is cancelled
		select {
		case <-wait:
		case <-ctx.Done():
			err = ctx.Err()
		}
	})
	return err
}

type forceFlushSpan struct {
	flushed chan struct{}
}

func (b *BatchQueueManager) ForceFlush(ctx context.Context) error {
	// Interrupt if context is already canceled.
	if err := ctx.Err(); err != nil {
		return err
	}
	// Do nothing after Shutdown.
	if atomic.LoadInt32(&b.stopped) != 0 {
		return nil
	}

	flushCh := make(chan struct{})
	b.enqueueBlockOnQueueFull(ctx, forceFlushSpan{flushed: flushCh}, 0) // must enqueue
	select {
	case <-flushCh: // wait until span is drained into batch before forceFlushSpan
	case <-ctx.Done():
		return ctx.Err()
	}

	var err error
	b.drainQueue(ctx)
	return err
}
