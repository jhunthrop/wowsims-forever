package core

import (
	"testing"
	"time"
)

// Classic's rule, which Forever keeps: haste shortens a cast, never the 1.5 s
// global cooldown. Client 1.60.1.70755 gives every global-cooldown spell
// StartRecoveryCategory 133 and StartRecoveryTime 1500 (Lightning Bolt 15208,
// Frostbolt 10181, Whirlwind 1680), and Forever's only global cooldown cuts are
// named effects (Nature's Grace "reduces your global cooldown", totems at
// 1 s), not haste. The cast path (cast.go) hastes CastTime alone and sets the
// global cooldown from max(GCD, CastTime): a 1.5 s cast hasted to 1.2 s still
// waits out the full 1.5 s global.
func TestHasteNeverShortensTheGlobalCooldown(t *testing.T) {
	hasted := Cast{GCD: GCDDefault, CastTime: 1200 * time.Millisecond}
	if got := hasted.EffectiveTime(); got != GCDDefault {
		t.Errorf("a cast hasted to 1.2 s holds the global cooldown for %s, want the full %s", got, GCDDefault)
	}
	slow := Cast{GCD: GCDDefault, CastTime: 2500 * time.Millisecond}
	if got := slow.EffectiveTime(); got != 2500*time.Millisecond {
		t.Errorf("a 2.5 s cast holds the global cooldown for %s, want its cast time", got)
	}
	cut := Cast{GCD: 500 * time.Millisecond}
	if got := cut.EffectiveTime(); got != GCDMin {
		t.Errorf("a global cooldown cut below the floor lasts %s, want %s", got, GCDMin)
	}
}
