#!/usr/bin/env bash
# follow.sh regenerates every client-derived artifact of the engine from one
# site build directory, in dependency order, and writes the follow report.
#
#   make follow SITE_DIR=/path/to/forever BUILD=1.60.1.70532
#   tools/follow.sh /path/to/forever 1.60.1.70532
#
# This script is the only definition of the follow sequence: the follow
# workflow, a person and the plan's local checks all call it. It refuses a
# dirty checkout, stops at the first generator that fails, and writes only
# the paths in allowlist() below. If the regenerated engine does not compile,
# it keeps the goldens as they were and the report carries the compiler
# output. Outputs (both untracked, dist/ is gitignored):
#
#   dist/FOLLOW-REPORT.md  what moved, opening with "verdict: clean|review"
#   dist/FOLLOW-PATHS      the allowlisted paths this run changed, one per line
#
# Exit status: 0 verdict clean, 3 verdict review, 1 any failure. make turns
# every non-zero status into 2, so a caller that needs 3 runs the script.
set -euo pipefail

readonly CLASSES=(druid hunter mage paladin priest rogue shaman warlock warrior)
readonly EXIT_FAILURE=1
readonly EXIT_REVIEW=3
readonly STAMP_LINES=10
# The sim/conformance tests that write the goldens and SUMMARY.md's
# generated blocks when FOREVER_UPDATE_GOLDEN=1.
readonly GOLDEN_WRITERS='^(TestConformance|TestConformanceSummaryBlocks|TestSetBonusConformance)$'
readonly BUILD_RE='^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$'

readonly FOREVER_BUILD_FILE=assets/db_inputs/FOREVER_BUILD
readonly WARRIOR_FIXTURE=tools/talentgen/testdata/warrior.json
readonly TALENT_FIXTURES=tools/talentgen/testdata
readonly CONFORMANCE=sim/core/testdata/conformance
readonly CLIENT_DIR=$CONFORMANCE/client
readonly RACES=$CLIENT_DIR/races.json
readonly REPORT=dist/FOLLOW-REPORT.md
readonly PATHS_OUT=dist/FOLLOW-PATHS

STEP=setup
die() {
	echo "follow: step '$STEP' failed: $*" >&2
	exit "$EXIT_FAILURE"
}

step() {
	STEP=$1
	echo "follow: == $STEP"
}

# allowlist prints the glob patterns of every path a follow of build $1 may
# change: the generated artifacts, their verbatim inputs, the goldens and
# the summary, and FOREVER_BUILD.
allowlist() {
	local build=$1 class
	for class in "${CLASSES[@]}"; do
		printf '%s\n' \
			"proto/$class.proto" \
			"sim/core/proto/$class.pb.go" \
			"sim/$class/talents_auto_gen.go" \
			"ui/core/talents/trees/$class.json" \
			"sim/$class/constants_auto_gen.go" \
			"sim/$class/relic_mods_auto_gen.go" \
			"$CLIENT_DIR/$class.json" \
			"$CLIENT_DIR/trainables/$class.json" \
			"$TALENT_FIXTURES/$class.json" \
			"$CONFORMANCE/$class.golden.md"
	done
	printf '%s\n' \
		sim/core/base_stats_auto_gen.go \
		sim/core/base_stats_levels_auto_gen.go \
		sim/core/client_sets_gen.go \
		"$RACES" \
		"assets/db_inputs/gametables/$build/*" \
		"assets/db_inputs/levels/$build.json" \
		"$CONFORMANCE/sets.golden.md" \
		"$CONFORMANCE/SUMMARY.md" \
		"$FOREVER_BUILD_FILE"
}

# changed_paths prints every path git reports as changed, untracked files
# listed individually rather than collapsed into their directory.
changed_paths() {
	git status --porcelain --untracked-files=all | cut -c4-
}

is_allowed() {
	local path=$1 pattern
	while IFS= read -r pattern; do
		# shellcheck disable=SC2053 # the pattern is a deliberate glob
		[[ $path == $pattern ]] && return 0
	done < <(allowlist "$BUILD")
	return 1
}

# 0. Arguments, toolchain and the scratch directory.
setup() {
	step "0 setup"
	[[ $# -eq 2 ]] || die "usage: tools/follow.sh <site dir> <build>"
	SITE=$(cd "$1" 2>/dev/null && pwd) || die "no site checkout at '$1'"
	BUILD=$2
	[[ $BUILD =~ $BUILD_RE ]] || die "build '$BUILD' is not a client build number"
	SRC=$SITE/data/builds/$BUILD
	[[ -d $SRC ]] || die "no site build directory $SRC"
	FORK=$(git rev-parse --show-toplevel) || die "not inside the fork's git checkout"
	cd "$FORK"
	local gobin
	gobin=$(go env GOPATH)/bin
	case ":$PATH:" in *":$gobin:"*) ;; *) export PATH=$PATH:$gobin ;; esac
	TMP=$(mktemp -d)
	trap 'rm -rf "$TMP"' EXIT
	BUILD_LOG=""
}

