package script

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TB is the part of testing.TB the engine uses.
type TB interface {
	Helper()
	Logf(format string, args ...any)
	Fatalf(format string, args ...any)
	Skipf(format string, args ...any)
	TempDir() string
	Cleanup(func())
}

// Engine runs scripts. One engine serves every script of a test run.
type Engine struct {
	BinDir      string            // directory holding the flagwire binary
	FixturesDir string            // tests/fixtures
	GoldenDir   string            // tests/golden
	HostEnv     map[string]string // variables passed through from the host
	Conditions  map[string]bool   // extra conditions, e.g. "net", "glas"
	GlasDir     string            // a Glas checkout, when [glas] holds
	ExecTimeout time.Duration     // per exec; 0 means 10 minutes
}

// state is one script run.
type state struct {
	t        TB
	e        *Engine
	s        *Script
	work     string
	cwd      string
	env      map[string]string
	stdout   string
	stderr   string
	exitCode int
	stdin    []byte
	ran      bool // an exec has run, so stdout, stderr and exitcode are set
}

// Run runs script s, failing or skipping t as the script says.
func (e *Engine) Run(t TB, s *Script) {
	t.Helper()
	work := t.TempDir()
	st := &state{t: t, e: e, s: s, work: work, cwd: work}
	st.setupEnv()
	for _, f := range s.Files {
		p := filepath.Join(work, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("%s: %v", s.Name, err)
		}
		if err := os.WriteFile(p, f.Data, 0o644); err != nil {
			t.Fatalf("%s: %v", s.Name, err)
		}
	}
	for _, l := range s.Lines {
		st.line(l)
	}
}

func (st *state) setupEnv() {
	env := map[string]string{}
	for k, v := range st.e.HostEnv {
		env[k] = v
	}
	home := filepath.Join(st.work, ".home")
	cfg := filepath.Join(st.work, ".config")
	_ = os.MkdirAll(home, 0o755)
	_ = os.MkdirAll(cfg, 0o755)
	gitconfig := filepath.Join(st.work, ".gitconfig")
	_ = os.WriteFile(gitconfig, []byte("[user]\n\tname = flagwire-test\n\temail = test@example.com\n"+
		"[init]\n\tdefaultBranch = main\n[commit]\n\tgpgsign = false\n[core]\n\thooksPath = /dev/null\n"), 0o644)
	path := st.e.BinDir
	if p := st.e.HostEnv["PATH"]; p != "" {
		path += string(os.PathListSeparator) + p
	}
	for k, v := range map[string]string{
		"WORK":                st.work,
		"HOME":                home,
		"XDG_CONFIG_HOME":     cfg,
		"FLAGWIRE_CONFIG_DIR": filepath.Join(cfg, "flagwire"),
		"FLAGWIRE_KEYRING":    "file",
		"PATH":                path,
		"GIT_CONFIG_GLOBAL":   gitconfig,
		"GIT_CONFIG_NOSYSTEM": "1",
		"NO_COLOR":            "1",
		"TZ":                  "UTC",
		"LC_ALL":              "C",
		"FIXTURES":            st.e.FixturesDir,
		"GOLDEN":              st.e.GoldenDir,
	} {
		env[k] = v
	}
	st.env = env
}

func (st *state) envList() []string {
	keys := make([]string, 0, len(st.env))
	for k := range st.env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+st.env[k])
	}
	return out
}

func (st *state) fatalf(l Line, format string, args ...any) {
	st.t.Helper()
	msg := fmt.Sprintf(format, args...)
	var b strings.Builder
	fmt.Fprintf(&b, "%s:%d: %s\n> %s\n", st.s.Name, l.No, msg, l.Text)
	if st.ran {
		fmt.Fprintf(&b, "[last exit code] %d\n", st.exitCode)
		if st.stdout != "" {
			fmt.Fprintf(&b, "[stdout]\n%s\n", clip(st.stdout))
		}
		if st.stderr != "" {
			fmt.Fprintf(&b, "[stderr]\n%s\n", clip(st.stderr))
		}
	}
	st.t.Fatalf("%s", b.String())
}

