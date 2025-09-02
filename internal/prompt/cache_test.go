// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package prompt

import (
	"testing"
	"time"

	. "github.com/bytedance/mockey"
	"github.com/coze-dev/cozeloop-go/entity"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPromptCache(t *testing.T) {
	Convey("Test PromptCache methods", t, func() {
		openAPI := &OpenAPIClient{}
		cache := newPromptCache("workspace1", openAPI)

		Convey("Test Get and Set methods", func() {
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
			}

			cache.Set("key1", "1.0", "", prompt)
			retrieved, found := cache.Get("key1", "1.0", "")
			So(found, ShouldBeTrue)
			So(retrieved, ShouldNotBeNil)
			So(retrieved.WorkspaceID, ShouldEqual, "workspace1")

			// Test retrieving a non-existent prompt
			_, found = cache.Get("nonexistent", "1.0", "")
			So(found, ShouldBeFalse)
		})

		Convey("Test Get and Set methods with empty version", func() {
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key2",
				Version:     "",
			}

			cache.Set("key2", "", "", prompt)
			retrieved, found := cache.Get("key2", "", "")
			So(found, ShouldBeTrue)
			So(retrieved, ShouldNotBeNil)
			So(retrieved.WorkspaceID, ShouldEqual, "workspace1")

			// Test retrieving a non-existent prompt with empty version
			_, found = cache.Get("nonexistent", "", "")
			So(found, ShouldBeFalse)
		})

		Convey("Test GetAllPromptQueries method", func() {
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
			}

			cache.Set("key1", "1.0", "", prompt)
			queries := cache.GetAllPromptQueries()
			So(len(queries), ShouldEqual, 1)
			So(queries[0].PromptKey, ShouldEqual, "key1")
			So(queries[0].Version, ShouldEqual, "1.0")
		})

		Convey("Test Start and Stop methods", func() {
			// Mock the MPullPrompt method to avoid actual API calls
			Mock((*OpenAPIClient).MPullPrompt).Return([]*PromptResult{
				{
					Query: PromptQuery{
						PromptKey: "key1",
						Version:   "1.0",
					},
					Prompt: &Prompt{
						WorkspaceID: "workspace1",
						PromptKey:   "key1",
						Version:     "1.0",
					},
				},
			}, nil).Build()

			cache := newPromptCache("workspace1", openAPI, withAsyncUpdate(true), withUpdateInterval(time.Second))
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
			}
			cache.Set(prompt.PromptKey, prompt.Version, "", prompt)
			time.Sleep(2 * time.Second) // Allow some time for async updates
			cache.Stop()
		})
	})
}
