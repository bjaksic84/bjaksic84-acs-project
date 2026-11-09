// Command lock guards the files that define "done": the acceptance suite,
// the spec, the plan, the agent's instructions and the CI workflow.
// tests/LOCKED lists them.
//
//	go run ./cmd/lock verify            hashes match tests/SHA256SUMS
//	go run ./cmd/lock update            rewrite tests/SHA256SUMS (humans only)
//	go run ./cmd/lock staged            pre-commit: refuse staged locked files
//	go run ./cmd/lock audit FROM TO     every commit in FROM..TO that changes a
//	                                    locked file says why, in a
//	                                    "Locked-change:" trailer
//
// Run it from tests/. FLAGWIRE_ALLOW_LOCKED_EDIT=1 lets "staged" through; the
// commit still needs its trailer, which CI checks.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	sumsFile   = "tests/SHA256SUMS"
	lockedFile = "tests/LOCKED"
	trailer    = "Locked-change:"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	root, err := repoRoot()
	if err != nil {
		fail("%v", err)
	}
	switch os.Args[1] {
	case "verify":
		err = verify(root)
	case "update":
		err = update(root)
	case "staged":
		err = staged(root)
	case "audit":
		if len(os.Args) != 4 {
			usage()
		}
		err = audit(root, os.Args[2], os.Args[3])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "lock:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: lock verify | update | staged | audit FROM TO")
	os.Exit(2)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "lock: "+format+"\n", args...)
	os.Exit(2)
}

func repoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git work tree")
	}
	return strings.TrimSpace(string(out)), nil
}

func git(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

// patterns reads tests/LOCKED: one path per line; a trailing slash means a
// whole directory.
func patterns(root string) ([]string, error) {
	f, err := os.Open(filepath.Join(root, lockedFile))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var pats []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		pats = append(pats, line)
	}
	return pats, sc.Err()
}

func isLocked(pats []string, path string) bool {
	if path == sumsFile {
		return true
	}
	for _, p := range pats {
		if strings.HasSuffix(p, "/") && strings.HasPrefix(path, p) || path == p {
			return true
		}
	}
	return false
}

// lockedFiles lists the tracked and untracked (not ignored) locked files.
func lockedFiles(root string) ([]string, error) {
	pats, err := patterns(root)
	if err != nil {
		return nil, err
	}
	out, err := git(root, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var files []string
	for _, p := range strings.Split(out, "\x00") {
		if p == "" || p == sumsFile || seen[p] || !isLocked(pats, p) {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			continue // deleted in the work tree
		}
		seen[p] = true
		files = append(files, p)
	}
	sort.Strings(files)
	return files, nil
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func current(root string) (map[string]string, error) {
	files, err := lockedFiles(root)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, f := range files {
		h, err := hashFile(filepath.Join(root, f))
		if err != nil {
			return nil, err
		}
		m[f] = h
	}
	return m, nil
}

func recorded(root string) (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(root, sumsFile))
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	for i, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		h, p, ok := strings.Cut(line, "  ")
		if !ok || len(h) != 64 {
			return nil, fmt.Errorf("%s:%d: malformed line", sumsFile, i+1)
		}
		m[p] = h
	}
	return m, nil
}

func verify(root string) error {
	want, err := recorded(root)
	if err != nil {
		return err
	}
	got, err := current(root)
	if err != nil {
		return err
	}
	var problems []string
	for p, h := range want {
		switch g, ok := got[p]; {
		case !ok:
			problems = append(problems, "missing: "+p)
		case g != h:
			problems = append(problems, "changed: "+p)
		}
	}
	for p := range got {
		if _, ok := want[p]; !ok {
			problems = append(problems, "not in SHA256SUMS: "+p)
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("locked files differ from %s:\n  %s\nA locked file changes only through a logged intervention (docs/INTERVENTIONS.md).", sumsFile, strings.Join(problems, "\n  "))
	}
	fmt.Printf("lock: %d locked files match %s\n", len(want), sumsFile)
	return nil
}

func update(root string) error {
	got, err := current(root)
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(got))
	for p := range got {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var b strings.Builder
	for _, p := range paths {
		fmt.Fprintf(&b, "%s  %s\n", got[p], p)
	}
	if err := os.WriteFile(filepath.Join(root, sumsFile), []byte(b.String()), 0o644); err != nil {
		return err
	}
	fmt.Printf("lock: wrote %d hashes to %s. Commit with a %q trailer.\n", len(paths), sumsFile, trailer)
	return nil
}

func staged(root string) error {
	pats, err := patterns(root)
	if err != nil {
		return err
	}
	out, err := git(root, "diff", "--cached", "--name-only", "-z")
	if err != nil {
		return err
	}
	var hits []string
	for _, p := range strings.Split(out, "\x00") {
		if p != "" && isLocked(pats, p) {
			hits = append(hits, p)
		}
	}
	if len(hits) == 0 {
		return nil
	}
	if os.Getenv("FLAGWIRE_ALLOW_LOCKED_EDIT") == "1" {
		fmt.Fprintf(os.Stderr, "lock: committing locked files (allowed by FLAGWIRE_ALLOW_LOCKED_EDIT=1); add a %q trailer:\n  %s\n", trailer, strings.Join(hits, "\n  "))
		return nil
	}
	return fmt.Errorf("this commit changes locked files:\n  %s\nThe coding agent never edits these. A person or the reviewing model may, with FLAGWIRE_ALLOW_LOCKED_EDIT=1,\n%q in the commit message, and an entry in docs/INTERVENTIONS.md.", strings.Join(hits, "\n  "), trailer)
}

func audit(root, from, to string) error {
	pats, err := patterns(root)
	if err != nil {
		return err
	}
	out, err := git(root, "rev-list", "--reverse", from+".."+to)
	if err != nil {
		return err
	}
	var bad []string
	n := 0
	for _, sha := range strings.Fields(out) {
		n++
		files, err := git(root, "diff-tree", "--no-commit-id", "--name-only", "-r", "-z", "--root", sha)
		if err != nil {
			return err
		}
		touched := false
		for _, p := range strings.Split(files, "\x00") {
			if p != "" && isLocked(pats, p) {
				touched = true
				break
			}
		}
		if !touched {
			continue
		}
		msg, err := git(root, "log", "-1", "--format=%B", sha)
		if err != nil {
			return err
		}
		if !hasTrailer(msg) {
			subject, _, _ := strings.Cut(msg, "\n")
			bad = append(bad, sha[:12]+" "+subject)
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("these commits change locked files without a %q trailer:\n  %s", trailer, strings.Join(bad, "\n  "))
	}
	fmt.Printf("lock: %d commits audited\n", n)
	return nil
}

func hasTrailer(msg string) bool {
	for _, line := range strings.Split(msg, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), trailer) {
			return true
		}
	}
	return false
}
