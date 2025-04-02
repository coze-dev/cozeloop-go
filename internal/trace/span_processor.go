// Copyright The OpenTelemetry Authors
// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: Apache-2.0
//
// This file has been modified by Bytedance Ltd. and/or its affiliates on 2025
//
// Original file was released under Apache-2.0, with the full license text
// available at https://github.com/open-telemetry/opentelemetry-go/blob/main/sdk/trace/span_processor.go and
// https://github.com/open-telemetry/opentelemetry-go/blob/main/sdk/trace/batch_span_processor.go.
//
// This modified file is released under the same license.

package trace

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/coze-dev/cozeloop-go/internal/httpclient"
	"github.com/coze-dev/cozeloop-go/internal/logger"
)

// Defaults for batchQueueManagerOptions.
const (
	DefaultMaxQueueLength         = 2048
	DefaultMaxExportBatchLength   = 512
	DefaultMaxExportBatchByteSize = 4 * 1024 * 1024 // 4MB
	MaxRetryExportBatchLength     = 50
	DefaultScheduleDelay          = 1000 // millisecond

	MaxFileQueueLength         = 512
	MaxFileExportBatchLength   = 5
	MaxFileExportBatchByteSize = 100 * 1024 * 1024 // 100MB
	FileScheduleDelay          = 5000              // millisecond
)

var _ SpanProcessor = (*BatchSpanProcessor)(nil)

type SpanProcessor interface {
	OnSpanEnd(ctx context.Context, s *Span)
	Shutdown(ctx context.Context) error
	ForceFlush(ctx context.Context) error
}

func NewBatchSpanProcessor(client *httpclient.Client) SpanProcessor {
	exporter := &SpanExporter{client: client}
	fileRetryQM := newBatchQueueManager(
		batchQueueManagerOptions{
			queueName:              "file retry",
			batchTimeout:           time.Duration(FileScheduleDelay) * time.Millisecond,
			maxQueueLength:         MaxFileQueueLength,
			maxExportBatchLength:   MaxFileExportBatchLength,
			maxExportBatchByteSize: MaxFileExportBatchByteSize,
			exportFunc:             newExportFilesFunc(exporter, nil),
		})
	fileQM := newBatchQueueManager(
		batchQueueManagerOptions{
			queueName:              "file",
			batchTimeout:           time.Duration(FileScheduleDelay) * time.Millisecond,
			maxQueueLength:         MaxFileQueueLength,
			maxExportBatchLength:   MaxFileExportBatchLength,
			maxExportBatchByteSize: MaxFileExportBatchByteSize,
			exportFunc:             newExportFilesFunc(exporter, fileRetryQM),
		})

	spanRetryQM := newBatchQueueManager(
		batchQueueManagerOptions{
			queueName:              "span retry",
			batchTimeout:           time.Duration(DefaultScheduleDelay) * time.Millisecond,
			maxQueueLength:         DefaultMaxQueueLength,
			maxExportBatchLength:   MaxRetryExportBatchLength,
			maxExportBatchByteSize: DefaultMaxExportBatchByteSize,
			exportFunc:             newExportSpansFunc(exporter, nil, fileQM),
		})

	spanQM := newBatchQueueManager(
		batchQueueManagerOptions{
			queueName:              "span",
			batchTimeout:           time.Duration(DefaultScheduleDelay) * time.Millisecond,
			maxQueueLength:         DefaultMaxQueueLength,
			maxExportBatchLength:   DefaultMaxExportBatchLength,
			maxExportBatchByteSize: DefaultMaxExportBatchByteSize,
			exportFunc:             newExportSpansFunc(exporter, spanRetryQM, fileQM),
		})

	return &BatchSpanProcessor{
		spanQM:      spanQM,
		spanRetryQM: spanRetryQM,
		fileQM:      fileQM,
		fileRetryQM: fileRetryQM,
	}
}

// BatchSpanProcessor implements SpanProcessor
type BatchSpanProcessor struct {
	spanQM      QueueManager
	spanRetryQM QueueManager
	fileQM      QueueManager
	fileRetryQM QueueManager

	exporter SpanExporter

	stopped int32
}

func (b *BatchSpanProcessor) OnSpanEnd(ctx context.Context, s *Span) {
	if atomic.LoadInt32(&b.stopped) != 0 {
		return
	}

	b.spanQM.Enqueue(ctx, s, s.bytesSize)
}

func (b *BatchSpanProcessor) Shutdown(ctx context.Context) error {
	if err := b.spanQM.Shutdown(ctx); err != nil {
		return err
	}
	if err := b.spanRetryQM.Shutdown(ctx); err != nil {
		return err
	}
	if err := b.fileQM.Shutdown(ctx); err != nil {
		return err
	}
	if err := b.fileRetryQM.Shutdown(ctx); err != nil {
		return err
	}

	atomic.StoreInt32(&b.stopped, 1)
	return nil
}

func (b *BatchSpanProcessor) ForceFlush(ctx context.Context) error {
	if err := b.spanQM.ForceFlush(ctx); err != nil {
		return err
	}
	if err := b.spanRetryQM.ForceFlush(ctx); err != nil {
		return err
	}
	if err := b.fileQM.ForceFlush(ctx); err != nil {
		return err
	}
	if err := b.fileRetryQM.ForceFlush(ctx); err != nil {
		return err
	}

	return nil
}

func newExportSpansFunc(exporter Exporter, spanRetryQueue QueueManager, fileQueue QueueManager) exportFunc {
	return func(ctx context.Context, l []interface{}) {
		spans := make([]*Span, 0, len(l))
		for _, s := range l {
			if span, ok := s.(*Span); ok {
				spans = append(spans, span)
			}
		}
		uploadSpans, uploadFiles := transferToUploadSpanAndFile(ctx, spans)
		if err := exporter.ExportSpans(ctx, uploadSpans); err != nil { // fail, send to retry queue.
			if spanRetryQueue != nil {
				for _, span := range spans {
					spanRetryQueue.Enqueue(ctx, span, span.bytesSize)
				}
			}
		} else { // success, send to file queue.
			for _, file := range uploadFiles {
				if file == nil {
					continue
				}
				if fileQueue != nil {
					fileQueue.Enqueue(ctx, file, int64(len(file.Data)))
				}
			}
		}
	}
}

func newExportFilesFunc(exporter Exporter, fileRetryQueue QueueManager) exportFunc {
	return func(ctx context.Context, l []interface{}) {
		files := make([]*UploadFile, 0, len(l))
		for _, f := range l {
			if file, ok := f.(*UploadFile); ok {
				files = append(files, file)
			}
		}
		if err := exporter.ExportFiles(ctx, files); err != nil {
			logger.CtxWarnf(ctx, "exporter export failed, err: %v", err)
			if fileRetryQueue != nil {
				for _, bat := range files {
					fileRetryQueue.Enqueue(ctx, bat, int64(len(bat.Data)))
				}
			}
		}
	}
}
