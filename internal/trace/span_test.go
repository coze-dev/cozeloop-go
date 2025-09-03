// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	. "github.com/bytedance/mockey"
	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/httpclient"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
	. "github.com/smartystreets/goconvey/convey"
)

func newMockSpan() *Span {
	return &Span{
		SpanContext: SpanContext{
			SpanID:  "SpanID",
			TraceID: "TraceID",
			Baggage: make(map[string]string),
		},
		SpanType:            "SpanType",
		Name:                "Name",
		WorkspaceID:         "WorkspaceID",
		ParentSpanID:        "ParentSpanID",
		StartTime:           time.Now(),
		Duration:            0,
		TagMap:              make(map[string]interface{}),
		StatusCode:          0,
		ultraLargeReport:    false,
		multiModalityKeyMap: make(map[string]struct{}),
		spanProcessor:       nil,
		flags:               0,
		isFinished:          0,
		lock:                sync.RWMutex{},
	}
}

func Test_SetTag(t *testing.T) {
	ctx := context.Background()
	tagKV := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	PatchConvey("Test tagKV is empty", t, func() {
		s := newMockSpan()
		s.SetTags(ctx, map[string]interface{}{})
		So(s.GetTagMap(), ShouldBeEmpty)
	})

	PatchConvey("Test GetRectifiedMap returns empty map and cutOffKeys", t, func() {
		s := newMockSpan()
		Mock(GetMethod(s, "GetRectifiedMap")).Return(map[string]interface{}{}, []string{"key1", "key2"}).Build()
		s.SetTags(ctx, tagKV)
		So(s.GetTagMap()[consts.CutOff], ShouldNotBeNil)
	})

	PatchConvey("Test GetRectifiedMap returns non-empty map and no cutOffKeys", t, func() {
		s := newMockSpan()
		expectedMap := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		}
		Mock(GetMethod(s, "GetRectifiedMap")).Return(expectedMap, []string{}).Build()
		s.SetTags(ctx, tagKV)
		So(s.GetTagMap(), ShouldResemble, expectedMap)
		Mock(GetMethod(s, "setTagItem")).Return().Build()
	})

	PatchConvey("Test GetRectifiedMap returns non-empty map and cutOffKeys", t, func() {
		s := newMockSpan()
		expectedMap := map[string]interface{}{
			consts.CutOff: []string{"key2"},
			"key1":        "value1",
		}
		Mock(GetMethod(s, "GetRectifiedMap")).Return(expectedMap, []string{"key2"}).Build()
		s.SetTags(ctx, tagKV)
		So(s.GetTagMap(), ShouldResemble, expectedMap)
	})
}

func Test_SetBaggage(t *testing.T) {
	ctx := context.Background()
	PatchConvey("Test SetBaggage with nil Span", t, func() {
		var nilSpan *Span
		nilSpan.SetBaggage(ctx, map[string]string{"key": "value"})
		// No assertions needed as the function should return early
	})

	PatchConvey("Test SetBaggage with empty baggageItem", t, func() {
		s := newMockSpan()
		s.SetBaggage(ctx, map[string]string{})
		So(s.GetBaggage(), ShouldBeEmpty)
	})

	PatchConvey("Test SetBaggage with valid baggageItem", t, func() {
		s := newMockSpan()
		Mock(isValidBaggageItem).Return(true).Build()
		baggageItem := map[string]string{"key1": "value1", "key2": "value2"}
		s.SetBaggage(ctx, baggageItem)
		expectedTagMap := map[string]interface{}{"key1": "value1", "key2": "value2"}
		expectedBaggage := map[string]string{"key1": "value1", "key2": "value2"}
		So(s.GetTagMap(), ShouldResemble, expectedTagMap)
		So(s.GetBaggage(), ShouldResemble, expectedBaggage)
	})

	PatchConvey("Test SetBaggage with invalid baggageItem", t, func() {
		s := newMockSpan()
		Mock(isValidBaggageItem).Return(false).Build()
		baggageItem := map[string]string{"key1": "value1", "key2": "value2"}
		s.SetBaggage(ctx, baggageItem)
		So(s.GetTagMap(), ShouldBeEmpty)
		So(s.GetBaggage(), ShouldBeEmpty)
	})
}

func Test_Finish(t *testing.T) {
	ctx := context.Background()
	httpClient := httpclient.NewClient("", nil, nil, nil)
	s := &Span{
		isFinished:    0,
		spanProcessor: NewBatchSpanProcessor(nil, httpClient, nil, nil),
		lock:          sync.RWMutex{},
		TagMap:        make(map[string]interface{}),
	}

	PatchConvey("Test Span is nil", t, func() {
		var nilSpan *Span
		nilSpan.Finish(ctx)
		// No assertions needed as the function should return immediately
	})

	PatchConvey("Test isDoFinish returns false", t, func() {
		s.isFinished = 1
		s.Finish(ctx)
		// No assertions needed as the function should return immediately
	})

	PatchConvey("Test Finish success", t, func() {
		s.isFinished = 0
		s.SetTags(ctx, map[string]interface{}{
			consts.StartTimeFirstResp: time.Now().UnixMilli(),
			tracespec.InputTokens:     101,
		})
		Mock(GetMethod(s.spanProcessor, "OnSpanEnd")).Return().Build()
		s.Finish(ctx)
		So(s.Duration, ShouldBeGreaterThan, 0)
		So(s.GetTagMap()[consts.LatencyFirstResp], ShouldBeGreaterThan, 0)
		So(s.GetTagMap()[tracespec.Tokens], ShouldEqual, 101)
	})
}

