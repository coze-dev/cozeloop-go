// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInterpolateJinja2(t *testing.T) {
	Convey("Test InterpolateJinja2 function", t, func() {
		Convey("Basic variable replacement", func() {
			Convey("Simple string variable", func() {
				template := "Hello {{ name }}"
				variables := map[string]any{"name": "world"}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Hello world")
			})

			Convey("Number variable", func() {
				template := "Count: {{ count }}"
				variables := map[string]any{"count": 42}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Count: 42")
			})

			Convey("Boolean variable", func() {
				template := "Active: {{ active }}"
				variables := map[string]any{"active": true}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Active: True")
			})

			Convey("Multiple variables", func() {
				template := "{{ greeting }} {{ name }}! Count: {{ count }}"
				variables := map[string]any{
					"greeting": "Hello",
					"name":     "world",
					"count":    42,
				}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Hello world! Count: 42")
			})
		})

		Convey("Complex Jinja2 syntax", func() {
			Convey("Conditional statement", func() {
				template := "{% if condition %}true{% else %}false{% endif %}"
				variables := map[string]any{"condition": true}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "true")

				variables = map[string]any{"condition": false}
				result, err = InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "false")
			})

			Convey("Loop statement", func() {
				template := "{% for item in items %}{{ item }}{% endfor %}"
				variables := map[string]any{"items": []string{"a", "b", "c"}}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "abc")
			})

			Convey("Loop with separator", func() {
				template := "{% for item in items %}{{ item }}{% if not loop.last %}, {% endif %}{% endfor %}"
				variables := map[string]any{"items": []string{"apple", "banana", "cherry"}}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "apple, banana, cherry")
			})

			Convey("Filter usage", func() {
				template := "{{ name|upper }}"
				variables := map[string]any{"name": "world"}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "WORLD")
			})
		})

		Convey("Array and complex data types", func() {
			Convey("Array of strings", func() {
				template := "Items: {% for item in items %}{{ item }} {% endfor %}"
				variables := map[string]any{"items": []string{"apple", "banana", "cherry"}}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Items: apple banana cherry ")
			})

			Convey("Array of integers", func() {
				template := "Numbers: {% for num in numbers %}{{ num }} {% endfor %}"
				variables := map[string]any{"numbers": []int{1, 2, 3}}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Numbers: 1 2 3 ")
			})

			Convey("Array of booleans", func() {
				template := "Flags: {% for flag in flags %}{{ flag }} {% endfor %}"
				variables := map[string]any{"flags": []bool{true, false, true}}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Flags: True False True ")
			})

			Convey("Nested object access", func() {
				template := "User: {{ user.name }}, Age: {{ user.age }}"
				variables := map[string]any{
					"user": map[string]any{
						"name": "Alice",
						"age":  30,
					},
				}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "User: Alice, Age: 30")
			})
		})

		Convey("Edge cases", func() {
			Convey("Empty template", func() {
				template := ""
				variables := map[string]any{"name": "world"}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "")
			})

			Convey("Template without variables", func() {
				template := "Hello world"
				variables := map[string]any{}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Hello world")
			})

			Convey("Variable not provided", func() {
				template := "Hello {{ name }}"
				variables := map[string]any{} // name not provided
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Hello ")
			})

			Convey("Nil variable value", func() {
				template := "Hello {{ name }}"
				variables := map[string]any{"name": nil}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Hello ")
			})
		})

		Convey("Error handling", func() {
			Convey("Invalid template syntax", func() {
				template := "Hello {{ undefined_variable.nonexistent_property }}" // Will cause runtime error
				variables := map[string]any{}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldNotBeNil)
				So(result, ShouldEqual, "")
				So(err.Error(), ShouldContainSubstring, "template render error")
			})

			Convey("Invalid Jinja2 syntax", func() {
				template := "{% for item in items %}{{ item }}" // Missing endfor
				variables := map[string]any{"items": []string{"a", "b"}}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldNotBeNil)
				So(result, ShouldEqual, "")
				So(err.Error(), ShouldContainSubstring, "template render error")
			})

			Convey("Invalid filter", func() {
				template := "{{ name|nonexistent_filter }}"
				variables := map[string]any{"name": "world"}
				result, err := InterpolateJinja2(template, variables)
				So(err, ShouldNotBeNil)
				So(result, ShouldEqual, "")
				So(err.Error(), ShouldContainSubstring, "template render error")
			})
		})
	})
}
