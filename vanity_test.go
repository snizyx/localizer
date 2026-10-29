// Copyright (c) 2026 Snizyx Software LLC. All rights reserved.
// SPDX-License-Identifier: NCSA

package localizer_test

import (
	"flag"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The module is published under the vanity path locale.dev/localizer. To find the code, the go command (or
// the module proxy) requests https://locale.dev/localizer/<package>?go-get=1 and reads the go-import meta
// tag; the site serves static pages for that from site/public/localizer. With GOPROXY=direct the go command
// asks for the imported package's own URL, so every importable package needs a page. This test keeps the
// pages in step with the packages; regenerate them with
//
//	go test -run TestVanityPages -update-vanity .
var updateVanity = flag.Bool("update-vanity", false, "rewrite the vanity import pages in site/public/localizer")

const (
	modulePath = "locale.dev/localizer"
	repoURL    = "https://github.com/snizyx/localizer"
	vanityDir  = "site/public/localizer"
)

// vanityPage is the page for one package: the go-import and go-source tags, and a redirect for people to
// the package's documentation.
func vanityPage(importPath string) string {
	doc := "https://pkg.go.dev/" + importPath
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="go-import" content="` + modulePath + ` git ` + repoURL + `">
<meta name="go-source" content="` + modulePath + ` ` + repoURL + ` ` + repoURL + `/tree/main{/dir} ` + repoURL + `/blob/main{/dir}/{file}#L{line}">
<meta name="robots" content="noindex">
<meta http-equiv="refresh" content="0; url=` + doc + `">
<title>` + importPath + `</title>
</head>
<body>
<a href="` + doc + `">` + importPath + `</a>
</body>
</html>
`
}

// importable lists the module's packages that other modules can import, as directories relative to the
// module root ("." for the root package): directories with non-test Go files, outside internal/, testdata/
// and the non-Go parts of the repository.
func importable(t *testing.T) []string {
	t.Helper()
	var dirs []string
	err := filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if p != "." && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") ||
				name == "internal" || name == "testdata" || name == "site" || name == "python" || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
			dirs = append(dirs, filepath.ToSlash(filepath.Dir(p)))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(dirs)
	out := dirs[:0]
	for i, d := range dirs {
		if i == 0 || d != dirs[i-1] {
			out = append(out, d)
		}
	}
	return out
}

func pageFile(dir string) string {
	return filepath.FromSlash(path.Join(vanityDir, dir, "index.html"))
}

func TestVanityPages(t *testing.T) {
	want := map[string]string{} // page file -> content
	for _, dir := range importable(t) {
		importPath := modulePath
		if dir != "." {
			importPath += "/" + dir
		}
		want[pageFile(dir)] = vanityPage(importPath)
	}
	if len(want) == 0 || want[pageFile(".")] == "" {
		t.Fatal("found no root package")
	}

	if *updateVanity {
		if err := os.RemoveAll(vanityDir); err != nil {
			t.Fatal(err)
		}
		for file, content := range want {
			if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return
	}

	for file, content := range want {
		got, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("%s: %v (run: go test -run TestVanityPages -update-vanity .)", file, err)
			continue
		}
		if string(got) != content {
			t.Errorf("%s is out of date (run: go test -run TestVanityPages -update-vanity .)", file)
		}
	}
	err := filepath.WalkDir(vanityDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && want[p] == "" {
			t.Errorf("%s belongs to no package; remove it (or run with -update-vanity)", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
