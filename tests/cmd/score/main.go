// Command score summarises an acceptance run: a pass table by feature and
// level, a CSV row for docs/metrics/score.csv, and the ratchet check.
//
//	go run ./cmd/score -results results.json [-csv FILE -commit SHA]
//	    [-ratchet FILE [-update-ratchet]] [-md FILE]
//
// The ratchet file lists tests that passed once. -ratchet fails if one of
// them now fails; -update-ratchet adds newly passing tests. Nothing is ever
// removed from it automatically.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

type result struct {
	Name    string  `json:"name"`
	Feature string  `json:"feature"`
	Level   string  `json:"level"`
	Tier    int     `json:"tier"`
	Status  string  `json:"status"`
	Title   string  `json:"title"`
	Outcome string  `json:"outcome"`
	Seconds float64 `json:"seconds"`
	Failure string  `json:"failure"`
}

var levels = []string{"basic", "standard", "advanced"}

func main() {
	resultsPath := flag.String("results", "", "results JSON written by the acceptance run (FLAGWIRE_RESULTS)")
	csvPath := flag.String("csv", "", "append a row to this CSV file")
	commit := flag.String("commit", "", "commit the run tested, for the CSV row")
	ratchet := flag.String("ratchet", "", "fail if a test listed in this file does not pass")
	update := flag.Bool("update-ratchet", false, "add every passing locked test to the -ratchet file")
	mdPath := flag.String("md", "", "also write the summary as Markdown to this file")
	flag.Parse()
	if *resultsPath == "" {
		fail("-results is required")
	}
	data, err := os.ReadFile(*resultsPath)
	if err != nil {
		fail("%v", err)
	}
	var all []result
	if err := json.Unmarshal(data, &all); err != nil {
		fail("%s: %v", *resultsPath, err)
	}
	if len(all) == 0 {
		fail("%s holds no results; did the acceptance run start?", *resultsPath)
	}
	summary := summarise(all)
	fmt.Print(summary)
	if *mdPath != "" {
		md := "```\n" + summary + "```\n"
		if err := os.WriteFile(*mdPath, []byte(md), 0o644); err != nil {
			fail("%v", err)
		}
	}
	if *csvPath != "" {
		if err := appendCSV(*csvPath, *commit, all); err != nil {
			fail("%v", err)
		}
	}
	if *ratchet != "" {
		if err := checkRatchet(*ratchet, all, *update); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "score: "+format+"\n", args...)
	os.Exit(2)
}

type count struct{ pass, total int }

func (c count) String() string {
	if c.total == 0 {
		return "-"
	}
	return fmt.Sprintf("%d/%d", c.pass, c.total)
}

