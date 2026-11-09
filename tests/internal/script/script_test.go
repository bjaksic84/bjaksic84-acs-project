package script

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeTB records a script's failure instead of failing the real test.
type fakeTB struct {
	*testing.T
	failed  string
	skipped string
}

type stop struct{}

func (f *fakeTB) Fatalf(format string, args ...any) {
	f.failed = fmt.Sprintf(format, args...)
	panic(stop{})
}

func (f *fakeTB) Skipf(format string, args ...any) {
	f.skipped = fmt.Sprintf(format, args...)
	panic(stop{})
}

// run runs a script and returns its failure message ("" if it passed).
func run(t *testing.T, archive string) (failed, skipped string) {
	t.Helper()
	s, err := Parse("selftest", []byte("# feature: selftest\n# level: basic\n# tier: 1\n# title: self test\n"+archive))
	if err != nil {
		t.Fatal(err)
	}
	e := &Engine{
		BinDir:      t.TempDir(),
		FixturesDir: t.TempDir(),
		GoldenDir:   t.TempDir(),
		HostEnv:     map[string]string{"PATH": os.Getenv("PATH")},
		Conditions:  map[string]bool{"net": false, "glas": false},
	}
	f := &fakeTB{T: t}
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(stop); !ok {
					panic(r)
				}
			}
		}()
		e.Run(f, s)
	}()
	return f.failed, f.skipped
}

func TestPassingScript(t *testing.T) {
	failed, _ := run(t, `
exec sh -c 'echo hello world; echo oops >&2'
stdout '^hello world$'
! stdout goodbye
stderr oops
exists a.txt dir/b.txt
! exists missing.txt
cmp a.txt dir/b.txt
grep '^same$' a.txt
cd dir
exists b.txt
env GREETING='hi there'
exec sh -c 'echo $GREETING'
stdout '^hi there$'
! exec sh -c 'exit 3'
exitcode 3
cp stdout out.txt
grep '^$' out.txt
mkdir x/y
exists x/y
-- a.txt --
same
-- dir/b.txt --
same
`)
	if failed != "" {
		t.Fatalf("script failed:\n%s", failed)
	}
}

func TestFailingStepsAreReported(t *testing.T) {
	for name, archive := range map[string]string{
		"exit":      "exec sh -c 'exit 1'\n",
		"stdout":    "exec sh -c 'echo a'\nstdout b\n",
		"negstdout": "exec sh -c 'echo a'\n! stdout a\n",
		"exists":    "exists nope\n",
		"cmp":       "cmp a b\n-- a --\nx\n-- b --\ny\n",
		"unknown":   "frobnicate\n",
		"condition": "[nosuch] exec true\n",
		"success":   "! exec true\n",
	} {
		t.Run(name, func(t *testing.T) {
			failed, _ := run(t, archive)
			if failed == "" {
				t.Fatal("script passed, want a failure")
			}
			if !strings.Contains(failed, "selftest:") {
				t.Errorf("failure does not name the script and line: %s", failed)
			}
		})
	}
}

func TestConditionsAndSkip(t *testing.T) {
	failed, skipped := run(t, "[net] exec false\n[!net] skip 'needs the network'\nexec false\n")
	if failed != "" {
		t.Fatalf("failed: %s", failed)
	}
	if !strings.Contains(skipped, "needs the network") {
		t.Fatalf("skipped = %q", skipped)
	}
}

func TestQuotesAndExpansion(t *testing.T) {
	w, err := words(`a 'b c' 'it''s' $X${Y}z '$X'`, func(n string) string { return map[string]string{"X": "1", "Y": "2"}[n] })
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b c", "it's", "12z", "$X"}
	if fmt.Sprint(w) != fmt.Sprint(want) {
		t.Fatalf("words = %q, want %q", w, want)
	}
	if _, err := words("'open", nil); err == nil {
		t.Fatal("unterminated quote accepted")
	}
}

func TestParseRejectsBadHeaders(t *testing.T) {
	for _, h := range []string{
		"# level: basic\n# tier: 1\n# title: x\n",
		"# feature: f\n# level: easy\n# tier: 1\n# title: x\n",
		"# feature: f\n# level: basic\n# tier: 9\n# title: x\n",
		"# feature: f\n# level: basic\n# tier: 1\n",
		"# feature: f\n# level: basic\n# tier: 1\n# title: x\n# status: maybe\n",
	} {
		if _, err := Parse("x", []byte(h)); err == nil {
			t.Errorf("header accepted:\n%s", h)
		}
	}
}

func TestNormalizeGo(t *testing.T) {
	a := "package p\n\nimport (\n\t\"fmt\"\n\n\t\"example.com/m/internal/flags\"\n)\n\nfunc f() {\n  fmt.Println(flags.Admin)\n}\n"
	b := "package p\n\nimport (\n\t\"example.com/m/internal/flags\"\n\t\"fmt\"\n)\n\nfunc f() {\n\tfmt.Println(flags.Admin)\n}\n"
	na, err := NormalizeGo([]byte(a))
	if err != nil {
		t.Fatal(err)
	}
	nb, err := NormalizeGo([]byte(b))
	if err != nil {
		t.Fatal(err)
	}
	if na != nb {
		t.Fatalf("not equal after normalising:\n%s\n---\n%s", na, nb)
	}
	c := strings.Replace(b, "flags.Admin", "flags.Reports", 1)
	nc, _ := NormalizeGo([]byte(c))
	if nc == nb {
		t.Fatal("different code normalised to the same text")
	}
	if _, err := NormalizeGo([]byte("package p\nfunc (")); err == nil {
		t.Fatal("broken Go accepted")
	}
}

