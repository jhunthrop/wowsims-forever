package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Aspect of the Falcon's own proc, from the 1.60.1.70755 client rows:
// SpellAuraOptions 239200 (spell 469145) ProcChance 5, ProcTypeMask 20
// (0x4 melee auto attack done, 0x10 melee-class spell done), and effect 2
// is aura 42 triggering Quick Strikes (469144, +30% melee attack speed,
// 12 sec). Falcon's SpellClassOptions mask is 0, so Deadly Aspects' chance
// mods (19552 effects 0 and 1, aura 107 misc 18, masks word0 1048576 =
// Aspect of the Hawk and word2 16 = Aspect of the Beast) never reach it:
// Falcon procs at a flat 5% on melee autos and melee strikes, talent or
// not, and never from a ranged attack.
func falconQuickStrikesRate(t *testing.T, deadlyAspects int32, spell *core.Spell) float64 {
	t.Helper()
	sim, hunter, target := newRunningHunterForTalentTest(t, 60, "deadly_aspects", deadlyAspects, proto.Hunter_Options_PetNone, 5, nil)

	falcon := hunter.GetAura("Aspect of the Falcon")
	if falcon == nil {
		t.Fatal("level-60 hunter has no Aspect of the Falcon aura registered")
	}
	falcon.Activate(sim)
	if falcon.OnSpellHitDealt == nil {
		t.Fatal("Aspect of the Falcon carries no proc of its own (client ProcChance 5 on 469145)")
	}
	quickStrikes := hunter.GetAura("Quick Strikes")
	if quickStrikes == nil {
		t.Fatal("no Quick Strikes aura registered")
	}

	landed := &core.SpellResult{Outcome: core.OutcomeLanded, Target: target}
	const trials = 20000
	procs := 0
	for i := 0; i < trials; i++ {
		falcon.OnSpellHitDealt(falcon, sim, spell, landed)
		if quickStrikes.IsActive() {
			procs++
			quickStrikes.Deactivate(sim)
		}
	}
	return float64(procs) / trials
}

func TestAspectOfTheFalconProcsQuickStrikesAtFivePercentWithoutTheTalent(t *testing.T) {
	for _, tc := range []struct {
		name string
		mask core.ProcMask
	}{
		{"melee auto", core.ProcMaskMeleeMHAuto},
		{"off-hand auto", core.ProcMaskMeleeOHAuto},
		{"melee strike", core.ProcMaskMeleeMHSpecial},
	} {
		if got := falconQuickStrikesRate(t, 0, &core.Spell{ProcMask: tc.mask}); got < 0.04 || got > 0.06 {
			t.Errorf("%s under Falcon, no Deadly Aspects: Quick Strikes rate %.4f, want 0.05", tc.name, got)
		}
	}
}

func TestDeadlyAspectsDoesNotRaiseAspectOfTheFalconsProcChance(t *testing.T) {
	if got := falconQuickStrikesRate(t, 5, &core.Spell{ProcMask: core.ProcMaskMeleeMHAuto}); got < 0.04 || got > 0.06 {
		t.Errorf("melee auto under Falcon with Deadly Aspects 5: Quick Strikes rate %.4f, want 0.05 (Falcon's mask is 0)", got)
	}
}

func TestAspectOfTheFalconNeverProcsFromRangedAttacks(t *testing.T) {
	for _, mask := range []core.ProcMask{core.ProcMaskRangedAuto, core.ProcMaskRangedSpecial} {
		if got := falconQuickStrikesRate(t, 5, &core.Spell{ProcMask: mask}); got != 0 {
			t.Errorf("ranged mask %v under Falcon: Quick Strikes rate %.4f, want 0", mask, got)
		}
	}
}

// Under Falcon, Deadly Aspects' ranged half stays off: its Auto Shot
// chance mod reaches only Aspect of the Hawk (19552 effect 0, word0
// 1048576), so a ranged auto under Falcon never speeds ranged attacks.
func TestDeadlyAspectsRangedHasteNeedsAspectOfTheHawk(t *testing.T) {
	sim, hunter, target := newRunningHunterForTalentTest(t, 60, "deadly_aspects", 5, proto.Hunter_Options_PetNone, 25, nil)
	hunter.GetAura("Aspect of the Falcon").Activate(sim)
	deadlyAspects := hunter.GetAura("Deadly Aspects")
	landed := &core.SpellResult{Outcome: core.OutcomeLanded, Target: target}
	for i := 0; i < 2000; i++ {
		deadlyAspects.OnSpellHitDealt(deadlyAspects, sim, &core.Spell{ProcMask: core.ProcMaskRangedAuto}, landed)
	}
	if hunter.HasActiveAura("Deadly Aspects Haste") {
		t.Error("Deadly Aspects sped ranged attacks under Aspect of the Falcon")
	}
}
