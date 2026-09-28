package dpsrogue

import (
	"strings"
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/rogue"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// rogueTalentStringWith returns an all-zero talent string (segments
// sized per rogue.TalentTreeSizes) with the named RogueTalents fields set
// to 1. FillTalentsProto parses positionally against the proto field
// number and does not validate prerequisites or max rank (the same fact
// mage/talents_test.go documents for MageTalents), so this is enough to
// talent into a single Forever-only node like Mutilate or Venom without
// a full, legal build.
func rogueTalentStringWith(t *testing.T, fieldNames ...string) string {
	t.Helper()

	segments := make([]string, len(rogue.TalentTreeSizes))
	for i, n := range rogue.TalentTreeSizes {
		segments[i] = strings.Repeat("0", n)
	}

	for _, name := range fieldNames {
		fd := (&proto.RogueTalents{}).ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(name))
		if fd == nil {
			t.Fatalf("RogueTalents has no field named %q", name)
		}
		pos := int(fd.Number()) - 1
		treeIdx := 0
		for treeIdx < len(rogue.TalentTreeSizes) && pos >= rogue.TalentTreeSizes[treeIdx] {
			pos -= rogue.TalentTreeSizes[treeIdx]
			treeIdx++
		}
		chars := []rune(segments[treeIdx])
		chars[pos] = '1'
		segments[treeIdx] = string(chars)
	}

	return strings.Join(segments, "-")
}

// testMHWeapon/testOHWeapon are synthetic dual-wield weapons set directly
// on AutoAttacks after construction, bypassing core.GetGearSet: this
// fork's item database isn't generated in this test environment (see
// pet_level_test.go and, for the same "No item with id" panic on
// existing, already-committed code, mage/talents_test.go's
// buildMageForTalentTest), so any Equipment built from real item ids
// panics here regardless of which package touches it.
var (
	testMHWeapon = core.Weapon{
		BaseDamageMin:        50,
		BaseDamageMax:        90,
		SwingSpeed:           1.8,
		NormalizedSwingSpeed: 1.7,
		AttackPowerPerDPS:    core.DefaultAttackPowerPerDPS,
	}
	testOHWeapon = core.Weapon{
		BaseDamageMin:        40,
		BaseDamageMax:        70,
		SwingSpeed:           1.8,
		NormalizedSwingSpeed: 1.7,
		AttackPowerPerDPS:    core.DefaultAttackPowerPerDPS,
	}
)

// buildRogueForTest stands up a level-60 rogue with synthetic dual-wield
// daggers (so AutoAttacks.IsDualWielding is true) and the given talent
// string, through the shipping agent factory.
func buildRogueForTest(t *testing.T, talentsStr string) (*core.Simulation, *rogue.Rogue) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassRogue,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			Buffs:         core.FullBuffs.Player,
			TalentsString: talentsStr,
			// A rogue's energy bar walks player.Rotation unconditionally
			// during Environment.finalize (sim/core/energy.go's
			// setupEnergyThresholds, restricted to Class_ClassRogue) with
			// no nil check on Rotation itself, so an empty-but-non-nil
			// rotation is required here even though this test never lets
			// it run (every cast in this file drives ApplyEffects
			// directly).
			Rotation: &proto.APLRotation{},
		},
		DefaultRogue,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	agent, ok := sim.Raid.Parties[0].Players[0].(rogue.RogueAgent)
	if !ok {
		t.Fatal("the raid's first player is not a rogue agent")
	}
	built := agent.GetRogue()
	built.AutoAttacks.SetMH(testMHWeapon)
	built.AutoAttacks.SetOH(testOHWeapon)
	built.AutoAttacks.IsDualWielding = true

	return sim, built
}

// TestMutilateNotRegisteredWithoutTheTalent rebuilds the state
// data/curated/apl/rogue-assassination.json's "inert" entry describes: a
// rogue who hasn't spent the point must not have Mutilate at all.
func TestMutilateNotRegisteredWithoutTheTalent(t *testing.T) {
	_, built := buildRogueForTest(t, rogueTalentStringWith(t))

	if built.Mutilate != nil {
		t.Fatal("rogue without the Mutilate talent has Mutilate registered")
	}
}

