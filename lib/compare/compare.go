/*
Copyright 2018 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package compare

import (
	"bytes"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"
	"unicode"

	"github.com/davecgh/go-spew/spew"
	"github.com/kylelemons/godebug/diff"
	"github.com/sergi/go-diff/diffmatchpatch"
	check "gopkg.in/check.v1"
)

// DeepCompare uses gocheck DeepEquals but provides nice diff if things are not equal
func DeepCompare(c *check.C, a, b interface{}) {
	c.Assert(a, check.DeepEquals, b, check.Commentf("%v\nStack:\n%v\n", Diff(a, b), string(debug.Stack())))
}

// DeepEquals is a gocheck checker that provides a readable diff in case
// comparison fails.
var DeepEquals check.Checker = &deepEqualsChecker{
	&check.CheckerInfo{Name: "DeepEquals", Params: []string{"obtained", "expected"}},
}

// Check expects two items in params (obtained and expected) and compares them using reflection.
// If comparison fails, it returns a readable diff in error.
// Implements gocheck checker interface
func (checker *deepEqualsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	if len(params) != 2 {
		panic("at least two params are required")
	}
	if isString(params[0]) && isString(params[1]) {
		info := *checker.CheckerInfo
		return (&stringEqualsChecker{
			CheckerInfo: &info,
		}).Check(params, names)
	}
	result = reflect.DeepEqual(params[0], params[1])
	if !result {
		error = Diff(params[0], params[1])
	}
	return result, error
}

// SortedSliceEquals is a gocheck checker that compares two slices after sorting them.
// It expects the slice parameters to implement sort.Interface
var SortedSliceEquals check.Checker = &sliceEqualsChecker{
	&check.CheckerInfo{Name: "SortedSliceEquals", Params: []string{"obtained", "expected"}},
}

// Check expects two slices in params (obtained and expected).
// The slices are sorted before comparison, hence they are expected to implement sort.Interface.
// If comparison fails, it returns a readable diff in error.
// Implements gocheck checker interface
func (checker *sliceEqualsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	obtained := params[0].(sort.Interface)
	sort.Sort(obtained)
	expected := params[1].(sort.Interface)
	sort.Sort(expected)

	result = reflect.DeepEqual(obtained, expected)
	if !result {
		error = Diff(obtained, expected)
	}
	return result, error
}

// StringEquals is a gocheck checker that compares two strings
var StringEquals check.Checker = &stringEqualsChecker{
	&check.CheckerInfo{Name: "StringEquals", Params: []string{"obtained", "expected"}},
}

// Check expects two slices in params (obtained and expected).
// The slices are sorted before comparison, hence they are expected to implement sort.Interface.
// If comparison fails, it returns a readable diff in error.
// Implements gocheck checker interface
func (checker *stringEqualsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	obtained := params[0].(string)
	expected := params[1].(string)
	result = reflect.DeepEqual(obtained, expected)
	if !result {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(obtained, expected, false)
		error = DiffPrettyText(*dmp, diffs)
	}
	return result, error
}

func DiffPrettyText(dmp diffmatchpatch.DiffMatchPatch, diffs []diffmatchpatch.Diff) string {
	var buf bytes.Buffer
	for _, diff := range diffs {
		text := diff.Text
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			_, _ = buf.WriteString("\x1b[32m")
			if nonPrintable(text) {
				text = replaceNonPrintable(text)
			}
			_, _ = buf.WriteString(text)
			_, _ = buf.WriteString("\x1b[0m")
		case diffmatchpatch.DiffDelete:
			_, _ = buf.WriteString("\x1b[31m")
			if nonPrintable(text) {
				text = replaceNonPrintable(text)
			}
			_, _ = buf.WriteString(text)
			_, _ = buf.WriteString("\x1b[0m")
		case diffmatchpatch.DiffEqual:
			_, _ = buf.WriteString(text)
		}
	}

	return buf.String()
}

func nonPrintable(s string) bool {
	for _, c := range s {
		if unicode.IsPrint(c) {
			return false
		}
	}
	return true
}

func replaceNonPrintable(s string) string {
	return replacer.Replace(s)
}

var replacer = strings.NewReplacer("\n", "⤶\n", "\r", "⤶\r", "\t", "⇥")

// Diff returns user friendly difference between two objects
func Diff(a, b interface{}) string {
	d := &spew.ConfigState{Indent: " ", DisableMethods: true, DisablePointerMethods: true, DisablePointerAddresses: true}
	return diff.Diff(d.Sdump(a), d.Sdump(b))
}

// Sdump returns debug-friendly text representation of a
func Sdump(a interface{}) string {
	d := &spew.ConfigState{Indent: " ", DisableMethods: true, DisablePointerMethods: true, DisablePointerAddresses: true}
	return d.Sdump(a)
}

type deepEqualsChecker struct {
	*check.CheckerInfo
}

type sliceEqualsChecker struct {
	*check.CheckerInfo
}

type stringEqualsChecker struct {
	*check.CheckerInfo
}

func isString(p interface{}) bool {
	_, ok := p.(string)
	return ok
}
