// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package prompt

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/coze-dev/cozeloop-go/internal/httpclient"
)

func TestOpenAPIClient_MPullPrompt(t *testing.T) {
	ctx := context.Background()
	client := &OpenAPIClient{
		httpClient: &httpclient.Client{},
	}

	Convey("Test MPullPrompt method", t, func() {
		Convey("When queries are less than maximum batch size", func() {
			// Mock the doMPullPrompt method
			mockDoMPullPrompt := Mock((*OpenAPIClient).doMPullPrompt).Return([]*PromptResult{
				{
					Query: PromptQuery{
						PromptKey: "key1",
						Version:   "1.0",
					},
					Prompt: &Prompt{
						WorkspaceID:    "workspace1",
						PromptKey:      "key1",
						Version:        "1.0",
						PromptTemplate: &PromptTemplate{},
					},
				},
			}, nil).Build()
			defer mockDoMPullPrompt.UnPatch()

			// Prepare test request
			req := MPullPromptRequest{
				WorkSpaceID: "workspace1",
				Queries: []PromptQuery{
					{PromptKey: "key1", Version: "1.0"},
				},
			}

			results, err := client.MPullPrompt(ctx, req)
			So(err, ShouldBeNil)
			So(results, ShouldNotBeNil)
			So(len(results), ShouldEqual, 1)
			So(results[0].Query.PromptKey, ShouldEqual, "key1")
			So(results[0].Prompt.WorkspaceID, ShouldEqual, "workspace1")
			So(mockDoMPullPrompt.Times(), ShouldEqual, 1)
		})

		Convey("When queries exceed maximum batch size", func() {
			// Create a request with more than maxPromptQueryBatchSize queries
			var queries []PromptQuery
			for i := 0; i < maxPromptQueryBatchSize+5; i++ {
				queries = append(queries, PromptQuery{
					PromptKey: "key" + string(rune('a'+i)),
					Version:   "1.0",
				})
			}

			req := MPullPromptRequest{
				WorkSpaceID: "workspace1",
				Queries:     queries,
			}

			// Mock the singleflightMPullPrompt method to return different results for each batch
			mockSingleflightMPullPrompt := Mock((*OpenAPIClient).singleflightMPullPrompt).
				To(func(ctx context.Context, req MPullPromptRequest) ([]*PromptResult, error) {
					var results []*PromptResult
					for _, q := range req.Queries {
						results = append(results, &PromptResult{
							Query: q,
							Prompt: &Prompt{
								WorkspaceID: req.WorkSpaceID,
								PromptKey:   q.PromptKey,
								Version:     q.Version,
							},
						})
					}
					return results, nil
				}).Build()
			defer mockSingleflightMPullPrompt.UnPatch()

			results, err := client.MPullPrompt(ctx, req)
			So(err, ShouldBeNil)
			So(results, ShouldNotBeNil)
			So(len(results), ShouldEqual, len(queries))
			So(mockSingleflightMPullPrompt.Times(), ShouldEqual, 2) // Should be called twice for 2 batches
		})

		Convey("When singleflightMPullPrompt returns error", func() {
			// Mock the singleflightMPullPrompt method to return an error
			mockError := Mock((*OpenAPIClient).singleflightMPullPrompt).
				Return(nil, errors.New("server error")).Build()
			defer mockError.UnPatch()

			req := MPullPromptRequest{
				WorkSpaceID: "workspace1",
				Queries: []PromptQuery{
					{PromptKey: "key1", Version: "1.0"},
				},
			}

			results, err := client.MPullPrompt(ctx, req)
			So(err, ShouldNotBeNil)
			So(results, ShouldBeNil)
			So(mockError.Times(), ShouldEqual, 1)
		})

		Convey("When queries are sorted correctly", func() {
			// Prepare unsorted queries
			req := MPullPromptRequest{
				WorkSpaceID: "workspace1",
				Queries: []PromptQuery{
					{PromptKey: "keyC", Version: "1.0"},
					{PromptKey: "keyA", Version: "2.0"},
					{PromptKey: "keyB", Version: "1.0"},
					{PromptKey: "keyA", Version: "1.0"},
				},
			}

			// Capture the sorted queries
			var capturedReq MPullPromptRequest
			Mock((*OpenAPIClient).singleflightMPullPrompt).
				To(func(ctx context.Context, req MPullPromptRequest) ([]*PromptResult, error) {
					capturedReq = req
					return []*PromptResult{}, nil
				}).Build()

			_, _ = client.MPullPrompt(ctx, req)

			// Verify queries are sorted correctly
			So(capturedReq.Queries[0].PromptKey, ShouldEqual, "keyA")
			So(capturedReq.Queries[0].Version, ShouldEqual, "1.0")
			So(capturedReq.Queries[1].PromptKey, ShouldEqual, "keyA")
			So(capturedReq.Queries[1].Version, ShouldEqual, "2.0")
			So(capturedReq.Queries[2].PromptKey, ShouldEqual, "keyB")
			So(capturedReq.Queries[3].PromptKey, ShouldEqual, "keyC")
		})
	})
}

