// Copyright © 2020 Bjørn Erik Pedersen <bjorn.erik.pedersen@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package libsass

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	qt "github.com/frankban/quicktest"
)

func _TestWithImportResolver(t *testing.T) {
	c := qt.New(t)
	src := bytes.NewBufferString(`
@import "colors";

div { p { color: $white; } }`)

	var dst bytes.Buffer

	importResolver := func(url string, prev string) (string, string, bool) {
		// This will make every import the same, which is probably not a common use
		// case.
		return url, `$white:    #fff`, true
	}

	transpiler, err := New(Options{ImportResolver: importResolver})
	c.Assert(err, qt.IsNil)

	_, err = transpiler.Execute(&dst, src)
	c.Assert(err, qt.IsNil)
	c.Assert(dst.String(), qt.Equals, "div p {\n  color: #fff; }\n")
}

func TestSassSyntax(t *testing.T) {
	c := qt.New(t)
	src := bytes.NewBufferString(`
$color: #333;

.content-navigation
  border-color: $color
`)

	var dst bytes.Buffer

	transpiler, err := New(Options{OutputStyle: CompressedStyle, SassSyntax: true})
	c.Assert(err, qt.IsNil)

	_, err = transpiler.Execute(&dst, src)
	c.Assert(err, qt.IsNil)
	c.Assert(dst.String(), qt.Equals, ".content-navigation{border-color:#333}\n")
}

func TestOutputStyle(t *testing.T) {
	c := qt.New(t)
	src := bytes.NewBufferString(`
div { p { color: #ccc; } }`)

	var dst bytes.Buffer

	transpiler, err := New(Options{OutputStyle: CompressedStyle})
	c.Assert(err, qt.IsNil)

	_, err = transpiler.Execute(&dst, src)
	c.Assert(err, qt.IsNil)
	c.Assert(dst.String(), qt.Equals, "div p{color:#ccc}\n")
}

func TestSourceMapSettings(t *testing.T) {
	dir, _ := ioutil.TempDir(os.TempDir(), "tocss")
	defer os.RemoveAll(dir)

	colors := filepath.Join(dir, "_colors.scss")

	ioutil.WriteFile(colors, []byte(`
$moo:       #f442d1 !default;
`), 0755)

	c := qt.New(t)
	src := bytes.NewBufferString(`
@import "colors";

div { p { color: $moo; } }`)

	var dst bytes.Buffer

	transpiler, err := New(Options{
		IncludePaths:            []string{dir},
		EnableEmbeddedSourceMap: false,
		SourceMapContents:       true,
		OmitSourceMapURL:        false,
		SourceMapFilename:       "source.map",
		OutputPath:              "outout.css",
		InputPath:               "input.scss",
		SourceMapRoot:           "/my/root",
	})
	c.Assert(err, qt.IsNil)

	result, err := transpiler.Execute(&dst, src)
	c.Assert(err, qt.IsNil)
	c.Assert(dst.String(), qt.Equals, "div p {\n  color: #f442d1; }\n\n/*# sourceMappingURL=source.map */")
	c.Assert(result.SourceMapFilename, qt.Equals, "source.map")

	c.Assert(`"sourceRoot": "/my/root",`, qt.Contains, `"sourceRoot": "/my/root",`)
	c.Assert(`"file": "outout.css",`, qt.Contains, `"file": "outout.css",`)
	c.Assert(`"input.scss",`, qt.Contains, `"input.scss",`)
	c.Assert(`mappings": "AAGA,AAAM,GAAH,CAAG,CAAC,CAAC;EAAE,KAAK,ECFH,OAAO,GDEM"`, qt.Contains, `mappings": "AAGA,AAAM,GAAH,CAAG,CAAC,CAAC;EAAE,KAAK,ECFH,OAAO,GDEM"`)
}

func TestConcurrentTranspile(t *testing.T) {

	c := qt.New(t)

	importResolver := func(url string, prev string) (string, string, bool) {
		return url, `$white:    #fff`, true
	}

	transpiler, err := New(Options{
		OutputStyle:    CompressedStyle,
		ImportResolver: importResolver})

	c.Assert(err, qt.IsNil)

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				src := bytes.NewBufferString(`
@import "colors";

div { p { color: $white; } }`)
				var dst bytes.Buffer
				_, err := transpiler.Execute(&dst, src)
				c.Assert(err, qt.IsNil)
				c.Assert(dst.String(), qt.Equals, "div p{color:#fff}\n")
			}
		}()
	}
	wg.Wait()
}

//  3000	    397942 ns/op	    2192 B/op	       4 allocs/op
func BenchmarkTranspile(b *testing.B) {
	srcs := `div { p { color: #ccc; } }`

	var src, dst bytes.Buffer

	transpiler, err := New(Options{OutputStyle: CompressedStyle})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		src.Reset()
		dst.Reset()
		src.WriteString(srcs)
		if _, err := transpiler.Execute(&dst, &src); err != nil {
			b.Fatal(err)
		}
		if dst.String() != "div p{color:#ccc}\n" {
			b.Fatal("Got:", dst.String())
		}
	}
}