# 1. Refuse a dirty checkout: the allowlist check needs a clean baseline.
refuse_dirty() {
	step "1 refuse a dirty checkout"
	local dirty
	dirty=$(git status --porcelain)
	[[ -z $dirty ]] || die "the checkout has uncommitted changes:"$'\n'"$dirty"
}

# 2. The previously followed build, seeded from the warrior fixture.
previous_build() {
	step "2 previous build"
	PREV=$(cat "$FOREVER_BUILD_FILE" 2>/dev/null || true)
	if [[ -z $PREV ]]; then
		PREV=$(python3 -I -c 'import json,sys; print(json.load(open(sys.argv[1]))["build"])' "$WARRIOR_FIXTURE") ||
			die "no $FOREVER_BUILD_FILE and no build field in $WARRIOR_FIXTURE"
	fi
	echo "follow: $PREV -> $BUILD"
}

# 3. Snapshot the before state the report diffs against.
snapshot() {
	step "3 snapshot"
	mkdir -p "$TMP/goldens" "$TMP/talents"
	cp "$CONFORMANCE"/*.golden.md "$TMP/goldens/" || die "copying the goldens"
	cp "$TALENT_FIXTURES"/*.json "$TMP/talents/" || die "copying the talent fixtures"
	cp "$RACES" "$TMP/races.json" || die "copying $RACES"
}

# 4. Verbatim copies of the site build's tables.
copy_inputs() {
	step "4 copy the site build's tables"
	mkdir -p "assets/db_inputs/gametables/$BUILD" "$CLIENT_DIR/trainables"
	cp "$SRC"/talents/*.json "$TALENT_FIXTURES/" || die "copying talents/"
	cp "$SRC"/spellconst/*.json "$CLIENT_DIR/" || die "copying spellconst/"
	cp "$SRC"/trainables/*.json "$CLIENT_DIR/trainables/" || die "copying trainables/"
	cp "$SRC/races.json" "$RACES" || die "copying races.json"
	cp "$SRC"/gametables/* "assets/db_inputs/gametables/$BUILD/" || die "copying gametables/"
}

# 5. Talent trees, their protos and the UI trees. make talents runs
# `make proto`, whose rule compares mtimes; make 3.81 (macOS) reads them to
# the second, so a talentgen that writes in the checkout's second leaves
# stale .pb.go behind. The forced rebuild makes the protos follow always.
gen_talents() {
	step "5 make talents"
	make talents TALENT_BUILD="$BUILD" SITE_DIR="$SITE" || die "make talents"
	make -B sim/core/proto/api.pb.go || die "make -B sim/core/proto/api.pb.go"
}

# 6. Per-class spell constants.
gen_spellconst() {
	step "6 make spellconst"
	make spellconst BUILD="$BUILD" SITE_DIR="$SITE" || die "make spellconst"
}

# 7. Base stats from the game tables and the per-level payload.
gen_base_stats() {
	step "7 base stats"
	local levels=assets/db_inputs/levels/$BUILD.json
	python3 -I tools/levels_from_site.py "$SRC/levels.json" "$levels" || die "levels_from_site.py"
	python3 -I tools/base_stats_parser.py --inputs "assets/db_inputs/gametables/$BUILD" \
		--build "$BUILD" --levels-json "$levels" || die "base_stats_parser.py"
}

# 8. Relic spell mods.
gen_relicmods() {
	step "8 make relicmods"
	make relicmods BUILD="$BUILD" SITE_DIR="$SITE" || die "make relicmods"
}

# 9. Item-set rows, generated by the site's script from outside both repos.
gen_set_rows() {
	step "9 set rows"
	(cd /tmp && python3 -I "$SITE/tools/gen_set_rows.py" "$SRC" "$FORK") || die "gen_set_rows.py"
}

# 10. Does the regenerated engine compile? A failure is review, not abort.
# sim/web embeds the gitignored binary_dist package, which CI also makes
# before it builds; without it every build would fail for a reason that is
# not the follow's.
build_engine() {
	step "10 go build"
	make binary_dist/dist.go || die "make binary_dist/dist.go"
	if ! go build --tags=with_db ./sim/... 2>"$TMP/build.log"; then
		BUILD_LOG=$TMP/build.log
		echo "follow: the engine does not compile at $BUILD; goldens kept, see the report's Build section" >&2
	fi
}

# 11. Regenerate the conformance goldens and the summary. Only the tests
# that write them run here: the package's other tests check hand-written
# files (dispositions.go, the expected verdicts) against the new goldens,
# and a new client build failing them is the review the report opens, not
# a generator failure. The fork suite runs them as the gate.
gen_goldens() {
	step "11 goldens"
	if [[ -n $BUILD_LOG ]]; then
		echo "follow: skipped, the engine does not compile"
		return
	fi
	FOREVER_UPDATE_GOLDEN=1 go test --tags=with_db -count=1 -run "$GOLDEN_WRITERS" ./sim/conformance/... ||
		die "regenerating the goldens"
}

# 12. Record the followed build.
record_build() {
	step "12 FOREVER_BUILD"
	echo "$BUILD" >"$FOREVER_BUILD_FILE"
}

assert_stamp() {
	local file=$1 stamp=$2
	[[ -f $file ]] || die "$file is missing"
	head -n "$STAMP_LINES" "$file" | grep -F -- "$stamp" >/dev/null || die "$file does not name '$stamp' in its header"
}

# 13. Every regenerated file names the requested build in its header.
assert_stamps() {
	step "13 build stamps"
	local class file
	for class in "${CLASSES[@]}"; do
		assert_stamp "sim/$class/talents_auto_gen.go" "build $BUILD"
		assert_stamp "sim/$class/constants_auto_gen.go" "Client build: $BUILD"
		file=sim/$class/relic_mods_auto_gen.go
		[[ ! -f $file ]] || assert_stamp "$file" "Client build: $BUILD"
		[[ -n $BUILD_LOG ]] || assert_stamp "$CONFORMANCE/$class.golden.md" "Client build: $BUILD"
	done
	assert_stamp sim/core/base_stats_auto_gen.go "Client build: $BUILD"
	assert_stamp sim/core/client_sets_gen.go "client build $BUILD"
}

# numstat prints `git diff --numstat` lines for the allowlisted paths,
# untracked files counted as all lines added.
numstat() {
	local -a patterns=()
	local pattern
	while IFS= read -r pattern; do patterns+=("$pattern"); done < <(allowlist "$BUILD")
	git diff --numstat -- "${patterns[@]}"
	local path
	while IFS= read -r path; do
		printf '%s\t0\t%s\n' "$(wc -l <"$path" | tr -d ' ')" "$path"
	done < <(git ls-files --others --exclude-standard -- "${patterns[@]}")
}

# 14. The follow report.
write_report() {
	step "14 follow report"
	mkdir -p dist
	numstat >"$TMP/numstat" || die "collecting the diffstat"
	go build -o "$TMP/followreport" ./tools/followreport || die "building tools/followreport"
	REPORT_STATUS=0
	"$TMP/followreport" -before "$TMP/goldens" -after "$CONFORMANCE" \
		-prev-talents "$TMP/talents" -talents "$TALENT_FIXTURES" \
		-prev-races "$TMP/races.json" -races "$RACES" \
		-build-log "$BUILD_LOG" -generated-diffstat "$TMP/numstat" \
		-out "$REPORT" || REPORT_STATUS=$?
	case $REPORT_STATUS in
	0 | "$EXIT_REVIEW") ;;
	*) die "tools/followreport exited $REPORT_STATUS" ;;
	esac
}

# 15. The changed paths, every one of them on the allowlist.
write_paths() {
	step "15 allowlist"
	local path outside=""
	: >"$PATHS_OUT"
	while IFS= read -r path; do
		if is_allowed "$path"; then
			echo "$path" >>"$PATHS_OUT"
		else
			outside+="  $path"$'\n'
		fi
	done < <(changed_paths)
	[[ -z $outside ]] || die "paths changed outside the allowlist (a script defect):"$'\n'"$outside"
}

# 16. The verdict, as the last line and the exit status.
verdict() {
	step "16 verdict"
	local line
	line=$(head -n 1 "$REPORT")
	case $line in
	"verdict: clean") [[ $REPORT_STATUS -eq 0 ]] || die "report says clean, followreport exited $REPORT_STATUS" ;;
	"verdict: review") [[ $REPORT_STATUS -eq $EXIT_REVIEW ]] || die "report says review, followreport exited $REPORT_STATUS" ;;
	*) die "$REPORT does not open with a verdict line: '$line'" ;;
	esac
	echo "$line"
	exit "$REPORT_STATUS"
}

main() {
	setup "$@"
	refuse_dirty
	previous_build
	snapshot
	copy_inputs
	gen_talents
	gen_spellconst
	gen_base_stats
	gen_relicmods
	gen_set_rows
	build_engine
	gen_goldens
	record_build
	assert_stamps
	write_report
	write_paths
	verdict
}

main "$@"
