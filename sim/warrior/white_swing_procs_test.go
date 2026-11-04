package warrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Unbridled Wrath (12322), the Flurry charge aura (12966) and Reckless
// Swings (1329555) all carry the client's ProcTypeMask 4, "done melee auto
// attack". Forever beta logs (ppach/marrow.github.io eternal/data/logs,
// 2026-09-23, build 1.60.1; forever-bugs #105) show the server does not
// count a queued Heroic Strike or Cleave as one: Zomd and Siax landed 228
// Heroic Strikes and Cleaves between them at a ~36% proc rate and none of
// their 472 Unbridled Wrath procs followed one. So the next-swing
// abilities, which the engine gives both a special and an auto mask, must
// not feed those three auras; only a pure white swing does.
func TestOnlyAPureWhiteSwingFeedsTheAutoAttackAuras(t *testing.T) {
	cases := []struct {
		name string
		mask core.ProcMask
		want bool
	}{
		{"main-hand white swing", core.ProcMaskMeleeMHAuto, true},
		{"off-hand white swing", core.ProcMaskMeleeOHAuto, true},
		{"Heroic Strike / Cleave", onNextSwingProcMask, false},
		{"main-hand ability", core.ProcMaskMeleeMHSpecial, false},
		{"off-hand ability (Whirlwind's off hand)", core.ProcMaskMeleeOHSpecial, false},
		{"ranged auto", core.ProcMaskRangedAuto, false},
	}
	for _, c := range cases {
		if got := countsAsWhiteSwing(c.mask); got != c.want {
			t.Errorf("%s (mask %v): countsAsWhiteSwing = %v, want %v", c.name, c.mask, got, c.want)
		}
	}
}

// The next-swing abilities keep their auto bit (they replace a white swing,
// reset the swing timer and are still main-hand melee for every other
// proc); the rule above is what keeps them off the mask-4 auras.
func TestTheNextSwingAbilitiesKeepTheirAutoBit(t *testing.T) {
	if !onNextSwingProcMask.Matches(core.ProcMaskMeleeMHAuto) || !onNextSwingProcMask.Matches(core.ProcMaskMeleeMHSpecial) {
		t.Errorf("onNextSwingProcMask = %v, want main-hand auto plus main-hand special", onNextSwingProcMask)
	}
}