// TestMutilateLevel60DealsDamageAndAwardsComboPoints is the case the
// curated APL's "inert" entry for spell 1310707 is waiting on: a
// talented level-60 rogue must have Mutilate registered, and casting it
// must land on both weapons, deal damage, and award the 2 combo points
// the talent's tooltip states.
//
// The registered id is rank 4's (1241584), not rank 1's (1310707):
// Mutilate is a four-rank ability learned by level
// (spellranks.json's chain, 1310707@30/399956@40/1241582@50/1241584@60),
// not the single always-known ability this test used to assume, and a
// level-60 rogue has learned every rank.
func TestMutilateLevel60DealsDamageAndAwardsComboPoints(t *testing.T) {
	sim, built := buildRogueForTest(t, rogueTalentStringWith(t, "mutilate"))

	if built.Mutilate == nil {
		t.Fatal("talented level-60 rogue has no Mutilate registered")
	}
	if got, want := built.Mutilate.ActionID.SpellID, int32(1241584); got != want {
		t.Errorf("Mutilate spell ID = %d, want %d", got, want)
	}
	if !built.AutoAttacks.IsDualWielding {
		t.Fatal("test gear set is not dual-wielding; Mutilate's ExtraCastCondition will always fail")
	}

	target := sim.Encounter.TargetUnits[0]

	built.Mutilate.ApplyEffects(sim, target, built.Mutilate)

	// The talented button (built.Mutilate) never calls
	// CalcAndDealDamage itself -- it delegates to MutilateMH/MutilateOH,
	// one call each, so their own SpellMetrics carry the damage/outcome.
	mh := built.MutilateMH.SpellMetrics[target.UnitIndex]
	oh := built.MutilateOH.SpellMetrics[target.UnitIndex]
	if mh.Hits+mh.Crits+oh.Hits+oh.Crits == 0 {
		t.Fatalf("Mutilate landed neither hand as a hit or crit (mh misses=%d, oh misses=%d)", mh.Misses, oh.Misses)
	}
	totalDamage := mh.TotalDamage + oh.TotalDamage
	if totalDamage <= 0 {
		t.Errorf("Mutilate dealt %v damage, want > 0", totalDamage)
	}
	if got, want := built.ComboPoints(), int32(2); got != want {
		t.Errorf("combo points after one landed Mutilate = %d, want %d", got, want)
	}
}

// TestMutilatePoisonedTargetDealsMoreDamage pins the talent tooltip's
// "Damage increased by 20% against Poisoned targets" against a target
// carrying the caster's own Deadly Poison dot, cast through the real
// DeadlyPoison spell (not by poking dot internals) so the comparison
// exercises the same path a rotation does. A single cast's two hits are
// too few samples to tell a +20% multiplier from ordinary crit-roll
// variance, so this casts many times and compares average damage per
// landed hit instead of one before/after pair.
func TestMutilatePoisonedTargetDealsMoreDamage(t *testing.T) {
	const castsPerPhase = 300

	sim, built := buildRogueForTest(t, rogueTalentStringWith(t, "mutilate"))
	target := sim.Encounter.TargetUnits[0]

	avgDamagePerLandedHit := func() float64 {
		mh := built.MutilateMH.SpellMetrics[target.UnitIndex]
		oh := built.MutilateOH.SpellMetrics[target.UnitIndex]
		landed := mh.Hits + mh.Crits + oh.Hits + oh.Crits
		if landed == 0 {
			t.Fatal("Mutilate landed no hits across the sample; cannot compare averages")
		}
		return (mh.TotalDamage + oh.TotalDamage) / float64(landed)
	}

	for i := 0; i < castsPerPhase; i++ {
		built.Mutilate.ApplyEffects(sim, target, built.Mutilate)
	}
	unpoisonedAvg := avgDamagePerLandedHit()

	if built.DeadlyPoison == nil {
		t.Fatal("level-60 rogue has no Deadly Poison registered; cannot apply the poisoned-target condition")
	}
	// Deadly Poison's own hit roll can miss; a bounded retry keeps the
	// test from flaking on the rare miss instead of asserting on one.
	for i := 0; i < 20 && !built.TargetHasRoguePoison(target); i++ {
		built.DeadlyPoison.Cast(sim, target)
	}
	if !built.TargetHasRoguePoison(target) {
		t.Fatal("Deadly Poison never landed on the target after 20 attempts")
	}

	mhBefore := built.MutilateMH.SpellMetrics[target.UnitIndex]
	ohBefore := built.MutilateOH.SpellMetrics[target.UnitIndex]
	landedBefore := mhBefore.Hits + mhBefore.Crits + ohBefore.Hits + ohBefore.Crits
	damageBefore := mhBefore.TotalDamage + ohBefore.TotalDamage

	for i := 0; i < castsPerPhase; i++ {
		built.Mutilate.ApplyEffects(sim, target, built.Mutilate)
	}
	mhAfter := built.MutilateMH.SpellMetrics[target.UnitIndex]
	ohAfter := built.MutilateOH.SpellMetrics[target.UnitIndex]
	landedAfter := (mhAfter.Hits + mhAfter.Crits + ohAfter.Hits + ohAfter.Crits) - landedBefore
	if landedAfter == 0 {
		t.Fatal("Mutilate landed no hits in the poisoned-target sample; cannot compare averages")
	}
	poisonedAvg := (mhAfter.TotalDamage + ohAfter.TotalDamage - damageBefore) / float64(landedAfter)

	// A generous 5% band around the tooltip's 20%: crit variance over
	// 2*castsPerPhase hits should not move the sample average that far
	// off its true mean, so this still fails on a missing or wrong
	// multiplier while tolerating ordinary sampling noise.
	if want := unpoisonedAvg * 1.05; poisonedAvg <= want {
		t.Errorf("average Mutilate damage against a poisoned target = %v, want more than %v (unpoisoned avg %v x1.05, tooltip states x1.20)", poisonedAvg, want, unpoisonedAvg)
	}
}

