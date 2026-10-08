package dpsrogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/rogue"
	googleproto "google.golang.org/protobuf/proto"
)

// The raid debuffs Judgement of Wisdom and Judgement of Light are how a
// request states that a raid paladin judges them. Wisdom returns the
// attacker's mana (a rogue has none, so it is read through a mana user in
// the priest and paladin tests); Light heals a melee attacker.

func debuffsWithJudgement(wisdom, light bool) *proto.Debuffs {
	debuffs := googleproto.Clone(core.FullBuffs.Debuffs).(*proto.Debuffs)
	debuffs.JudgementOfWisdom = wisdom
	debuffs.JudgementOfLight = light
	return debuffs
}

func buildRogueUnderDebuffs(t *testing.T, debuffs *proto.Debuffs) (*core.Simulation, *rogue.Rogue) {
	t.Helper()
	return buildRogueWithBuffs(t, "", core.FullBuffs.Raid, debuffs)
}

func TestJudgementOfLightRaidDebuffRegistersTheAura(t *testing.T) {
	sim, _ := buildRogueUnderDebuffs(t, debuffsWithJudgement(false, true))
	if sim.GetTargetUnit(0).GetAura("Judgement of Light") == nil {
		t.Fatal("the Judgement of Light raid debuff did not register the aura on the target")
	}
	sim, _ = buildRogueUnderDebuffs(t, debuffsWithJudgement(false, false))
	if sim.GetTargetUnit(0).GetAura("Judgement of Light") != nil {
		t.Fatal("a target without the debuff has the aura")
	}
}

// A melee hit that lands on the judged target heals its attacker for the
// rank's amount (61 at level 60) about half the time.
func TestJudgementOfLightHealsAMeleeAttackerAboutHalfTheTime(t *testing.T) {
	sim, built := buildRogueUnderDebuffs(t, debuffsWithJudgement(false, true))
	built.EnableHealthBar()

	aura := sim.GetTargetUnit(0).GetAura("Judgement of Light")
	hit := &core.SpellResult{Outcome: core.OutcomeHit}
	const swings = 400
	for i := 0; i < swings; i++ {
		aura.OnSpellHitTaken(aura, sim, built.AutoAttacks.MHAuto(), hit)
	}
	metrics := built.JolHealthMetrics
	if metrics == nil || float64(metrics.Events) < 0.4*swings || float64(metrics.Events) > 0.6*swings {
		t.Fatalf("Judgement of Light healed on %v of %d hits, want about half", metrics, swings)
	}
	if got := metrics.Gain / float64(metrics.Events); got != 61 {
		t.Errorf("each Judgement of Light heal = %v, want 61", got)
	}
}

func TestJudgementOfLightIgnoresSpellsAndMisses(t *testing.T) {
	sim, built := buildRogueUnderDebuffs(t, debuffsWithJudgement(false, true))
	built.EnableHealthBar()
	aura := sim.GetTargetUnit(0).GetAura("Judgement of Light")
	for i := 0; i < 200; i++ {
		aura.OnSpellHitTaken(aura, sim, built.AutoAttacks.MHAuto(), &core.SpellResult{Outcome: core.OutcomeMiss})
		aura.OnSpellHitTaken(aura, sim, &core.Spell{Unit: &built.Unit, ProcMask: core.ProcMaskSpellDamage}, &core.SpellResult{Outcome: core.OutcomeHit})
	}
	if built.JolHealthMetrics != nil && built.JolHealthMetrics.Events != 0 {
		t.Error("Judgement of Light healed from a miss or a spell")
	}
}
