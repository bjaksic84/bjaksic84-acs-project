// Package script runs flagwire's acceptance scripts.
//
// A script is a txtar archive: a comment section with commands, then files
// marked "-- path --". The command language is a small subset of
// testscript's (github.com/rogpeppe/go-internal/testscript), with a few
// commands of its own. tests/README.md documents every command.
package script

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Levels, in the order the score lists them.
var Levels = []string{"basic", "standard", "advanced"}

// Meta is the "# key: value" header of a script.
type Meta struct {
	Feature string // the feature the script tests, e.g. "scan-routes"
	Level   string // basic, standard or advanced
	Tier    int    // 1, 2 or 3
	Status  string // "locked" (default) or "draft"
	Title   string // one line saying what the script checks
}

// File is one file of the archive.
type File struct {
	Name string
	Data []byte
}

// Line is one command line of the script.
type Line struct {
	No   int    // 1-based line number in the archive
	Text string // the line as written
}

// Script is a parsed archive.
type Script struct {
	Name  string // path relative to the scripts directory, without .txtar
	Path  string
	Meta  Meta
	Lines []Line
	Files []File
}

var (
	markerRE = regexp.MustCompile(`^-- (.+) --$`)
	metaRE   = regexp.MustCompile(`^#\s*(feature|level|tier|status|title):\s*(.*?)\s*$`)
)

// ParseFile reads and parses the script at path. name is its test name.
func ParseFile(path, name string) (*Script, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s, err := Parse(name, data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	s.Path = path
	return s, nil
}

// Parse parses a script from its archive text.
func Parse(name string, data []byte) (*Script, error) {
	s := &Script{Name: name, Meta: Meta{Status: "locked"}}
	lines := strings.SplitAfter(string(data), "\n")
	i := 0
	for ; i < len(lines); i++ {
		raw := strings.TrimRight(lines[i], "\r\n")
		if markerRE.MatchString(raw) {
			break
		}
		if m := metaRE.FindStringSubmatch(raw); m != nil {
			if err := s.Meta.set(m[1], m[2]); err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			continue
		}
		text := strings.TrimSpace(raw)
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		s.Lines = append(s.Lines, Line{No: i + 1, Text: text})
	}
	var cur *File
	for ; i < len(lines); i++ {
		raw := strings.TrimRight(lines[i], "\r\n")
		if m := markerRE.FindStringSubmatch(raw); m != nil {
			s.Files = append(s.Files, File{Name: strings.TrimSpace(m[1])})
			cur = &s.Files[len(s.Files)-1]
			continue
		}
		cur.Data = append(cur.Data, lines[i]...)
	}
	if err := s.Meta.check(); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, f := range s.Files {
		if f.Name == "" || filepath.IsAbs(f.Name) || strings.Contains(f.Name, "..") {
			return nil, fmt.Errorf("bad file name %q", f.Name)
		}
		if seen[f.Name] {
			return nil, fmt.Errorf("file %q appears twice", f.Name)
		}
		seen[f.Name] = true
	}
	return s, nil
}

func (m *Meta) set(key, val string) error {
	switch key {
	case "feature":
		m.Feature = val
	case "level":
		m.Level = val
	case "tier":
		n, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("tier %q is not a number", val)
		}
		m.Tier = n
	case "status":
		m.Status = val
	case "title":
		m.Title = val
	}
	return nil
}

func (m Meta) check() error {
	var problems []string
	if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(m.Feature) {
		problems = append(problems, "missing or bad '# feature:'")
	}
	okLevel := false
	for _, l := range Levels {
		okLevel = okLevel || m.Level == l
	}
	if !okLevel {
		problems = append(problems, "'# level:' must be basic, standard or advanced")
	}
	if m.Tier < 1 || m.Tier > 3 {
		problems = append(problems, "'# tier:' must be 1, 2 or 3")
	}
	if m.Status != "locked" && m.Status != "draft" {
		problems = append(problems, "'# status:' must be locked or draft")
	}
	if m.Title == "" {
		problems = append(problems, "missing '# title:'")
	}
	if len(problems) > 0 {
		return fmt.Errorf("header: %s", strings.Join(problems, "; "))
	}
	return nil
}

// words splits a command line into words. Single quotes group words and
// stop expansion; ” inside quotes is a literal quote. $NAME and ${NAME}
// outside quotes are replaced by expand.
func words(line string, expand func(string) string) ([]string, error) {
	var out []string
	var cur bytes.Buffer
	inWord, quoted := false, false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quoted:
			if c == '\'' {
				if i+1 < len(line) && line[i+1] == '\'' {
					cur.WriteByte('\'')
					i++
					continue
				}
				quoted = false
				continue
			}
			cur.WriteByte(c)
		case c == '\'':
			quoted, inWord = true, true
		case c == ' ' || c == '\t':
			if inWord {
				out = append(out, cur.String())
				cur.Reset()
				inWord = false
			}
		case c == '$':
			name, n := varName(line[i+1:])
			if name == "" {
				cur.WriteByte(c)
				inWord = true
				continue
			}
			cur.WriteString(expand(name))
			i += n
			inWord = true
		default:
			cur.WriteByte(c)
			inWord = true
		}
	}
	if quoted {
		return nil, fmt.Errorf("unterminated quote")
	}
	if inWord {
		out = append(out, cur.String())
	}
	return out, nil
}

// varName reads a variable name after '$'. It returns the name and how many
// bytes it used.
func varName(s string) (string, int) {
	if strings.HasPrefix(s, "{") {
		end := strings.IndexByte(s, '}')
		if end < 0 {
			return "", 0
		}
		return s[1:end], end + 1
	}
	n := 0
	for n < len(s) && (s[n] == '_' || s[n] >= 'a' && s[n] <= 'z' || s[n] >= 'A' && s[n] <= 'Z' || n > 0 && s[n] >= '0' && s[n] <= '9') {
		n++
	}
	return s[:n], n
}
