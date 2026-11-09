// Package tests holds flagwire's locked acceptance suite. Nothing in this
// directory changes without a "Locked-change:" commit (see README.md).
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bjaksic84/bjaksic84-acs-project/tests/internal/script"
)

var (
	engine   *script.Engine
	buildErr error
	results  struct {
		sync.Mutex
		list []Result
	}
)

// Result is one script's outcome, as cmd/score reads it.
type Result struct {
	Name    string  `json:"name"`
	Feature string  `json:"feature"`
	Level   string  `json:"level"`
	Tier    int     `json:"tier"`
	Status  string  `json:"status"` // locked or draft
	Title   string  `json:"title"`
	Outcome string  `json:"outcome"` // pass, fail or skip
	Seconds float64 `json:"seconds"`
	Failure string  `json:"failure,omitempty"`
}

func TestMain(m *testing.M) {
	root, err := filepath.Abs("..")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	tmp, err := os.MkdirTemp("", "flagwire-acceptance-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	bin := filepath.Join(tmp, "bin")
	_ = os.MkdirAll(bin, 0o755)
	buildErr = buildFlagwire(root, bin)

	engine = &script.Engine{
		BinDir:      bin,
		FixturesDir: filepath.Join(root, "tests", "fixtures"),
		GoldenDir:   filepath.Join(root, "tests", "golden"),
		HostEnv:     hostEnv(),
		Conditions: map[string]bool{
			"net":    os.Getenv("FLAGWIRE_TEST_OFFLINE") != "1",
			"glas":   isDir(os.Getenv("FLAGWIRE_GLAS_DIR")),
			"drafts": os.Getenv("FLAGWIRE_TEST_DRAFTS") == "1",
		},
		GlasDir: os.Getenv("FLAGWIRE_GLAS_DIR"),
	}
	code := m.Run()
	if out := os.Getenv("FLAGWIRE_RESULTS"); out != "" {
		if err := writeResults(out); err != nil {
			fmt.Fprintln(os.Stderr, "writing results:", err)
			code = 2
		}
	}
	os.RemoveAll(tmp)
	os.Exit(code)
}

// buildFlagwire builds src/cmd/flagwire into bin, or links FLAGWIRE_BIN there.
func buildFlagwire(root, bin string) error {
	dst := filepath.Join(bin, "flagwire")
	if runtime.GOOS == "windows" {
		dst += ".exe"
	}
	if pre := os.Getenv("FLAGWIRE_BIN"); pre != "" {
		abs, err := filepath.Abs(pre)
		if err != nil {
			return err
		}
		return os.Symlink(abs, dst)
	}
	src := filepath.Join(root, "src")
	if _, err := os.Stat(filepath.Join(src, "go.mod")); err != nil {
		return fmt.Errorf("src/go.mod does not exist yet")
	}
	cmd := exec.Command("go", "build", "-o", dst, "./cmd/flagwire")
	cmd.Dir = src
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build ./cmd/flagwire in src: %v\n%s", err, out.String())
	}
	return nil
}

// hostEnv picks the host variables a script may see. Go's caches are passed
// explicitly, because scripts get their own HOME.
func hostEnv() map[string]string {
	env := map[string]string{}
	for _, k := range []string{
		"PATH", "TMPDIR", "GOROOT", "GOPROXY", "GOSUMDB", "GONOSUMDB", "GOPRIVATE", "GONOPROXY",
		"GOFLAGS", "GOTOOLCHAIN", "GOINSECURE",
		"HTTPS_PROXY", "HTTP_PROXY", "NO_PROXY", "https_proxy", "http_proxy", "no_proxy",
		"SSL_CERT_FILE", "SSL_CERT_DIR", "SYSTEMROOT",
	} {
		if v, ok := os.LookupEnv(k); ok {
			env[k] = v
		}
	}
	out, err := exec.Command("go", "env", "-json", "GOPATH", "GOMODCACHE", "GOCACHE").Output()
	if err == nil {
		var goenv map[string]string
		if json.Unmarshal(out, &goenv) == nil {
			for k, v := range goenv {
				if v != "" {
					env[k] = v
				}
			}
		}
	}
	return env
}

func isDir(p string) bool {
	if p == "" {
		return false
	}
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// loadScripts parses every script under scripts/.
func loadScripts(t *testing.T) []*script.Script {
	t.Helper()
	var list []*script.Script
	err := filepath.WalkDir("scripts", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".txtar") {
			return err
		}
		name := strings.TrimSuffix(filepath.ToSlash(strings.TrimPrefix(p, "scripts"+string(filepath.Separator))), ".txtar")
		s, err := script.ParseFile(p, name)
		if err != nil {
			return err
		}
		list = append(list, s)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list
}

// TestScripts checks that every script parses and names its feature after
// its directory. It needs no flagwire binary.
func TestScripts(t *testing.T) {
	for _, s := range loadScripts(t) {
		dir := strings.SplitN(s.Name, "/", 2)[0]
		if dir != s.Meta.Feature {
			t.Errorf("%s: feature %q does not match its directory %q", s.Name, s.Meta.Feature, dir)
		}
		base := s.Name[strings.LastIndex(s.Name, "/")+1:]
		if !strings.HasPrefix(base, s.Meta.Level+"-") {
			t.Errorf("%s: file name should start with its level, %q", s.Name, s.Meta.Level+"-")
		}
		for _, err := range engine.Lint(s) {
			t.Error(err)
		}
	}
}

// TestAcceptance runs every script against the flagwire binary.
func TestAcceptance(t *testing.T) {
	for _, s := range loadScripts(t) {
		s := s
		t.Run(s.Name, func(t *testing.T) {
			t.Parallel()
			start := time.Now()
			r := Result{Name: s.Name, Feature: s.Meta.Feature, Level: s.Meta.Level, Tier: s.Meta.Tier, Status: s.Meta.Status, Title: s.Meta.Title}
			defer func() {
				r.Seconds = time.Since(start).Seconds()
				switch {
				case t.Skipped():
					r.Outcome = "skip"
				case t.Failed():
					r.Outcome = "fail"
				default:
					r.Outcome = "pass"
				}
				results.Lock()
				results.list = append(results.list, r)
				results.Unlock()
			}()
			if s.Meta.Status == "draft" && !engine.Conditions["drafts"] {
				t.Skip("draft: runs only with FLAGWIRE_TEST_DRAFTS=1")
			}
			if buildErr != nil {
				r.Failure = "flagwire did not build"
				t.Fatalf("flagwire did not build: %v", buildErr)
			}
			ft := &failureTracker{T: t, r: &r}
			engine.Run(ft, s)
		})
	}
}

// failureTracker records the first line of a failure for the score.
type failureTracker struct {
	*testing.T
	r *Result
}

func (f *failureTracker) Fatalf(format string, args ...any) {
	f.T.Helper()
	msg := fmt.Sprintf(format, args...)
	first := msg
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		first = msg[:i]
	}
	f.r.Failure = first
	f.T.Fatal(msg)
}

func writeResults(path string) error {
	results.Lock()
	defer results.Unlock()
	sort.Slice(results.list, func(i, j int) bool { return results.list[i].Name < results.list[j].Name })
	data, err := json.MarshalIndent(results.list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