func clip(s string) string {
	const max = 6000
	if len(s) <= max {
		return s
	}
	return s[:max/2] + "\n...[clipped]...\n" + s[len(s)-max/2:]
}

// cond reports whether a condition such as "net", "!net" or "exec:git" holds.
func (st *state) cond(c string) (bool, error) {
	neg := strings.HasPrefix(c, "!")
	c = strings.TrimPrefix(c, "!")
	var v bool
	switch {
	case c == runtime.GOOS:
		v = true
	case c == "linux" || c == "darwin" || c == "windows":
		v = false
	case strings.HasPrefix(c, "exec:"):
		_, err := lookPath(strings.TrimPrefix(c, "exec:"), st.env["PATH"], st.cwd)
		v = err == nil
	default:
		known, ok := st.e.Conditions[c]
		if !ok {
			return false, fmt.Errorf("unknown condition [%s]", c)
		}
		v = known
	}
	return v != neg, nil
}

func (st *state) line(l Line) {
	st.t.Helper()
	text := l.Text
	// Conditions: [net] [!glas] ...
	for strings.HasPrefix(text, "[") {
		end := strings.IndexByte(text, ']')
		if end < 0 {
			st.fatalf(l, "unterminated condition")
		}
		ok, err := st.cond(text[1:end])
		if err != nil {
			st.fatalf(l, "%v", err)
		}
		if !ok {
			return
		}
		text = strings.TrimSpace(text[end+1:])
	}
	mode := "" // "" must succeed, "!" must fail, "?" may fail
	if strings.HasPrefix(text, "! ") || strings.HasPrefix(text, "? ") {
		mode = text[:1]
		text = strings.TrimSpace(text[2:])
	}
	args, err := words(text, func(name string) string { return st.env[name] })
	if err != nil {
		st.fatalf(l, "%v", err)
	}
	if len(args) == 0 {
		return
	}
	cmd, args := args[0], args[1:]
	fn, ok := commands[cmd]
	if !ok {
		st.fatalf(l, "unknown command %q", cmd)
	}
	if mode == "?" && cmd != "exec" {
		st.fatalf(l, "'?' only goes with exec")
	}
	fn(st, l, mode, args)
}

type command func(st *state, l Line, mode string, args []string)

var commands map[string]command

func init() {
	commands = map[string]command{
		"cd":         cmdCd,
		"chmod":      cmdChmod,
		"cmp":        cmdCmp,
		"cmpgo":      cmdCmpGo,
		"cp":         cmdCp,
		"env":        cmdEnv,
		"exec":       cmdExec,
		"exists":     cmdExists,
		"exitcode":   cmdExitCode,
		"fakegithub": cmdFakeGitHub,
		"fakellm":    cmdFakeLLM,
		"fixture":    cmdFixture,
		"gitinit":    cmdGitInit,
		"glascopy":   cmdGlasCopy,
		"grep":       cmdGrep,
		"jsonmatch":  cmdJSONMatch,
		"mkdir":      cmdMkdir,
		"mv":         cmdMv,
		"rm":         cmdRm,
		"skip":       cmdSkip,
		"stderr":     cmdStderr,
		"stdin":      cmdStdin,
		"stdout":     cmdStdout,
		"unset":      cmdUnset,
	}
}

func (st *state) path(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(st.cwd, filepath.FromSlash(p))
}

func noNeg(st *state, l Line, mode, name string) {
	if mode != "" {
		st.fatalf(l, "%s does not take '!'", name)
	}
}

func nargs(st *state, l Line, args []string, min, max int, usage string) {
	if len(args) < min || (max >= 0 && len(args) > max) {
		st.fatalf(l, "usage: %s", usage)
	}
}

func cmdCd(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "cd")
	nargs(st, l, args, 1, 1, "cd DIR")
	dir := st.path(args[0])
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		st.fatalf(l, "%s is not a directory", dir)
	}
	st.cwd = dir
	st.env["PWD"] = dir
}

