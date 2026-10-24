// Command followreport diffs a client-build follow's conformance goldens,
// talent tables and racial text, and names a verdict: clean, or review
// because something moved that only a person can judge.
//
// Exit 0 means clean, 3 means review, 1 means the inputs could not be read.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Exit codes.
const (
	exitClean  = 0
	exitError  = 1
	exitReview = 3
	goldenGlob = "*.golden.md"
	goldenExt  = ".golden.md"
	reportMode = 0o644
	// setsGolden is the item-set conformance golden, which is not a class table.
	setsGolden = "sets"
)

type options struct {
	before, after, prevTalents, talents, prevRaces, races string
	buildLog, diffstat, out                               string
}

func parseFlags(args []string, stderr io.Writer) (options, error) {
	var o options
	fs := flag.NewFlagSet("followreport", flag.ContinueOnError)
	fs.SetOutput(stderr)
	for name, dst := range map[string]*string{
		"before": &o.before, "after": &o.after,
		"prev-talents": &o.prevTalents, "talents": &o.talents,
		"prev-races": &o.prevRaces, "races": &o.races,
		"build-log": &o.buildLog, "generated-diffstat": &o.diffstat, "out": &o.out,
	} {
		fs.StringVar(dst, name, "", "")
	}
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	for name, v := range map[string]string{
		"before": o.before, "after": o.after, "prev-talents": o.prevTalents,
		"talents": o.talents, "prev-races": o.prevRaces, "races": o.races, "out": o.out,
	} {
		if v == "" {
			return o, fmt.Errorf("followreport: -%s is required", name)
		}
	}
	return o, nil
}

func loadGoldens(dir string) (map[string]Golden, error) {
	paths, err := filepath.Glob(filepath.Join(dir, goldenGlob))
	if err != nil {
		return nil, fmt.Errorf("followreport: %s: %w", dir, err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("followreport: %s: no goldens found", dir)
	}
	out := map[string]Golden{}
	for _, p := range paths {
		name := strings.TrimSuffix(filepath.Base(p), goldenExt)
		if name == setsGolden {
			continue
		}
		g, err := ParseGolden(p)
		if err != nil {
			return nil, err
		}
		out[name] = g
	}
	return out, nil
}

// readOptional returns a file's text, or "" when no path is given or the file is blank.
func readOptional(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("followreport: %s: %w", path, err)
	}
	return strings.TrimSpace(string(body)), nil
}

func buildReport(o options) (Report, error) {
	before, err := loadGoldens(o.before)
	if err != nil {
		return Report{}, err
	}
	after, err := loadGoldens(o.after)
	if err != nil {
		return Report{}, err
	}
	prevTalents, err := LoadTalents(o.prevTalents)
	if err != nil {
		return Report{}, err
	}
	talents, err := LoadTalents(o.talents)
	if err != nil {
		return Report{}, err
	}
	racials, err := DiffRacials(o.prevRaces, o.races)
	if err != nil {
		return Report{}, err
	}
	buildLog, err := readOptional(o.buildLog)
	if err != nil {
		return Report{}, err
	}
	stat, err := readOptional(o.diffstat)
	if err != nil {
		return Report{}, err
	}
	return assemble(before, after, DiffTalents(prevTalents, talents), racials, buildLog, stat), nil
}

func assemble(before, after map[string]Golden, t TalentDelta, racials []RacialChange, buildLog, stat string) Report {
	classes := map[string]ClassDelta{}
	for name, a := range after {
		if b, ok := before[name]; ok {
			classes[name] = CompareClass(b, a)
		}
	}
	namesBefore, namesAfter := goldenNames(before), goldenNames(after)
	r := Report{
		BuildLog:  buildLog,
		Classes:   classes,
		ClassesIn: DiffClassSets(namesBefore, namesAfter),
		Talents:   t,
		Racials:   racials,
	}
	if stat != "" {
		r.Diffstat = strings.Split(stat, "\n")
	}
	r.Verdict = Verdict(classes, t, racials, buildLog, namesBefore, namesAfter)
	return r
}

func goldenNames(m map[string]Golden) []string {
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func run(args []string, stderr io.Writer) int {
	o, err := parseFlags(args, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitError
	}
	report, err := buildReport(o)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitError
	}
	if err := os.WriteFile(o.out, []byte(Render(report)), reportMode); err != nil {
		fmt.Fprintf(stderr, "followreport: %s: %v\n", o.out, err)
		return exitError
	}
	if report.Verdict == verdictClean {
		return exitClean
	}
	return exitReview
}

func main() { os.Exit(run(os.Args[1:], os.Stderr)) }
