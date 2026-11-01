package mage

import (
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// TestIgniteDoesNotCountDamageBonusesTwice pins Forever's 24 September
// 2026 beta note ("Ignite: no longer counts damage bonuses twice"): the
// burn is a share of the crit's finished damage, which already carries
// every attacker multiplier, so the dot must snapshot a multiplier of 1
// rather than apply the mage's damage multipliers a second time on each
// tick, as the vanilla engine did.
// newIgniteMageSim builds a level-60 Troll mage with Ignite 5/5 and Fire
// Power 5/5 and no rotation, so the test drives every cast itself.
func newIgniteMageSim(t *testing.T) (*core.Simulation, *Mage) {
	t.Helper()
	talentsStr := talentStringWithRank(t, ForeverFrostTalents, "ignite", 5)
	talentsStr = talentStringWithRank(t, talentsStr, "fire_power", 5)

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassMage,
			Race:               proto.Race_RaceTroll,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              core.FullBuffs.Player,
			TalentsString:      talentsStr,
			DistanceFromTarget: 5,
		},
		PlayerOptions,
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

	agent, ok := sim.Raid.Parties[0].Players[0].(MageAgent)
	if !ok {
		t.Fatal("the raid's first player is not a mage agent")
	}
	built := agent.GetMage()
	if built.Talents.Ignite != 5 || built.igniteTick == nil {
		t.Fatalf("Ignite 5/5 expected with its dot registered; got rank %d", built.Talents.Ignite)
	}

	return sim, built
}

func TestIgniteDoesNotCountDamageBonusesTwice(t *testing.T) {
	sim, built := newIgniteMageSim(t)
	target := sim.Encounter.TargetUnits[0]
	fireball := built.Fireball[core.MaxTrainerRank(FireballRanks)]
	if fireball == nil {
		t.Fatal("level-60 mage has no top-rank Fireball")
	}
	// Force the crit Ignite rolls off, instead of relying on the seed.
	fireball.BonusCritRating += 100 * core.CritRatingPerCritChance
	fireball.ApplyEffects(sim, target, fireball)
	waitForOutcomes(sim)

	dot := built.igniteTick.Dot(target)
	if dot == nil || !dot.IsActive() {
		t.Fatal("a Fireball crit did not apply Ignite")
	}
	if got := dot.SnapshotAttackerMultiplier; got != 1 {
		t.Errorf("Ignite SnapshotAttackerMultiplier = %v, want 1: the crit's damage already carries Fire Power and every other attacker multiplier", got)
	}
	if dot.SnapshotBaseDamage <= 0 {
		t.Errorf("Ignite SnapshotBaseDamage = %v, want > 0", dot.SnapshotBaseDamage)
	}
}

// Forever's Ignite ticks crit (the owner's periodic crit rule: every true
// dot crits), on the spell table with the spell multiplier. A tick crit
// must not feed Ignite: the talent's proc mask (spell 11119,
// SpellAuraOptions ProcTypeMask 0x10000, harmful spell damage) has no
// periodic bit, so only the triggering Fireball creates the burn and the
// burn runs its two ticks and ends.
func TestIgniteTicksCritWithoutFeedingIgnite(t *testing.T) {
	sim, built := newIgniteMageSim(t)
	target := sim.Encounter.TargetUnits[0]
	fireball := built.Fireball[core.MaxTrainerRank(FireballRanks)]
	fireball.BonusCritRating += 100 * core.CritRatingPerCritChance
	built.igniteTick.BonusCritRating += 100 * core.CritRatingPerCritChance

	fireball.ApplyEffects(sim, target, fireball)
	waitForOutcomes(sim)
	dot := built.igniteTick.Dot(target)
	if !dot.IsActive() || dot.GetStacks() != 1 {
		t.Fatalf("a Fireball crit should apply one Ignite stack; active %v, stacks %d", dot.IsActive(), dot.GetStacks())
	}

	maxStacks := dot.GetStacks()
	deadline := sim.CurrentTime + 3*IgniteTicks*time.Second
	for sim.CurrentTime < deadline && !sim.Step() {
		maxStacks = max(maxStacks, dot.GetStacks())
	}

	m := built.igniteTick.SpellMetrics[target.UnitIndex]
	if m.CritTicks != IgniteTicks {
		t.Errorf("Ignite CritTicks = %d (plain %d), want %d at 100%% crit", m.CritTicks, m.Ticks, IgniteTicks)
	}
	if maxStacks != 1 || m.CritTicks+m.Ticks != IgniteTicks {
		t.Errorf("Ignite's own crits fed Ignite: peak stacks %d, ticks %d, want 1 stack and %d ticks", maxStacks, m.CritTicks+m.Ticks, IgniteTicks)
	}
}

// Blizzard's beta note reads "Ignite no longer double dips on % damage
// increase modifiers". The crit Ignite shares already carries the
// target's percentage modifiers (Improved Scorch's Fire vulnerability,
// Curse of the Elements), so the burn's ticks must not take them again.
func TestIgniteTicksSkipTheTargetsDamageTakenModifiers(t *testing.T) {
	sim, built := newIgniteMageSim(t)
	target := sim.Encounter.TargetUnits[0]
	core.ImprovedScorchAura(target).Activate(sim)
	core.ImprovedScorchAura(target).SetStacks(sim, 5)

	attackTable := built.AttackTables[target.UnitIndex][built.igniteTick.CastType]
	if got := built.igniteTick.TargetDamageMultiplier(attackTable, true); got != 1 {
		t.Errorf("Ignite's tick takes a target damage multiplier of %v, want 1: the crit it shares already took them", got)
	}
}
