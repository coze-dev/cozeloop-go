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
	"fmt"
	"sync/atomic"
	"time"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/httpclient"
)

// Defaults for batchQueueManagerOptions.
const (
	DefaultMaxQueueLength         = 1024
	DefaultMaxRetryQueueLength    = 512
	DefaultMaxExportBatchLength   = 100
	DefaultMaxExportBatchByteSize = 4 * 1024 * 1024 // 4MB
	MaxRetryExportBatchLength     = 50
	DefaultScheduleDelay          = 1000 // millisecond

	MaxFileQueueLength         = 512
	MaxFileExportBatchLength   = 1
	MaxFileExportBatchByteSize = 100 * 1024 * 1024 // 100MB
	FileScheduleDelay          = 5000              // millisecond
)

type QueueConf struct {
	SpanQueueLength          int
	SpanMaxExportBatchLength int
}

var _ SpanProcessor = (*BatchSpanProcessor)(nil)

type SpanProcessor interface {
	OnSpanEnd(ctx context.Context, s *Span)
	Shutdown(ctx context.Context) error
	ForceFlush(ctx context.Context) error
}

func NewBatchSpanProcessor(
	ex Exporter,
	client *httpclient.Client,
	uploadPath *UploadPath,
	finishEventProcessor func(ctx context.Context, info *consts.FinishEventInfo),
	queueConf *QueueConf,
) SpanProcessor {
	var exporter Exporter
	spanPath := pathIngestTrace
	filePath := pathUploadFile
	if uploadPath != nil {
		if uploadPath.spanUploadPath != "" {
			spanPath = uploadPath.spanUploadPath
		}
		if uploadPath.fileUploadPath != "" {
			filePath = uploadPath.fileUploadPath
		}
	}
	exporter = &SpanExporter{
		client: client,
		uploadPath: UploadPath{
			spanUploadPath: spanPath,
			fileUploadPath: filePath,
		},
	}
	if ex != nil {
		exporter = ex
	}
	spanQueueLength := DefaultMaxQueueLength
	spanMaxExportBatchLength := DefaultMaxExportBatchLength
	if queueConf != nil {
		if queueConf.SpanQueueLength > 0 {
			spanQueueLength = queueConf.SpanQueueLength
		}
		if queueConf.SpanMaxExportBatchLength > 0 { // todo: need max limit
			spanMaxExportBatchLength = queueConf.SpanMaxExportBatchLength
		}
	}

	fileRetryQM := newBatchQueueManager(
		batchQueueManagerOptions{
			queueName:              queueNameFileRetry,
			batchTimeout:           time.Duration(FileScheduleDelay) * time.Millisecond,
			maxQueueLength:         MaxFileQueueLength,
			maxExportBatchLength:   MaxFileExportBatchLength,
			maxExportBatchByteSize: MaxFileExportBatchByteSize,
			exportFunc:             newExportFilesFunc(exporter, nil, finishEventProcessor),
			finishEventProcessor:   finishEventProcessor,
		})
	fileQM := newBatchQueueManager(
		batchQueueManagerOptions{
			queueName:              queueNameFile,
			batchTimeout:           time.Duration(FileScheduleDelay) * time.Millisecond,
			maxQueueLength:         MaxFileQueueLength,
			maxExportBatchLength:   MaxFileExportBatchLength,
			maxExportBatchByteSize: MaxFileExportBatchByteSize,
			exportFunc:             newExportFilesFunc(exporter, fileRetryQM, finishEventProcessor),
			finishEventProcessor:   finishEventProcessor,
		})

	spanRetryQM := newBatchQueueManager(
		batchQueueManagerOptions{
			queueName:              queueNameSpanRetry,
			batchTimeout:           time.Duration(DefaultScheduleDelay) * time.Millisecond,
			maxQueueLength:         DefaultMaxRetryQueueLength,
			maxExportBatchLength:   MaxRetryExportBatchLength,
			maxExportBatchByteSize: DefaultMaxExportBatchByteSize,
			exportFunc:             newExportSpansFunc(exporter, nil, fileQM, finishEventProcessor),
			finishEventProcessor:   finishEventProcessor,
		})

	spanQM := newBatchQueueManager(
		batchQueueManagerOptions{
			queueName:              queueNameSpan,
			batchTimeout:           time.Duration(DefaultScheduleDelay) * time.Millisecond,
			maxQueueLength:         spanQueueLength,
			maxExportBatchLength:   spanMaxExportBatchLength,
			maxExportBatchByteSize: DefaultMaxExportBatchByteSize,
			exportFunc:             newExportSpansFunc(exporter, spanRetryQM, fileQM, finishEventProcessor),
			finishEventProcessor:   finishEventProcessor,
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

func newExportSpansFunc(
	exporter Exporter,
	spanRetryQueue QueueManager,
	fileQueue QueueManager,
	finishEventProcessor func(ctx context.Context, info *consts.FinishEventInfo),
) exportFunc {
	return func(ctx context.Context, l []interface{}) {
		spans := make([]*Span, 0, len(l))
		for _, s := range l {
			if span, ok := s.(*Span); ok {
				spans = append(spans, span)
			}
		}
		var errMsg string
		var isFail bool
		uploadSpans, uploadFiles := transferToUploadSpanAndFile(ctx, spans)
		before := time.Now()
		err := exporter.ExportSpans(ctx, uploadSpans)
		tsMs := time.Now().Sub(before).Milliseconds()
		if err != nil { // fail, send to retry queue.
			if spanRetryQueue != nil {
				for _, span := range spans {
					spanRetryQueue.Enqueue(ctx, span, span.bytesSize)
				}
				errMsg = fmt.Sprintf("%v, retry later", err.Error())
			} else {
				errMsg = fmt.Sprintf("%v, retry second time failed", err.Error())
			}
			isFail = true
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
		if finishEventProcessor != nil {
			finishEventProcessor(ctx, &consts.FinishEventInfo{
				EventType:   consts.SpanFinishEventFlushSpanRate,
				IsEventFail: isFail,
				ItemNum:     len(uploadSpans),
				DetailMsg:   errMsg,
				ExtraParams: &consts.FinishEventInfoExtra{
					LatencyMs: tsMs,
				},
			})
		}
	}
}

func newExportFilesFunc(
	exporter Exporter,
	fileRetryQueue QueueManager,
	finishEventProcessor func(ctx context.Context, info *consts.FinishEventInfo),
) exportFunc {
	return func(ctx context.Context, l []interface{}) {
		files := make([]*entity.UploadFile, 0, len(l))
		for _, f := range l {
			if file, ok := f.(*entity.UploadFile); ok {
				files = append(files, file)
			}
		}
		var errMsg string
		var isFail bool
		before := time.Now()
		err := exporter.ExportFiles(ctx, files)
		tsMs := time.Now().Sub(before).Milliseconds()
		if err != nil {
			if fileRetryQueue != nil {
				for _, bat := range files {
					fileRetryQueue.Enqueue(ctx, bat, int64(len(bat.Data)))
				}
				errMsg = fmt.Sprintf("%v, retry later", err.Error())
			} else {
				errMsg = fmt.Sprintf("%v, retry second time failed", err.Error())
			}
			isFail = true
		}
		if finishEventProcessor != nil {
			finishEventProcessor(ctx, &consts.FinishEventInfo{
				EventType:   consts.SpanFinishEventFlushFileRate,
				IsEventFail: isFail,
				ItemNum:     len(files),
				DetailMsg:   errMsg,
				ExtraParams: &consts.FinishEventInfoExtra{
					LatencyMs: tsMs,
				},
			})
		}
	}
}
