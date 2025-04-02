// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package loop

import (
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