func cmdChmod(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "chmod")
	nargs(st, l, args, 2, -1, "chmod MODE FILE...")
	perm, err := strconv.ParseUint(args[0], 8, 32)
	if err != nil {
		st.fatalf(l, "bad mode %q", args[0])
	}
	for _, f := range args[1:] {
		if err := os.Chmod(st.path(f), fs.FileMode(perm)); err != nil {
			st.fatalf(l, "%v", err)
		}
	}
}

// read returns the text of a file, or of the last stdout or stderr.
func (st *state) read(l Line, name string) []byte {
	switch name {
	case "stdout":
		if !st.ran {
			st.fatalf(l, "no exec has run yet")
		}
		return []byte(st.stdout)
	case "stderr":
		if !st.ran {
			st.fatalf(l, "no exec has run yet")
		}
		return []byte(st.stderr)
	}
	data, err := os.ReadFile(st.path(name))
	if err != nil {
		st.fatalf(l, "%v", err)
	}
	return data
}

func cmdCmp(st *state, l Line, mode string, args []string) {
	nargs(st, l, args, 2, 2, "cmp FILE1 FILE2")
	a, b := st.read(l, args[0]), st.read(l, args[1])
	same := bytes.Equal(a, b)
	if mode == "!" {
		if same {
			st.fatalf(l, "%s and %s are the same", args[0], args[1])
		}
		return
	}
	if !same {
		st.fatalf(l, "%s and %s differ:\n%s", args[0], args[1], diff(string(a), string(b)))
	}
}

func cmdCmpGo(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "cmpgo")
	nargs(st, l, args, 2, 2, "cmpgo GOT.go WANT.go")
	got, err := NormalizeGo(st.read(l, args[0]))
	if err != nil {
		st.fatalf(l, "%s: %v", args[0], err)
	}
	want, err := NormalizeGo(st.read(l, args[1]))
	if err != nil {
		st.fatalf(l, "%s (the expected file): %v", args[1], err)
	}
	if got != want {
		st.fatalf(l, "%s differs from %s (after gofmt and import sorting; - got, + want):\n%s", args[0], args[1], diff(got, want))
	}
}

func cmdCp(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "cp")
	nargs(st, l, args, 2, -1, "cp SRC... DST")
	dst := st.path(args[len(args)-1])
	srcs := args[:len(args)-1]
	dstIsDir := false
	if fi, err := os.Stat(dst); err == nil && fi.IsDir() {
		dstIsDir = true
	}
	if len(srcs) > 1 && !dstIsDir {
		st.fatalf(l, "cp: %s is not a directory", dst)
	}
	for _, src := range srcs {
		data := st.read(l, src)
		target := dst
		if dstIsDir {
			target = filepath.Join(dst, filepath.Base(src))
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			st.fatalf(l, "%v", err)
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			st.fatalf(l, "%v", err)
		}
	}
}

func cmdEnv(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "env")
	nargs(st, l, args, 1, -1, "env KEY=VALUE...")
	for _, a := range args {
		k, v, ok := strings.Cut(a, "=")
		if !ok || k == "" {
			st.fatalf(l, "usage: env KEY=VALUE")
		}
		st.env[k] = v
	}
}

func cmdUnset(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "unset")
	for _, a := range args {
		delete(st.env, a)
	}
}

func cmdExec(st *state, l Line, mode string, args []string) {
	nargs(st, l, args, 1, -1, "exec PROGRAM [ARGS...]")
	prog, err := lookPath(args[0], st.env["PATH"], st.cwd)
	if err != nil {
		st.fatalf(l, "%v", err)
	}
	timeout := st.e.ExecTimeout
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c := exec.CommandContext(ctx, prog, args[1:]...)
	c.Dir = st.cwd
	c.Env = st.envList()
	var out, errb bytes.Buffer
	c.Stdout, c.Stderr = &out, &errb
	if st.stdin != nil {
		c.Stdin = bytes.NewReader(st.stdin)
		st.stdin = nil
	}
	start := time.Now()
	runErr := c.Run()
	st.stdout, st.stderr, st.ran = out.String(), errb.String(), true
	st.exitCode = 0
	var ee *exec.ExitError
	switch {
	case runErr == nil:
	case errors.As(runErr, &ee):
		st.exitCode = ee.ExitCode()
	case ctx.Err() != nil:
		st.fatalf(l, "timed out after %s", timeout)
	default:
		st.fatalf(l, "%v", runErr)
	}
	st.t.Logf("%s:%d: > %s  [exit %d, %s]", st.s.Name, l.No, l.Text, st.exitCode, time.Since(start).Round(time.Millisecond))
	switch mode {
	case "":
		if st.exitCode != 0 {
			st.fatalf(l, "unexpected exit code %d", st.exitCode)
		}
	case "!":
		if st.exitCode == 0 {
			st.fatalf(l, "unexpected success")
		}
	}
}

