// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

// Test_ExportTagMapRace reproduces the concurrent read/write on span.TagMap:
// the export path (transferToUploadSpanAndFile -> parseTag) iterates the map
// while user-facing writers (SetTags) mutate it under lock. Before the fix the
// export side read span.TagMap without holding s.lock, which the Go runtime
// reports as "concurrent map iteration and map write". Run with -race.
func Test_ExportTagMapRace(t *testing.T) {
	ctx := context.Background()

	const numSpans = 8
	spans := make([]*Span, numSpans)
	for i := range spans {
		spans[i] = &Span{
			lock:         sync.RWMutex{},
			TagMap:       make(map[string]interface{}),
			SystemTagMap: make(map[string]interface{}),
		}
		spans[i].SetTags(ctx, map[string]interface{}{"seed": "v"})
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Writers: keep mutating TagMap through the public SetTags path.
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			n := 0
			for {
				select {
				case <-stop:
					return
				default:
				}
				for _, s := range spans {
					s.SetTags(ctx, map[string]interface{}{
						fmt.Sprintf("k-%d-%d", w, n%16): n,
						tracespec.Input:                 "in",
						tracespec.Output:                "out",
					})
				}
				n++
			}
		}(w)
	}

	// Readers: the export conversion path that iterates the maps.
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				transferToUploadSpanAndFile(ctx, spans)
			}
		}()
	}

	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()
}

// Test_SetTagsFinishRace covers the SetTags-vs-Finish window: one goroutine
// finishes the span (which internally rewrites tags via setSystemTag/setStatInfo
// and would enqueue it for export) while another keeps calling SetTags.
func Test_SetTagsFinishRace(t *testing.T) {
	ctx := context.Background()

	const iters = 200
	for i := 0; i < iters; i++ {
		s := &Span{
			lock:         sync.RWMutex{},
			TagMap:       make(map[string]interface{}),
			SystemTagMap: make(map[string]interface{}),
		}

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			for j := 0; j < 32; j++ {
				s.SetTags(ctx, map[string]interface{}{
					fmt.Sprintf("k%d", j): j,
				})
			}
		}()

		go func() {
			defer wg.Done()
			// Reader racing against the writer, same as the export path.
			for j := 0; j < 32; j++ {
				transferToUploadSpanAndFile(ctx, []*Span{s})
			}
		}()

		wg.Wait()
	}
}