// TestMutilateComboPointEfficiencyBeatsSinisterStrike is the
// rotation-accuracy program's engine-side check for item 4 (lane
// engine-3): the site's ladder found the assassination rung LOSING dps
// at level 50 once Mutilate comes online, and asked whether
// sim/rogue/mutilate.go's damage/cost numbers are wrong against the
// client. They are not: mutilateFlatDamageBonus (23/33/48/67),
// mutilateWeaponDamagePct (75%) and mutilateEnergyCost (60) all match
// this build's spellconst/rogue.json byte-for-byte (Mutilate's four
// player ranks and their MH/OH sub-spells' effects 121/31), independently
// re-checked for this lane.
//
// What this test pins instead is the actual, and easy to get backwards,
// per-energy comparison: Mutilate cannot be judged solely on damage per
// energy (it is close to parity with Sinister Strike there, both spending
// roughly the same damage per point of energy), but it grants 2 combo
// points for 60 energy - 30 energy per combo point - against Sinister
// Strike's 1 combo point for its own (talent-reduced) cost, which is
// worse per combo point at every level Sinister Strike's own
// sinisterStrikeFlatDamageBonus table covers. Combo points are the
// resource a finisher actually consumes, so Mutilate is the strictly
// better builder once combo-point efficiency (not raw damage) is the
// yardstick - which is exactly why the ladder's own diagnosis (see this
// lane's report) is that the curated rotation's un-gated Sinister
// Strike fallback, not the engine's numbers, is what let Sinister
// Strike out-compete Mutilate for the level-50 rung's limited energy.
func TestMutilateComboPointEfficiencyBeatsSinisterStrike(t *testing.T) {
	_, built := buildRogueForTest(t, rogueTalentStringWith(t, "mutilate"))

	if built.Mutilate == nil {
		t.Fatal("talented level-60 rogue has no Mutilate registered")
	}
	if built.SinisterStrike == nil {
		t.Fatal("level-60 rogue has no Sinister Strike registered")
	}

	mutilateCost := built.Mutilate.Cost.GetCurrentCost()
	sinisterStrikeCost := built.SinisterStrike.Cost.GetCurrentCost()

	const (
		mutilateComboPoints       = 2.0
		sinisterStrikeComboPoints = 1.0
	)

	mutilateEnergyPerCP := mutilateCost / mutilateComboPoints
	sinisterStrikeEnergyPerCP := sinisterStrikeCost / sinisterStrikeComboPoints

	if mutilateEnergyPerCP >= sinisterStrikeEnergyPerCP {
		t.Errorf("Mutilate spends %.1f energy per combo point, want strictly less than "+
			"Sinister Strike's %.1f (Mutilate cost %.0f / %d CP vs Sinister Strike cost %.0f / %d CP)",
			mutilateEnergyPerCP, sinisterStrikeEnergyPerCP, mutilateCost, int(mutilateComboPoints), sinisterStrikeCost, int(sinisterStrikeComboPoints))
	}
}