func lookPath(name, pathEnv, cwd string) (string, error) {
	if strings.ContainsRune(name, '/') {
		p := name
		if !filepath.IsAbs(p) {
			p = filepath.Join(cwd, p)
		}
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
		return "", fmt.Errorf("program %s not found", name)
	}
	for _, dir := range filepath.SplitList(pathEnv) {
		p := filepath.Join(dir, name)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return p, nil
		}
	}
	return "", fmt.Errorf("program %q not found on PATH", name)
}

func cmdExists(st *state, l Line, mode string, args []string) {
	nargs(st, l, args, 1, -1, "exists PATH...")
	for _, p := range args {
		_, err := os.Stat(st.path(p))
		if mode == "!" && err == nil {
			st.fatalf(l, "%s exists", p)
		}
		if mode == "" && err != nil {
			st.fatalf(l, "%s does not exist", p)
		}
	}
}

func cmdExitCode(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "exitcode")
	nargs(st, l, args, 1, 1, "exitcode N")
	want, err := strconv.Atoi(args[0])
	if err != nil {
		st.fatalf(l, "bad exit code %q", args[0])
	}
	if !st.ran {
		st.fatalf(l, "no exec has run yet")
	}
	if st.exitCode != want {
		st.fatalf(l, "exit code is %d, want %d", st.exitCode, want)
	}
}

func cmdFixture(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "fixture")
	nargs(st, l, args, 1, 2, "fixture NAME [DIR]")
	src := filepath.Join(st.e.FixturesDir, args[0])
	if fi, err := os.Stat(src); err != nil || !fi.IsDir() {
		st.fatalf(l, "no fixture %q in %s", args[0], st.e.FixturesDir)
	}
	dst := args[0]
	if len(args) == 2 {
		dst = args[1]
	}
	if err := copyDir(src, st.path(dst), nil); err != nil {
		st.fatalf(l, "%v", err)
	}
}

func cmdGlasCopy(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "glascopy")
	nargs(st, l, args, 1, 1, "glascopy DIR")
	if st.e.GlasDir == "" {
		st.fatalf(l, "glascopy needs FLAGWIRE_GLAS_DIR; guard the line with [glas]")
	}
	skip := map[string]bool{".git": true, ".dev": true, "bin": true}
	if err := copyDir(st.e.GlasDir, st.path(args[0]), skip); err != nil {
		st.fatalf(l, "%v", err)
	}
}

func copyDir(src, dst string, skip map[string]bool) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() && skip[rel] {
			return filepath.SkipDir
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

func cmdGitInit(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "gitinit")
	nargs(st, l, args, 0, 1, "gitinit [DIR]")
	dir := st.cwd
	if len(args) == 1 {
		dir = st.path(args[0])
	}
	for _, a := range [][]string{
		{"init", "-q", "-b", "main"},
		{"add", "-A"},
		{"commit", "-q", "--no-verify", "-m", "initial"},
	} {
		c := exec.Command("git", a...)
		c.Dir = dir
		c.Env = st.envList()
		if out, err := c.CombinedOutput(); err != nil {
			st.fatalf(l, "git %s: %v\n%s", strings.Join(a, " "), err, out)
		}
	}
}

