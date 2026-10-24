package mage

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	googleproto "google.golang.org/protobuf/proto"
)

// debuffsWithoutWintersChill is the full raid debuff set minus the other
// mage's Winter's Chill, which stacks the same target aura every 1.5 sec.
func debuffsWithoutWintersChill() *proto.Debuffs {
	debuffs := googleproto.Clone(core.FullBuffs.Debuffs).(*proto.Debuffs)
	debuffs.WintersChill = false
	return debuffs
}

func newWintersChillSim(t *testing.T, talents string) (*Mage, *core.Simulation, *core.Unit) {
	t.Helper()
	return newFrostMageSimWithTalentsAndDebuffs(t, bossTarget(), talents, debuffsWithoutWintersChill())
}

// Winter's Chill since client 1.60.1.70532: Frostbolt and Fingers of
// Frost empowered Ice Lance stack it, Cone of Cold reads and spends it.

func wintersChillStacksOn(mage *Mage, target *core.Unit) int32 {
	return mage.wintersChillStacks(target)
}

func TestFrostboltStacksWintersChillAndConeOfColdSpendsIt(t *testing.T) {
	built, sim, boss := newWintersChillSim(t, ForeverFrostTalents)
	frostbolt := highestRegistered(t, "Frostbolt", built.Frostbolt)
	forceOutcome(frostbolt, false)
	for i := 0; i < 10; i++ {
		castAndLand(sim, boss, frostbolt)
	}
	if got := wintersChillStacksOn(built, boss); got == 0 {
		t.Fatal("10 landed Frostbolts at Winter's Chill 5/5 left no stacks on the target")
	}

	cone := built.ConeOfCold[ConeOfColdRanks]
	cone.ApplyEffects(sim, boss, cone)
	if got := wintersChillStacksOn(built, boss); got != 0 {
		t.Errorf("Cone of Cold left %d Winter's Chill stacks, want 0: it spends them", got)
	}
	for i := 0; i < 10; i++ {
		cone.ApplyEffects(sim, boss, cone)
	}
	if got := wintersChillStacksOn(built, boss); got != 0 {
		t.Errorf("10 Cones of Cold stacked Winter's Chill to %d, want 0: Cone of Cold is not an applier", got)
	}
}

func TestIceLanceStacksWintersChillOnlyOnAFingersOfFrostCharge(t *testing.T) {
	built, sim, boss := newWintersChillSim(t, fingersTalents(t, 2))
	lance := built.IceLance[IceLanceRanks]
	forceOutcome(lance, false)
	for i := 0; i < 20; i++ {
		// A landed Ice Lance is a Chill and can grant a charge the next
		// cast would spend; a "plain" cast starts without one.
		built.FingersOfFrostAura.Deactivate(sim)
		castAndLand(sim, boss, lance)
	}
	if got := wintersChillStacksOn(built, boss); got != 0 {
		t.Errorf("20 plain Ice Lances stacked Winter's Chill to %d, want 0", got)
	}
	for i := 0; i < 10; i++ {
		built.grantFingersOfFrost(sim)
		castAndLand(sim, boss, lance)
	}
	if got := wintersChillStacksOn(built, boss); got == 0 {
		t.Error("10 Fingers of Frost empowered Ice Lances left no Winter's Chill stacks")
	}
}

// coneDamageWithStacks is Cone of Cold's damage over casts, with the
// target carrying stacks Winter's Chill stacks before each cast.
func coneDamageWithStacks(t *testing.T, stacks int32, casts int) (float64, *Mage, *core.Unit) {
	t.Helper()
	built, sim, boss := newWintersChillSim(t, ForeverFrostTalents)
	cone := built.ConeOfCold[ConeOfColdRanks]
	forceOutcome(cone, false)
	for i := 0; i < casts; i++ {
		if stacks > 0 {
			aura := built.WintersChillAuras.Get(boss)
			aura.Activate(sim)
			aura.SetStacks(sim, stacks)
		}
		cone.ApplyEffects(sim, boss, cone)
	}
	return cone.SpellMetrics[boss.UnitIndex].TotalDamage, built, boss
}

// Five stacks: x1.25 damage taken and +0.35 of the mage's bonus damage.
func TestWintersChillStacksRaiseConeOfColdDamage(t *testing.T) {
	const casts = 400
	none, built, boss := coneDamageWithStacks(t, 0, casts)
	five, _, _ := coneDamageWithStacks(t, core.WintersChillMaxStacks, casts)

	cone := built.ConeOfCold[ConeOfColdRanks]
	roll := built.clientRoll(ConeOfColdBaseDamage[ConeOfColdRanks], ConeOfColdPointsPerLevel[ConeOfColdRanks], ConeOfColdLevel[ConeOfColdRanks], ConeOfColdMaxLevel[ConeOfColdRanks])
	base := (roll[0] + roll[1]) / 2
	bonus := cone.GetBonusDamage(boss)
	stacks := float64(core.WintersChillMaxStacks)
	want := (1 + wintersChillConeOfColdDamagePerStack*stacks) *
		(base + (cone.BonusCoefficient+wintersChillConeOfColdCoefficientPerStack*stacks)*bonus) /
		(base + cone.BonusCoefficient*bonus)
	got := five / none
	if got < want*0.96 || got > want*1.04 {
		t.Errorf("five Winter's Chill stacks changed Cone of Cold damage by x%.3f, want about x%.3f (base %.0f, bonus damage %.0f at coefficient %.3f)", got, want, base, bonus, cone.BonusCoefficient)
	}
	if cone.BonusCoefficient != ConeOfColdSpellCoeff[ConeOfColdRanks] {
		t.Errorf("Cone of Cold's coefficient is %.3f after casting, want the client's %.3f restored", cone.BonusCoefficient, ConeOfColdSpellCoeff[ConeOfColdRanks])
	}
}