func Test_SpanSpecialTag(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	s := &Span{
		isFinished: 0,
		// spanProcessor:     GetBatchSpanProcessor(httpClient, GetBatchFileProcessor(httpClient)),
		lock:   sync.RWMutex{},
		TagMap: make(map[string]interface{}),
	}

	PatchConvey("Test Span is nil", t, func() {
		var nilSpan *Span
		span := nilSpan

		span.SetInput(ctx, "llm input") // just text
		span.SetOutput(ctx, "llm output")

		span.SetError(ctx, errors.New("my err")) // done
		span.SetStatusCode(ctx, 0)               // done

		span.SetUserID(ctx, "123456")        // done
		span.SetUserIDBaggage(ctx, "123456") // done

		span.SetMessageID(ctx, "11111111")        // done
		span.SetMessageIDBaggage(ctx, "11111111") // done

		span.SetThreadID(ctx, "11111111")        // done
		span.SetThreadIDBaggage(ctx, "11111111") // done

		span.SetPrompt(ctx, entity.Prompt{}) // done

		span.SetModelProvider(ctx, "openai") // done

		span.SetModelName(ctx, "gpt-4-1106-preview") // done

		span.SetInputTokens(ctx, 232) // done

		span.SetOutputTokens(ctx, 1211) // done

		span.SetStartTimeFirstResp(ctx, now.Add(2*time.Second).UnixMilli()) // done

		So(len(span.GetTagMap()), ShouldEqual, 0)
	})

	PatchConvey("Test common span", t, func() {
		span := s

		span.SetInput(ctx, "llm input") // just text
		So(span.GetTagMap()[tracespec.Input], ShouldEqual, "llm input")

		imageBase64Str, err := getImageBytes("https://www.w3schools.com/w3images/lights.jpg")
		So(err, ShouldBeNil)
		span.SetInput(ctx, tracespec.ModelInput{ // multi-modality input
			Messages: []*tracespec.ModelMessage{
				{
					Parts: []*tracespec.ModelMessagePart{
						{
							Type:     "text",
							Text:     "test txt",
							ImageURL: nil,
							FileURL:  nil,
						},
						{
							Type: "image",
							Text: "",
							ImageURL: &tracespec.ModelImageURL{
								Name:   "test image url",
								URL:    "https://www.w3schools.com/w3images/lights.jpg", // valid image URL
								Detail: "",
							},
							FileURL: nil,
						},
						{
							Type: "image",
							Text: "",
							ImageURL: &tracespec.ModelImageURL{
								Name:   "test image binary",
								URL:    imageBase64Str, // image binary only support Base64 of JPEG、PNG、WebP
								Detail: "",
							},
							FileURL: nil,
						},
					},
				},
			},
			Tools:           nil,
			ModelToolChoice: nil,
		})
		So(span.GetTagMap()[tracespec.Input], ShouldNotBeNil)

		span.SetOutput(ctx, "llm output")
		So(span.GetTagMap()[tracespec.Output], ShouldEqual, "llm output")

		span.SetError(ctx, errors.New("my err")) // done
		So(span.GetTagMap()[tracespec.Error], ShouldEqual, "my err")

		span.SetStatusCode(ctx, 0) // done
		So(span.StatusCode, ShouldEqual, 0)

		span.SetUserID(ctx, "123456")         // done
		span.SetUserIDBaggage(ctx, "1234567") // done

		span.SetMessageID(ctx, "11111111")        // done
		span.SetMessageIDBaggage(ctx, "11111111") // done

		span.SetThreadID(ctx, "11111111")        // done
		span.SetThreadIDBaggage(ctx, "11111111") // done

		span.SetPrompt(ctx, entity.Prompt{PromptKey: "test.test.test", Version: "v1"}) // done

		span.SetModelProvider(ctx, "openai") // done

		span.SetModelName(ctx, "gpt-4-1106-preview") // done

		span.SetInputTokens(ctx, 232) // done

		span.SetOutputTokens(ctx, 1211) // done

		span.SetStartTimeFirstResp(ctx, now.Add(2*time.Second).UnixMilli()) // done

		So(len(span.GetTagMap()), ShouldEqual, 14)
		So(len(span.GetBaggage()), ShouldEqual, 7)
	})
}

func getImageBytes(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch image: status code %d", resp.StatusCode)
	}

	imageData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %v", err)
	}

	return string(imageData), nil
}
