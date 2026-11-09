package script

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Lint checks a script without running it, so a broken script is caught
// before any flagwire exists: every condition and command is known, every
// regular expression compiles, every archive JSON file parses, and the
// fixtures, golden files and archive files it names exist.
func (e *Engine) Lint(s *Script) []error {
	var errs []error
	fail := func(l Line, format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s:%d: %s", s.Name, l.No, fmt.Sprintf(format, args...)))
	}
	archive := map[string]bool{}
	for _, f := range s.Files {
		archive[f.Name] = true
		if strings.HasSuffix(f.Name, ".json") && !json.Valid(f.Data) {
			errs = append(errs, fmt.Errorf("%s: archive file %s is not valid JSON", s.Name, f.Name))
		}
	}
	expand := func(name string) string {
		switch name {
		case "GOLDEN":
			return e.GoldenDir
		case "FIXTURES":
			return e.FixturesDir
		}
		return "$" + name
	}
	for _, l := range s.Lines {
		text := l.Text
		for strings.HasPrefix(text, "[") {
			end := strings.IndexByte(text, ']')
			if end < 0 {
				fail(l, "unterminated condition")
				break
			}
			c := strings.TrimPrefix(text[1:end], "!")
			switch {
			case c == "linux" || c == "darwin" || c == "windows", strings.HasPrefix(c, "exec:"):
			default:
				if _, ok := e.Conditions[c]; !ok {
					fail(l, "unknown condition [%s]", c)
				}
			}
			text = strings.TrimSpace(text[end+1:])
		}
		mode := ""
		if strings.HasPrefix(text, "! ") || strings.HasPrefix(text, "? ") {
			mode, text = text[:1], strings.TrimSpace(text[2:])
		}
		args, err := words(text, expand)
		if err != nil {
			fail(l, "%v", err)
			continue
		}
		if len(args) == 0 {
			continue
		}
		cmd, args := args[0], args[1:]
		if _, ok := commands[cmd]; !ok {
			fail(l, "unknown command %q", cmd)
			continue
		}
		if mode == "?" && cmd != "exec" {
			fail(l, "'?' only goes with exec")
		}
		switch cmd {
		case "stdout", "stderr", "grep":
			if len(args) == 0 {
				fail(l, "%s needs a regular expression", cmd)
			} else if _, err := regexp.Compile("(?m)" + args[0]); err != nil {
				fail(l, "%v", err)
			}
		case "fixture":
			if len(args) > 0 && !isDir(filepath.Join(e.FixturesDir, args[0])) {
				fail(l, "no fixture %q", args[0])
			}
		}
		for i, a := range args {
			for _, dir := range []string{e.GoldenDir, e.FixturesDir} {
				if dir != "" && strings.HasPrefix(a, dir+"/") {
					if _, err := os.Stat(a); err != nil {
						fail(l, "%s does not exist", strings.TrimPrefix(a, filepath.Dir(dir)+"/"))
					}
				}
			}
			// A file the script copies or compares from its own archive.
			names := cmd == "cp" && i == 0 || cmd == "cmp" || cmd == "cmpgo" || cmd == "jsonmatch" ||
				cmd == "stdin" || cmd == "fakellm"
			if rest, ok := strings.CutPrefix(a, "../"); ok && names && !strings.Contains(rest, "..") && !archive[rest] {
				fail(l, "%s is not in the archive", rest)
			}
		}
	}
	return errs
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}
