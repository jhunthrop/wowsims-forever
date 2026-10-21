package proto_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The generated protobuf code is committed (see PORTING.md): the site
// consumes this repository as a Go module at a pinned pseudo-version, and
// Go resolves packages from the commit, not from a build step. That makes
// drift between proto/*.proto and sim/core/proto/*.pb.go a real bug, so
// this test regenerates into a temp dir and diffs.
func TestGeneratedProtosMatchSources(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not installed; CI enforces this test")
	}
	if _, err := exec.LookPath("protoc-gen-go"); err != nil {
		t.Skip("protoc-gen-go not installed; run go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6")
	}

	root := filepath.Join("..", "..", "..")
	tmp := t.TempDir()
	// protoc writes to <out>/proto, because the .proto files declare
	// `option go_package = "./proto";` (relative to the --go_out directory,
	// matching how the makefile invokes protoc with --go_out=./sim/core).
	cmd := exec.Command("protoc", "-I=./proto", "--go_out="+tmp, "./proto/api.proto",
		"./proto/apl.proto", "./proto/common.proto", "./proto/druid.proto",
		"./proto/hunter.proto", "./proto/mage.proto", "./proto/paladin.proto",
		"./proto/priest.proto", "./proto/rogue.proto", "./proto/shaman.proto",
		"./proto/test.proto", "./proto/ui.proto", "./proto/warlock.proto",
		"./proto/warrior.proto")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("protoc failed: %v\n%s", err, out)
	}

	genDir := filepath.Join(tmp, "proto")
	entries, err := os.ReadDir(genDir)
	if err != nil {
		t.Fatalf("reading regenerated dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("protoc produced no files")
	}
	for _, e := range entries {
		want, err := os.ReadFile(filepath.Join(genDir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("%s is not committed: %v (run `make proto` and commit the result)", e.Name(), err)
		}
		if string(got) != string(want) {
			t.Errorf("%s is stale; run `make proto` and commit the result", e.Name())
		}
	}
}
