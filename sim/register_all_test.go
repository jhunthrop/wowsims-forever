package sim

import (
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// RegisterAll writes agent factories into package-level maps in core.
// The site's Cloud Run job calls it from inside its per-run Execute
// rather than once in main(), so concurrent runs call it concurrently;
// the unsynchronised package bool that used to guard it made the first
// of those a data race and a possible double registration (which
// core.RegisterAgentFactory panics on).
//
// The concurrency half of this test cannot witness the first-call race:
// raid_test.go's init() calls RegisterAll before any test in this
// binary runs, so by here the registration is always already done. So
// the guard itself is asserted from the source, the way
// TestProvisionalConstantsAreDeclared asserts its file's marking — and
// the goroutines below still hold the re-entrant path, under -race.
func TestRegisterAllIsGuardedByASyncOnce(t *testing.T) {
	src, err := os.ReadFile("register_all.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "registerOnce.Do(registerAll)") {
		t.Error("RegisterAll no longer dispatches through a sync.Once; a plain bool is a data race for the concurrent caller (the site's Cloud Run job) and can register twice")
	}
	if strings.Contains(string(src), "var registered = false") {
		t.Error("the unsynchronised `registered` bool is back in register_all.go")
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			RegisterAll()
		}()
	}
	wg.Wait()

	if got := core.PlayerProtoToSpec(&proto.Player{Spec: &proto.Player_Mage{Mage: &proto.Mage{}}}); got != proto.Spec_SpecMage {
		t.Errorf("PlayerProtoToSpec for a mage is %v after RegisterAll, want %v", got, proto.Spec_SpecMage)
	}
}