func summarise(all []result) string {
	var b strings.Builder
	type row struct {
		feature string
		tier    int
		by      map[string]count
		total   count
	}
	rows := map[string]*row{}
	tiers := map[int]*count{}
	var locked, skipped, drafts count
	var failures []result
	for _, r := range all {
		if r.Status == "draft" {
			drafts.total++
			if r.Outcome == "pass" {
				drafts.pass++
			}
			continue
		}
		rw := rows[r.Feature]
		if rw == nil {
			rw = &row{feature: r.Feature, tier: r.Tier, by: map[string]count{}}
			rows[r.Feature] = rw
		}
		c := rw.by[r.Level]
		c.total++
		rw.total.total++
		if tiers[r.Tier] == nil {
			tiers[r.Tier] = &count{}
		}
		tiers[r.Tier].total++
		locked.total++
		switch r.Outcome {
		case "pass":
			c.pass++
			rw.total.pass++
			tiers[r.Tier].pass++
			locked.pass++
		case "skip":
			skipped.total++
		default:
			failures = append(failures, r)
		}
		rw.by[r.Level] = c
	}
	list := make([]*row, 0, len(rows))
	for _, r := range rows {
		list = append(list, r)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].tier != list[j].tier {
			return list[i].tier < list[j].tier
		}
		return list[i].feature < list[j].feature
	})
	fmt.Fprintf(&b, "flagwire acceptance score, locked tests\n\n")
	fmt.Fprintf(&b, "%-16s %-4s %-8s %-8s %-8s %s\n", "feature", "tier", "basic", "standard", "advanced", "total")
	for _, r := range list {
		fmt.Fprintf(&b, "%-16s %-4d %-8s %-8s %-8s %s\n", r.feature, r.tier, r.by["basic"], r.by["standard"], r.by["advanced"], r.total)
	}
	b.WriteString("\n")
	tierKeys := make([]int, 0, len(tiers))
	for k := range tiers {
		tierKeys = append(tierKeys, k)
	}
	sort.Ints(tierKeys)
	for _, k := range tierKeys {
		fmt.Fprintf(&b, "tier %d: %s\n", k, tiers[k])
	}
	pct := 0.0
	if locked.total > 0 {
		pct = 100 * float64(locked.pass) / float64(locked.total)
	}
	fmt.Fprintf(&b, "all locked: %s (%.1f%%)\n", locked, pct)
	fmt.Fprintf(&b, "skipped: %d   drafts (not counted): %d, %d passing\n", skipped.total, drafts.total, drafts.pass)
	if len(failures) > 0 {
		fmt.Fprintf(&b, "\nfailing (%d):\n", len(failures))
		for i, f := range failures {
			if i == 25 {
				fmt.Fprintf(&b, "  ... and %d more\n", len(failures)-25)
				break
			}
			msg := f.Failure
			if len(msg) > 110 {
				msg = msg[:110] + "…"
			}
			fmt.Fprintf(&b, "  %-44s %s\n", f.Name, msg)
		}
	}
	return b.String()
}

const csvHeader = "date,commit,passed,failed,skipped,locked,tier1_passed,tier1_total,tier2_passed,tier2_total,tier3_passed,tier3_total"

func appendCSV(path, commit string, all []result) error {
	var passed, failed, skipped, locked int
	tp := map[int]int{}
	tt := map[int]int{}
	for _, r := range all {
		if r.Status == "draft" {
			continue
		}
		locked++
		tt[r.Tier]++
		switch r.Outcome {
		case "pass":
			passed++
			tp[r.Tier]++
		case "skip":
			skipped++
		default:
			failed++
		}
	}
	_, err := os.Stat(path)
	newFile := os.IsNotExist(err)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if newFile {
		fmt.Fprintln(f, csvHeader)
	}
	_, err = fmt.Fprintf(f, "%s,%s,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
		time.Now().UTC().Format(time.RFC3339), commit, passed, failed, skipped, locked,
		tp[1], tt[1], tp[2], tt[2], tp[3], tt[3])
	return err
}

func checkRatchet(path string, all []result, update bool) error {
	listed := map[string]bool{}
	if f, err := os.Open(path); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if line := strings.TrimSpace(sc.Text()); line != "" && !strings.HasPrefix(line, "#") {
				listed[line] = true
			}
		}
		f.Close()
	} else if !os.IsNotExist(err) {
		return err
	}
	byName := map[string]result{}
	for _, r := range all {
		byName[r.Name] = r
	}
	var broken []string
	for name := range listed {
		r, ok := byName[name]
		switch {
		case !ok:
			broken = append(broken, name+" (no longer exists)")
		case r.Outcome == "skip":
			fmt.Fprintf(os.Stderr, "ratchet: %s was skipped, not checked\n", name)
		case r.Outcome != "pass":
			broken = append(broken, name)
		}
	}
	if len(broken) > 0 {
		sort.Strings(broken)
		return fmt.Errorf("ratchet: these tests passed before and fail now:\n  %s", strings.Join(broken, "\n  "))
	}
	if !update {
		return nil
	}
	for _, r := range all {
		if r.Status != "draft" && r.Outcome == "pass" {
			listed[r.Name] = true
		}
	}
	names := make([]string, 0, len(listed))
	for n := range listed {
		names = append(names, n)
	}
	sort.Strings(names)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := io.Writer(f)
	fmt.Fprintln(w, "# Acceptance tests that have passed. make score keeps every one passing;")
	fmt.Fprintln(w, "# make score-update adds new ones. Removing a line needs a logged intervention.")
	for _, n := range names {
		fmt.Fprintln(w, n)
	}
	return nil
}
