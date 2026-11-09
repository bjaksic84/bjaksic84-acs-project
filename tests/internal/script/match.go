package script

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strings"
)

// NormalizeGo formats Go source the way cmpgo compares it: gofmt, with every
// import in a single sorted block, so import grouping doesn't matter.
func NormalizeGo(src []byte) (string, error) {
	formatted, err := format.Source(src)
	if err != nil {
		return "", err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", formatted, parser.ParseComments)
	if err != nil {
		return "", err
	}
	var specs []string
	start, end := -1, -1
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		if start < 0 {
			start = fset.Position(gd.Pos()).Offset
		}
		end = fset.Position(gd.End()).Offset
		for _, s := range gd.Specs {
			is := s.(*ast.ImportSpec)
			line := is.Path.Value
			if is.Name != nil {
				line = is.Name.Name + " " + line
			}
			specs = append(specs, line)
		}
	}
	if start < 0 {
		return string(formatted), nil
	}
	sort.Slice(specs, func(i, j int) bool {
		pi, pj := importPath(specs[i]), importPath(specs[j])
		if pi != pj {
			return pi < pj
		}
		return specs[i] < specs[j]
	})
	var b bytes.Buffer
	b.Write(formatted[:start])
	b.WriteString("import (\n")
	for _, s := range specs {
		b.WriteString("\t" + s + "\n")
	}
	b.WriteString(")")
	b.Write(formatted[end:])
	out, err := format.Source(b.Bytes())
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func importPath(spec string) string {
	if i := strings.IndexByte(spec, '"'); i >= 0 {
		return spec[i:]
	}
	return spec
}

// JSONMatch reports whether got matches the pattern want.
//
//   - Objects: every key of want must be in got, and match. Extra keys in
//     got are fine.
//   - Arrays: same length, element by element. An array whose first element
//     is the string "...contains" instead needs each of its other elements
//     to match a different element of got, in any order.
//   - The string "...any" matches any value; a string starting with "re:"
//     is a regular expression a string value must match.
//   - Everything else must be equal. Numbers compare as numbers.
func JSONMatch(got, want []byte) error {
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		return fmt.Errorf("got is not JSON: %v", err)
	}
	if err := json.Unmarshal(want, &w); err != nil {
		return fmt.Errorf("want is not JSON: %v", err)
	}
	return match("$", g, w)
}

func match(path string, got, want any) error {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: got %s, want an object", path, short(got))
		}
		keys := make([]string, 0, len(w))
		for k := range w {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			gv, ok := g[k]
			if !ok {
				return fmt.Errorf("%s: missing key %q", path, k)
			}
			if err := match(path+"."+k, gv, w[k]); err != nil {
				return err
			}
		}
		return nil
	case []any:
		g, ok := got.([]any)
		if !ok {
			return fmt.Errorf("%s: got %s, want an array", path, short(got))
		}
		if len(w) > 0 && w[0] == "...contains" {
			used := make([]bool, len(g))
			for i, we := range w[1:] {
				found := false
				for j, ge := range g {
					if !used[j] && match("", ge, we) == nil {
						used[j], found = true, true
						break
					}
				}
				if !found {
					return fmt.Errorf("%s: no element matches pattern %d: %s", path, i+1, short(we))
				}
			}
			return nil
		}
		if len(g) != len(w) {
			return fmt.Errorf("%s: got %d elements, want %d\ngot: %s", path, len(g), len(w), short(got))
		}
		for i := range w {
			if err := match(fmt.Sprintf("%s[%d]", path, i), g[i], w[i]); err != nil {
				return err
			}
		}
		return nil
	case string:
		if w == "...any" {
			return nil
		}
		if strings.HasPrefix(w, "re:") {
			gs, ok := got.(string)
			if !ok {
				return fmt.Errorf("%s: got %s, want a string matching %q", path, short(got), w[3:])
			}
			re, err := regexp.Compile(w[3:])
			if err != nil {
				return fmt.Errorf("%s: bad pattern: %v", path, err)
			}
			if !re.MatchString(gs) {
				return fmt.Errorf("%s: %q does not match %q", path, gs, w[3:])
			}
			return nil
		}
	}
	if !jsonEqual(got, want) {
		return fmt.Errorf("%s: got %s, want %s", path, short(got), short(want))
	}
	return nil
}

func jsonEqual(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return bytes.Equal(ab, bb)
}

func short(v any) string {
	b, _ := json.Marshal(v)
	s := string(b)
	if len(s) > 600 {
		s = s[:600] + "…"
	}
	return s
}

// diff is a plain line diff: "-" lines are only in a, "+" only in b.
func diff(a, b string) string {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	// Longest common subsequence, fine for files of a few hundred lines.
	n, m := len(al), len(bl)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if al[i] == bl[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}
	var out strings.Builder
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && al[i] == bl[j]:
			out.WriteString("  " + al[i] + "\n")
			i++
			j++
		case j < m && (i == n || lcs[i][j+1] >= lcs[i+1][j]):
			out.WriteString("+ " + bl[j] + "\n")
			j++
		default:
			out.WriteString("- " + al[i] + "\n")
			i++
		}
	}
	return out.String()
}