func TestJSONMatch(t *testing.T) {
	got := `{"schema":"x/v1","n":3,"list":[{"id":"a","line":1,"extra":true},{"id":"b","line":2}],"name":"route:GET /x"}`
	for _, want := range []string{
		`{"schema":"x/v1"}`,
		`{"n":3.0}`,
		`{"list":[{"id":"a"},{"id":"b","line":2}]}`,
		`{"list":["...contains",{"id":"b"}]}`,
		`{"list":"...any","name":"re:^route:GET "}`,
	} {
		if err := JSONMatch([]byte(got), []byte(want)); err != nil {
			t.Errorf("%s: %v", want, err)
		}
	}
	for _, want := range []string{
		`{"missing":1}`,
		`{"list":[{"id":"a"}]}`,
		`{"list":[{"id":"b"},{"id":"a"}]}`,
		`{"list":["...contains",{"id":"c"}]}`,
		`{"list":["...contains",{"id":"a"},{"id":"a"}]}`,
		`{"name":"re:^job:"}`,
		`{"n":"3"}`,
	} {
		if err := JSONMatch([]byte(got), []byte(want)); err == nil {
			t.Errorf("%s matched, want a mismatch", want)
		}
	}
}

func TestFakeLLM(t *testing.T) {
	dir := t.TempDir()
	srv, err := startFakeLLM([]byte(`{"responses":["{\"features\":[]}",{"status":503}]}`), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	post := func() (int, string) {
		req, _ := http.NewRequest("POST", srv.URL+"/v1/chat/completions", strings.NewReader(`{"model":"fake-model","messages":[]}`))
		req.Header.Set("Authorization", "Bearer k")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(body)
	}
	code, body := post()
	var reply struct {
		Choices []struct{ Message struct{ Content string } }
	}
	if code != 200 || json.Unmarshal([]byte(body), &reply) != nil || reply.Choices[0].Message.Content != `{"features":[]}` {
		t.Fatalf("first reply %d %s", code, body)
	}
	if code, _ := post(); code != 503 {
		t.Fatalf("second reply %d, want 503", code)
	}
	if code, _ := post(); code != 500 {
		t.Fatalf("third reply %d, want 500", code)
	}
	log, _ := os.ReadFile(filepath.Join(dir, "requests.jsonl"))
	if strings.Count(string(log), "\n") != 3 || !strings.Contains(string(log), `"authorization":"Bearer k"`) {
		t.Fatalf("log:\n%s", log)
	}
}

func TestFakeGitHub(t *testing.T) {
	dir := t.TempDir()
	srv, err := startFakeGitHub(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	do := func(method, path, token, body string) (int, map[string]any) {
		req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var m map[string]any
		json.NewDecoder(resp.Body).Decode(&m)
		return resp.StatusCode, m
	}
	if code, m := do("GET", "/user", "gho_x", ""); code != 200 || m["login"] != "octo-tester" {
		t.Fatalf("GET /user: %d %v", code, m)
	}
	if code, _ := do("GET", "/user", "bad-token", ""); code != 401 {
		t.Fatalf("bad token: %d", code)
	}
	if code, m := do("POST", "/repos/acme/shop/pulls", "gho_x", `{"title":"t","head":"flagwire/admin","base":"main"}`); code != 201 || m["html_url"] != "https://github.com/acme/shop/pull/7" {
		t.Fatalf("create PR: %d %v", code, m)
	}
	if code, _ := do("POST", "/repos/acme/shop/pulls", "gho_x", `{"title":"t"}`); code != 422 {
		t.Fatalf("PR without head: %d", code)
	}
	form := url.Values{"client_id": {"c"}, "device_code": {"dc-123"}}.Encode()
	if code, m := do("POST", "/login/device/code", "", form); code != 200 || m["user_code"] != "WDJB-MJHT" {
		t.Fatalf("device code: %d %v", code, m)
	}
	if _, m := do("POST", "/login/oauth/access_token", "", form); m["error"] != "authorization_pending" {
		t.Fatalf("first poll: %v", m)
	}
	if _, m := do("POST", "/login/oauth/access_token", "", form); m["access_token"] != "gho_devicetoken" {
		t.Fatalf("second poll: %v", m)
	}
}

func TestLint(t *testing.T) {
	dir := t.TempDir()
	e := &Engine{FixturesDir: dir, GoldenDir: dir, Conditions: map[string]bool{"net": true}}
	s, err := Parse("x/basic-lint", []byte(`# feature: x
# level: basic
# tier: 1
# title: a broken script

[nosuch] exec true
frobnicate
? cp a b
stdout '(unclosed'
fixture missing
cp ../absent.txt here.txt
cmp $GOLDEN/nothing.go here.txt
-- bad.json --
{"a": }
`))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"unknown condition [nosuch]",
		`unknown command "frobnicate"`,
		"'?' only goes with exec",
		"missing closing )",
		`no fixture "missing"`,
		"absent.txt is not in the archive",
		"nothing.go does not exist",
		"bad.json is not valid JSON",
	}
	var got []string
	for _, err := range e.Lint(s) {
		got = append(got, err.Error())
	}
	all := strings.Join(got, "\n")
	for _, w := range want {
		if !strings.Contains(all, w) {
			t.Errorf("Lint did not report %q; it reported:\n%s", w, all)
		}
	}
}
