package script

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// requestLog appends one JSON line per request to a file.
type requestLog struct {
	mu   sync.Mutex
	path string
}

func newRequestLog(dir string) (*requestLog, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &requestLog{path: filepath.Join(dir, "requests.jsonl")}, nil
}

func (l *requestLog) add(r *http.Request, body []byte) {
	entry := map[string]any{
		"method":        r.Method,
		"path":          r.URL.Path,
		"authorization": r.Header.Get("Authorization"),
		"accept":        r.Header.Get("Accept"),
	}
	var js any
	switch {
	case len(body) == 0:
	case json.Unmarshal(body, &js) == nil:
		entry["body"] = js
	default:
		if form, err := url.ParseQuery(string(body)); err == nil {
			flat := map[string]string{}
			for k, v := range form {
				flat[k] = strings.Join(v, ",")
			}
			entry["body"] = flat
		} else {
			entry["body"] = string(body)
		}
	}
	line, _ := json.Marshal(entry)
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(line, '\n'))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// startFakeLLM serves an OpenAI-compatible chat API that answers with the
// scripted responses in order. The responses file is
//
//	{"responses": ["<message content>", {"status": 503}, ...]}
//
// where a string is the assistant's message content and an object is an HTTP
// error with that status.
func startFakeLLM(spec []byte, logDir string) (*httptest.Server, error) {
	var cfg struct {
		Responses []json.RawMessage `json:"responses"`
	}
	if err := json.Unmarshal(spec, &cfg); err != nil {
		return nil, fmt.Errorf("fakellm responses: %v", err)
	}
	log, err := newRequestLog(logDir)
	if err != nil {
		return nil, err
	}
	var mu sync.Mutex
	next := 0
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/models", func(w http.ResponseWriter, r *http.Request) {
		log.add(r, nil)
		writeJSON(w, 200, map[string]any{"object": "list", "data": []any{map[string]any{"id": "fake-model", "object": "model"}}})
	})
	mux.HandleFunc("POST /v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		log.add(r, body)
		mu.Lock()
		i := next
		next++
		mu.Unlock()
		if i >= len(cfg.Responses) {
			writeJSON(w, 500, map[string]any{"error": map[string]any{"message": "fakellm: no more scripted responses"}})
			return
		}
		raw := cfg.Responses[i]
		var content string
		if json.Unmarshal(raw, &content) != nil {
			var e struct {
				Status int `json:"status"`
			}
			json.Unmarshal(raw, &e)
			if e.Status == 0 {
				e.Status = 500
			}
			writeJSON(w, e.Status, map[string]any{"error": map[string]any{"message": "fakellm: scripted error"}})
			return
		}
		writeJSON(w, 200, map[string]any{
			"id":      fmt.Sprintf("chatcmpl-fake-%d", i+1),
			"object":  "chat.completion",
			"created": 0,
			"model":   "fake-model",
			"choices": []any{map[string]any{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": content},
				"finish_reason": "stop",
			}},
			"usage": map[string]any{"prompt_tokens": 100, "completion_tokens": 50, "total_tokens": 150},
		})
	})
	return httptest.NewServer(mux), nil
}

// startFakeGitHub serves the parts of GitHub's REST API and OAuth device
// flow that flagwire uses. Any token is accepted except an empty one or one
// that starts with "bad". Pull requests are numbered from 7 and issues from
// 101; each one's body is also written to pull-<n>.md or issue-<n>.md in
// logDir, so a script can cmp it.
func startFakeGitHub(logDir string) (*httptest.Server, error) {
	log, err := newRequestLog(logDir)
	if err != nil {
		return nil, err
	}
	var mu sync.Mutex
	polls := 0
	pull := 6
	issue := 100
	saveBody := func(name, body string) {
		os.WriteFile(filepath.Join(logDir, name), []byte(body), 0o644)
	}
	authed := func(w http.ResponseWriter, r *http.Request) bool {
		h := r.Header.Get("Authorization")
		tok := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(h, "Bearer "), "token "))
		if h == "" || tok == "" || strings.HasPrefix(tok, "bad") {
			writeJSON(w, 401, map[string]any{"message": "Bad credentials"})
			return false
		}
		return true
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login/device/code", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		log.add(r, body)
		writeJSON(w, 200, map[string]any{
			"device_code":      "dc-123",
			"user_code":        "WDJB-MJHT",
			"verification_uri": "https://github.com/login/device",
			"expires_in":       900,
			"interval":         1,
		})
	})
	mux.HandleFunc("POST /login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		log.add(r, body)
		mu.Lock()
		polls++
		n := polls
		mu.Unlock()
		// GitHub answers 200 with an "error" field while the user hasn't
		// entered the code yet.
		if n == 1 {
			writeJSON(w, 200, map[string]any{"error": "authorization_pending"})
			return
		}
		writeJSON(w, 200, map[string]any{"access_token": "gho_devicetoken", "token_type": "bearer", "scope": "repo"})
	})
	mux.HandleFunc("GET /user", func(w http.ResponseWriter, r *http.Request) {
		log.add(r, nil)
		if authed(w, r) {
			writeJSON(w, 200, map[string]any{"login": "octo-tester", "id": 1})
		}
	})
	mux.HandleFunc("GET /repos/{owner}/{repo}", func(w http.ResponseWriter, r *http.Request) {
		log.add(r, nil)
		if authed(w, r) {
			writeJSON(w, 200, map[string]any{"full_name": r.PathValue("owner") + "/" + r.PathValue("repo"), "default_branch": "main"})
		}
	})
	mux.HandleFunc("POST /repos/{owner}/{repo}/pulls", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		log.add(r, body)
		if !authed(w, r) {
			return
		}
		var pr struct {
			Head, Base, Title, Body string
		}
		if json.Unmarshal(body, &pr) != nil || pr.Head == "" || pr.Base == "" || pr.Title == "" {
			writeJSON(w, 422, map[string]any{"message": "Validation Failed"})
			return
		}
		mu.Lock()
		pull++
		n := pull
		mu.Unlock()
		saveBody(fmt.Sprintf("pull-%d.md", n), pr.Body)
		full := r.PathValue("owner") + "/" + r.PathValue("repo")
		writeJSON(w, 201, map[string]any{"number": n, "state": "open", "html_url": fmt.Sprintf("https://github.com/%s/pull/%d", full, n)})
	})
	mux.HandleFunc("POST /repos/{owner}/{repo}/issues", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		log.add(r, body)
		if !authed(w, r) {
			return
		}
		var is struct {
			Body string
		}
		json.Unmarshal(body, &is)
		mu.Lock()
		issue++
		n := issue
		mu.Unlock()
		saveBody(fmt.Sprintf("issue-%d.md", n), is.Body)
		full := r.PathValue("owner") + "/" + r.PathValue("repo")
		writeJSON(w, 201, map[string]any{"number": n, "state": "open", "html_url": fmt.Sprintf("https://github.com/%s/issues/%d", full, n)})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		log.add(r, body)
		writeJSON(w, 404, map[string]any{"message": "Not Found"})
	})
	return httptest.NewServer(mux), nil
}