func TestOpenAPIClient_SingleflightMPullPrompt(t *testing.T) {
	ctx := context.Background()
	client := &OpenAPIClient{
		httpClient: &httpclient.Client{},
	}

	Convey("Test singleflightMPullPrompt method", t, func() {
		Convey("When doMPullPrompt succeeds", func() {
			// Mock the doMPullPrompt method
			expectedResults := []*PromptResult{
				{
					Query: PromptQuery{PromptKey: "key1", Version: "1.0"},
					Prompt: &Prompt{
						WorkspaceID: "workspace1",
						PromptKey:   "key1",
						Version:     "1.0",
					},
				},
			}
			mockDoMPullPrompt := Mock((*OpenAPIClient).doMPullPrompt).Return(expectedResults, nil).Build()
			defer mockDoMPullPrompt.UnPatch()

			req := MPullPromptRequest{
				WorkSpaceID: "workspace1",
				Queries: []PromptQuery{
					{PromptKey: "key1", Version: "1.0"},
				},
			}

			results, err := client.singleflightMPullPrompt(ctx, req)
			So(err, ShouldBeNil)
			So(results, ShouldResemble, expectedResults)
			So(mockDoMPullPrompt.Times(), ShouldEqual, 1)

			// Call again to test singleflight deduplication
			results2, err := client.singleflightMPullPrompt(ctx, req)
			So(err, ShouldBeNil)
			So(results2, ShouldResemble, expectedResults)
			// The doMPullPrompt should still be called second time due to singleflight
			So(mockDoMPullPrompt.Times(), ShouldEqual, 2)
		})

		Convey("When doMPullPrompt returns error", func() {
			// Mock the doMPullPrompt method to return an error
			mockError := Mock((*OpenAPIClient).doMPullPrompt).Return(nil, errors.New("server error")).Build()
			defer mockError.UnPatch()

			req := MPullPromptRequest{
				WorkSpaceID: "workspace1",
				Queries: []PromptQuery{
					{PromptKey: "key1", Version: "1.0"},
				},
			}

			results, err := client.singleflightMPullPrompt(ctx, req)
			So(err, ShouldNotBeNil)
			So(results, ShouldBeNil)
			So(mockError.Times(), ShouldEqual, 1)
		})

		Convey("When doMPullPrompt returns nil", func() {
			// Mock the doMPullPrompt method to return nil
			mockNil := Mock((*OpenAPIClient).doMPullPrompt).Return(nil, nil).Build()
			defer mockNil.UnPatch()

			req := MPullPromptRequest{
				WorkSpaceID: "workspace1",
				Queries: []PromptQuery{
					{PromptKey: "key1", Version: "1.0"},
				},
			}

			results, err := client.singleflightMPullPrompt(ctx, req)
			So(err, ShouldBeNil)
			So(results, ShouldBeNil)
			So(mockNil.Times(), ShouldEqual, 1)
		})
	})
}

func TestOpenAPIClient_DoMPullPrompt(t *testing.T) {
	ctx := context.Background()
	client := &OpenAPIClient{
		httpClient: &httpclient.Client{},
	}

	Convey("Test doMPullPrompt method", t, func() {
		Convey("When HTTP request succeeds", func() {
			// Prepare a test response
			promptResults := []*PromptResult{
				{
					Query: PromptQuery{PromptKey: "key1", Version: "1.0"},
					Prompt: &Prompt{
						WorkspaceID: "workspace1",
						PromptKey:   "key1",
						Version:     "1.0",
					},
				},
			}
			mockResp := MPullPromptResponse{
				BaseResponse: httpclient.BaseResponse{Code: 0, Msg: "success"},
				Data: PromptResultData{
					Items: promptResults,
				},
			}

			// Mock the HTTP Post method
			mockPost := Mock((*httpclient.Client).Post).To(func(ctx context.Context, path string, req interface{}, resp httpclient.OpenAPIResponse) error {
				// Verify path
				So(path, ShouldEqual, mpullPromptPath)

				// Unmarshal the mock response into the provided response object
				mockRespBytes, _ := json.Marshal(mockResp)
				return json.Unmarshal(mockRespBytes, resp)
			}).Build()
			defer mockPost.UnPatch()

			req := MPullPromptRequest{
				WorkSpaceID: "workspace1",
				Queries: []PromptQuery{
					{PromptKey: "key1", Version: "1.0"},
				},
			}

			results, err := client.doMPullPrompt(ctx, req)
			So(err, ShouldBeNil)
			So(results, ShouldResemble, promptResults)
			So(mockPost.Times(), ShouldEqual, 1)
		})

		Convey("When HTTP request fails", func() {
			// Mock the HTTP Post method to return an error
			mockError := Mock((*httpclient.Client).Post).Return(errors.New("server error")).Build()
			defer mockError.UnPatch()

			req := MPullPromptRequest{
				WorkSpaceID: "workspace1",
				Queries: []PromptQuery{
					{PromptKey: "key1", Version: "1.0"},
				},
			}

			results, err := client.doMPullPrompt(ctx, req)
			So(err, ShouldNotBeNil)
			So(results, ShouldBeNil)
			So(mockError.Times(), ShouldEqual, 1)
		})
	})
}

func TestHelperFunctions(t *testing.T) {
	Convey("Test the parseCacheKey function", t, func() {
		Convey("When key format is valid", func() {
			key := "prompt_hub:key1:1.0"
			promptKey, version, ok := parseCacheKey(key)
			So(ok, ShouldBeTrue)
			So(promptKey, ShouldEqual, "key1")
			So(version, ShouldEqual, "1.0")
		})

		Convey("When key format is invalid", func() {
			invalidKeys := []string{
				"prompt_hub:key1", // missing version
				"key1:1.0",        // missing prefix
				"invalid",         // no separators
			}

			for _, key := range invalidKeys {
				promptKey, version, ok := parseCacheKey(key)
				So(ok, ShouldBeFalse)
				So(promptKey, ShouldEqual, "")
				So(version, ShouldEqual, "")
			}
		})
	})
}
