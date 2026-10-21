package retribution

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/paladin"
)

// retSpellFingerprints builds a bare level 60 Retribution paladin wearing
// only the given relic (0 for none) and snapshots every spell it has.
func retSpellFingerprints(t *testing.T, relicID int32, talents string) map[string]core.SpellFingerprint {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               proto.Race_RaceHuman,
		Level:              60,
		Equipment:          core.RelicEquipment(relicID),
		TalentsString:      talents,
		DistanceFromTarget: 5,
	}, PlayerOptionsSealofRighteousness)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)
	return core.SpellFingerprints(&env.Raid.Parties[0].Players[0].GetCharacter().Unit)
}

func changedByRelic(t *testing.T, relicID int32, talents string) []core.SpellChange {
	t.Helper()
	changed := core.ChangedSpells(retSpellFingerprints(t, 0, talents), retSpellFingerprints(t, relicID, talents))
	if len(changed) == 0 {
		t.Fatalf("relic %d changed no spell", relicID)
	}
	return changed
}

// Libram of Law (client spell 1291086): Judgement damage +4%, on Judgement of
// Righteousness only - the client's family does not include Judgement of
// Command, the seal's own proc, or any other spell.
func TestLibramOfLawRaisesJudgementOfRighteousnessDamageOnly(t *testing.T) {
	for _, change := range changedByRelic(t, paladin.LibramOfLaw, Phase45RetTalents) {
		if change.After.ClassSpellMask&paladin.PaladinSpellMaskJudgementOfRighteousness == 0 {
			t.Errorf("%s changed but is not Judgement of Righteousness: %+v", change.Key, change.After)
		}
		if got, want := change.After.DamageMultiplier/change.Before.DamageMultiplier, 1.04; got < want-1e-9 || got > want+1e-9 {
			t.Errorf("%s damage multiplier ratio = %v, want %v", change.Key, got, want)
		}
	}
}

// Libram of Invocation (client spell 1249005): Seal spell mana cost -5%.
func TestLibramOfInvocationCutsSealCastCostOnly(t *testing.T) {
	sealCasts := paladin.PaladinSpellMaskSealOfRighteousnessCast | paladin.PaladinSpellMaskSealOfCommandCast | paladin.PaladinSpellMaskSealOfTheCrusaderCast
	for _, change := range changedByRelic(t, paladin.LibramOfInvocation, Phase45RetTalents) {
		if change.After.ClassSpellMask&sealCasts == 0 {
			t.Errorf("%s changed but is not a Seal cast: %+v", change.Key, change.After)
		}
		if got := change.After.CostMultiplier - change.Before.CostMultiplier; got != -5 {
			t.Errorf("%s cost multiplier moved by %d, want -5", change.Key, got)
		}
	}
}

// Libram of Infusion (client spell 1306429): Holy Shock crit chance +6%. Holy
// Shock is a Holy tree talent, so the paladin here takes it and nothing else.
func TestLibramOfInfusionRaisesHolyShockCritOnly(t *testing.T) {
	for _, change := range changedByRelic(t, paladin.LibramOfInfusion, talentString(t, map[string]int{"holy_shock": 1})) {
		if change.After.ClassSpellMask&paladin.PaladinSpellMaskHolyShock == 0 {
			t.Errorf("%s changed but is not Holy Shock: %+v", change.Key, change.After)
		}
		if got := change.After.BonusCritRating - change.Before.BonusCritRating; got != 6*core.CritRatingPerCritChance {
			t.Errorf("%s bonus crit moved by %v, want 6%%", change.Key, got)
		}
	}
}

const sealOfTheCrusaderRank6 = 20308

// Libram of Fervor (client spell 28852): +48 attack power while Seal of the
// Crusader is up. The seal must give the attack power back when it fades -
// a sign slip on expiry once left the libram's 48 on the paladin for good.
func TestLibramOfFervorAttackPowerReturnsWhenTheSealFades(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Class:     proto.Class_ClassPaladin,
		Race:      proto.Race_RaceHuman,
		Level:     60,
		Equipment: core.RelicEquipment(paladin.LibramOfFervor),
	}, PlayerOptionsSealofRighteousness)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()

	character := sim.Raid.Parties[0].Players[0].GetCharacter()
	seal := character.GetAuraByID(core.ActionID{SpellID: sealOfTheCrusaderRank6})
	if seal == nil {
		t.Fatal("no rank 6 Seal of the Crusader aura on a level 60 paladin")
	}

	idle := character.GetStat(stats.AttackPower)
	seal.Activate(sim)
	const rank6AttackPower = 306.0 + 2.4*(60-52) // the seal's own rank 6 value at level 60 (sotc.go)
	if got := character.GetStat(stats.AttackPower) - idle; math.Abs(got-(rank6AttackPower+48)) > 1e-6 {
		t.Errorf("attack power with the seal up = +%v, want +%v (rank 6 plus the libram's 48)", got, rank6AttackPower+48)
	}
	seal.Deactivate(sim)
	if got := character.GetStat(stats.AttackPower); math.Abs(got-idle) > 1e-6 {
		t.Errorf("attack power after the seal faded = %v, want the idle %v back", got, idle)
	}
}
