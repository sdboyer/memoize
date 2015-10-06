package memoize_test

import (
	"fmt"
	"go/build"
	"go/token"
	"testing"

	"golang.org/x/tools/go/buildutil"
)

func TestBasicMemoize(t *testing.T) {
	defer func(savedDryRun bool, savedReportError func(token.Position, string)) {
		DryRun = savedDryRun
		reportError = savedReportError
	}(DryRun, reportError)
	DryRun = true

	var ctxt *build.Context
	for _, test := range []struct {
		ctxt *build.Context    // if nil, use previous ctxt
		want map[string]string // contents of updated files
	}{
		{
			ctxt: fakeContext(map[string][]string{
				"base": {`
// +build ignore
//go:generate memoize
//memoize:exfunc

package base

func Simple(x int) int {
	return x>>2 + 1
}
`},
			}),
			want: map[string]string{
				"/go/src/base/0.go": `
// +build ignore
//go:generate memoize
//memoize:exfunc

package base

func Simple(x int) int {
	return x>>2 + 1
}
`,
				"/go/src/base/0_memo.go": `
package base

var simple_memocache = make(map[int]int)

func Simple(x int) int {
	var ret int
	if ret, exists := simple_memocache[x]; !exists {
		ret = x>>2 + 1

		simple_memocache[x] = ret
	}

	return ret
}
`,
			},
		},
		{
			ctxt: fakeContext(map[string][]string{
				"base": {`
// +build ignore
//go:generate memoize
//memoize:exfunc

package base

func TwoParam(x int) int {
	return x>>2 + 1
}
`},
			}),
			want: map[string]string{
				"/go/src/base/0.go": `
// +build ignore
//go:generate memoize
//memoize:exfunc

package base

func TwoParam(x, y int) int {
	return x>>2 - y
}
`,
				"/go/src/base/0_memo.go": `
package base

type twoParam_memokey struct {
	x, y int
}
var twoParam_memocache = make(map[twoParam_memokey]int)

func TwoParam(x, y int) int {
	var ret int
	if ret, exists := twoParam_memocache[twoParam_memokey{x, y}]; !exists {
		ret = x>>2 - y

		twoParam_memocache[twoParam_memokey{x, y}] = ret
	}

	return ret
}
`},
		},
	} {
		if test.ctxt != nil {
			ctxt = test.ctxt
		}
		Run(ctxt, "0.go")
		// blah blah test stuff blah blah
	}
}

// Simplifying wrapper around buildutil.FakeContext for packages whose
// filenames are sequentially numbered (%d.go).  pkgs maps a package
// import path to its list of file contents.
func fakeContext(pkgs map[string][]string) *build.Context {
	pkgs2 := make(map[string]map[string]string)
	for path, files := range pkgs {
		filemap := make(map[string]string)
		for i, contents := range files {
			filemap[fmt.Sprintf("%d.go", i)] = contents
		}
		pkgs2[path] = filemap
	}
	return buildutil.FakeContext(pkgs2)
}
