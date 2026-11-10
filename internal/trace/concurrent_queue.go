// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"runtime/debug"
	"sync"

	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/internal/util"
)

type concurrentQueueItem struct {
	value    interface{}
	byteSize int64
}

// processConcurrentQueue owns batching and flush barriers. An unbuffered jobs
// channel bounds in-flight exports without adding another pending batch queue.
func (b *BatchQueueManager) processConcurrentQueue() {
	defer b.timer.Stop()
	ctx := context.Background()
	jobs := make(chan []interface{})
	var workers sync.WaitGroup
	var exports sync.WaitGroup
	for i := 0; i < b.o.exportConcurrency; i++ {
		workers.Add(1)
		util.GoSafe(ctx, func() {
			defer workers.Done()
			for batch := range jobs {
				// Recover per batch so a custom exporter panic cannot remove a
				// worker and leave subsequent flushes waiting forever.
				func() {
					defer exports.Done()
					defer func() {
						if err := recover(); err != nil {
							logger.CtxErrorf(ctx, "concurrent export panic: %v: %s", err, debug.Stack())
						}
					}()
					if b.exportFunc != nil {
						b.exportFunc(ctx, batch)
					}
				}()
			}
		})
	}
	defer func() {
		close(jobs)
		workers.Wait()
	}()

	batch := make([]interface{}, 0, b.o.maxExportBatchLength)
	var batchByteSize int64
	exportBatch := func() {
		if len(batch) == 0 {
			return
		}
		exports.Add(1)
		jobs <- batch
		// Workers own dispatched slices until their export completes.
		batch = make([]interface{}, 0, b.o.maxExportBatchLength)
		batchByteSize = 0
	}
	resetTimer := func() {
		if !b.timer.Stop() {
			select {
			case <-b.timer.C:
			default:
			}
		}
		b.timer.Reset(b.o.batchTimeout)
	}
	consume := func(item interface{}) {
		if marker, ok := item.(forceFlushSpan); ok {
			exportBatch()
			exports.Wait()
			close(marker.flushed)
			resetTimer()
			return
		}
		entry := item.(concurrentQueueItem)
		batch = append(batch, entry.value)
		batchByteSize += entry.byteSize
		if len(batch) >= b.o.maxExportBatchLength || batchByteSize >= int64(b.o.maxExportBatchByteSize) {
			exportBatch()
			resetTimer()
		}
	}

	for {
		select {
		case item := <-b.queue:
			consume(item)
		case <-b.timer.C:
			exportBatch()
			resetTimer()
		case <-b.stopCh:
			for {
				select {
				case item := <-b.queue:
					consume(item)
				default:
					exportBatch()
					exports.Wait()
					return
				}
			}
		}
	}
}
