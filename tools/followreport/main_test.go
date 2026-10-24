package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func baseArgs(t *testing.T, before, after, talents string) ([]string, string) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "report.md")
	td := func(n string) string { return filepath.Join("testdata", n) }
	return []string{
		"-before", td(before), "-after", td(after),
		"-prev-talents", td("talents-prev"), "-talents", td(talents),
		"-prev-races", td("races/prev.json"), "-races", td("races/same.json"),
		"-build-log", "", "-generated-diffstat", "", "-out", out,
	}, out
}

func TestRunClean(t *testing.T) {
	args, out := baseArgs(t, "two-classes", "two-classes", "talents-same")
	if code := run(args, io.Discard); code != exitClean {
		t.Fatalf("exit = %d", code)
	}
	body, err := os.ReadFile(out)
	if err != nil || !strings.HasPrefix(string(body), "verdict: clean\n") {
		t.Fatalf("report = %q, err = %v", body, err)
	}
}

func TestRunReviewOnTalentChange(t *testing.T) {
	args, out := baseArgs(t, "two-classes", "two-classes", "talents-cur")
	if code := run(args, io.Discard); code != exitReview {
		t.Fatalf("exit = %d", code)
	}
	body, _ := os.ReadFile(out)
	if !strings.HasPrefix(string(body), "verdict: review\n") {
		t.Fatalf("report = %q", body)
	}
}

func TestRunReviewOnClassGone(t *testing.T) {
	args, _ := baseArgs(t, "two-classes", "one-class", "talents-same")
	if code := run(args, io.Discard); code != exitReview {
		t.Fatalf("exit = %d", code)
	}
}

func TestRunBuildLogAndDiffstat(t *testing.T) {
	args, out := baseArgs(t, "two-classes", "two-classes", "talents-same")
	dir := t.TempDir()
	log, stat := filepath.Join(dir, "build.log"), filepath.Join(dir, "stat.txt")
	if err := os.WriteFile(log, []byte("boom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stat, []byte("2\t1\tsim/a.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	args[13], args[15] = log, stat
	if code := run(args, io.Discard); code != exitReview {
		t.Fatalf("exit = %d", code)
	}
	body, _ := os.ReadFile(out)
	if !strings.Contains(string(body), "## Build") || !strings.Contains(string(body), "sim/a.go: +2 -1") {
		t.Fatalf("report = %s", body)
	}
}

func TestRunEmptyBuildLogFileIsClean(t *testing.T) {
	args, _ := baseArgs(t, "two-classes", "two-classes", "talents-same")
	empty := filepath.Join(t.TempDir(), "build.log")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	args[13] = empty
	if code := run(args, io.Discard); code != exitClean {
		t.Fatalf("exit = %d", code)
	}
}

func TestRunParseErrorsExitOne(t *testing.T) {
	args, _ := baseArgs(t, "two-classes", "no-such", "talents-same")
	if code := run(args, io.Discard); code != exitError {
		t.Fatalf("missing dir: exit = %d", code)
	}
	if code := run([]string{"-before", "x"}, io.Discard); code != exitError {
		t.Fatalf("missing flags: exit = %d", code)
	}
	if code := run([]string{"-nope"}, io.Discard); code != exitError {
		t.Fatalf("bad flag: exit = %d", code)
	}
}