func cmdGrep(st *state, l Line, mode string, args []string) {
	nargs(st, l, args, 2, 2, "grep REGEXP FILE")
	re, err := regexp.Compile("(?m)" + args[0])
	if err != nil {
		st.fatalf(l, "%v", err)
	}
	data := st.read(l, args[1])
	found := re.Match(data)
	if mode == "!" && found {
		st.fatalf(l, "%s matches %q", args[1], args[0])
	}
	if mode == "" && !found {
		st.fatalf(l, "%s does not match %q:\n%s", args[1], args[0], clip(string(data)))
	}
}

func cmdJSONMatch(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "jsonmatch")
	nargs(st, l, args, 2, 2, "jsonmatch GOT WANT")
	if err := JSONMatch(st.read(l, args[0]), st.read(l, args[1])); err != nil {
		st.fatalf(l, "%s does not match %s: %v", args[0], args[1], err)
	}
}

func cmdMkdir(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "mkdir")
	for _, a := range args {
		if err := os.MkdirAll(st.path(a), 0o755); err != nil {
			st.fatalf(l, "%v", err)
		}
	}
}

func cmdMv(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "mv")
	nargs(st, l, args, 2, 2, "mv FROM TO")
	if err := os.Rename(st.path(args[0]), st.path(args[1])); err != nil {
		st.fatalf(l, "%v", err)
	}
}

func cmdRm(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "rm")
	for _, a := range args {
		if err := os.RemoveAll(st.path(a)); err != nil {
			st.fatalf(l, "%v", err)
		}
	}
}

func cmdSkip(st *state, l Line, mode string, args []string) {
	st.t.Skipf("%s: %s", st.s.Name, strings.Join(args, " "))
}

func cmdStdin(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "stdin")
	nargs(st, l, args, 1, 1, "stdin FILE")
	st.stdin = st.read(l, args[0])
}

func cmdStdout(st *state, l Line, mode string, args []string) {
	matchOutput(st, l, mode, args, "stdout", st.stdout)
}

func cmdStderr(st *state, l Line, mode string, args []string) {
	matchOutput(st, l, mode, args, "stderr", st.stderr)
}

func matchOutput(st *state, l Line, mode string, args []string, name, text string) {
	nargs(st, l, args, 1, 1, name+" REGEXP")
	if !st.ran {
		st.fatalf(l, "no exec has run yet")
	}
	re, err := regexp.Compile("(?m)" + args[0])
	if err != nil {
		st.fatalf(l, "%v", err)
	}
	found := re.MatchString(text)
	if mode == "!" && found {
		st.fatalf(l, "%s matches %q", name, args[0])
	}
	if mode == "" && !found {
		st.fatalf(l, "%s does not match %q", name, args[0])
	}
}

func cmdFakeLLM(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "fakellm")
	nargs(st, l, args, 1, 1, "fakellm RESPONSES.json")
	srv, err := startFakeLLM(st.read(l, args[0]), filepath.Join(st.work, ".fakellm"))
	if err != nil {
		st.fatalf(l, "%v", err)
	}
	st.t.Cleanup(srv.Close)
	st.env["FLAGWIRE_LLM_BASE_URL"] = srv.URL + "/v1"
	st.env["FLAGWIRE_LLM_MODEL"] = "fake-model"
	st.env["FLAGWIRE_LLM_API_KEY"] = "fake-llm-key"
	st.env["FAKELLM_LOG"] = filepath.Join(st.work, ".fakellm", "requests.jsonl")
}

func cmdFakeGitHub(st *state, l Line, mode string, args []string) {
	noNeg(st, l, mode, "fakegithub")
	nargs(st, l, args, 0, 0, "fakegithub")
	srv, err := startFakeGitHub(filepath.Join(st.work, ".fakegithub"))
	if err != nil {
		st.fatalf(l, "%v", err)
	}
	st.t.Cleanup(srv.Close)
	st.env["FLAGWIRE_GITHUB_API_URL"] = srv.URL
	st.env["FLAGWIRE_GITHUB_OAUTH_URL"] = srv.URL
	st.env["FAKEGITHUB_LOG"] = filepath.Join(st.work, ".fakegithub", "requests.jsonl")
}
