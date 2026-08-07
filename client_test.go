// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package cozeloop

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewClient(t *testing.T) {
	Convey("new client repeatedly", t, func() {
		client1, err := NewClient(WithWorkspaceID("123"), WithAPIToken("token"))
		So(err, ShouldBeNil)
		client2, err := NewClient(WithWorkspaceID("123"), WithAPIToken("token"))
		So(err, ShouldBeNil)
		client3, err := NewClient(WithWorkspaceID("456"), WithAPIToken("token"))
		So(err, ShouldBeNil)

		So(client1, ShouldEqual, client2)
		So(client1, ShouldNotEqual, client3)
	})
}

func TestNoSample(t *testing.T) {
	Convey("Test NoSample spans", t, func() {
		client, err := NewClient(WithWorkspaceID("test-workspace"), WithAPIToken("test-token"))
		So(err, ShouldBeNil)
		So(client, ShouldNotBeNil)

		ctx := context.Background()

		// 1. Create a normal span
		normalCtx, normalSpan := client.StartSpan(ctx, "normal-span", "test-type")
		So(normalSpan, ShouldNotBeNil)
		So(normalSpan.GetSpanID(), ShouldNotBeEmpty)

		// 2. Create a NoSample span using WithNoSample()
		noSampleCtx, noSampleSpan := client.StartSpan(normalCtx, "nosample-span", "test-type", WithNoSample())
		So(noSampleSpan, ShouldNotBeNil)
		So(noSampleSpan.GetSpanID(), ShouldBeEmpty) // NoopSpan has empty SpanID

		// 3. Create a child span from NoSample span - should also be NoopSpan
		_, childSpan := client.StartSpan(noSampleCtx, "child-span", "test-type")
		So(childSpan, ShouldNotBeNil)
		So(childSpan.GetSpanID(), ShouldBeEmpty) // Child of NoopSpan is also NoopSpan
	})
}
