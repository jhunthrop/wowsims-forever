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
	talentsStr = talentStringWithRank(t, talentsStr, "improved_scorch", 3)

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

// Forever's Ignite ticks never crit. The talent (11119) applies 412538, a
// dummy periodic aura (226) whose 2 s pulses cast the server tick 412545,
// and 412545 carries SPELL_ATTR2_CANT_CRIT (Attributes_2 0x20000004, 70888
// and 70938): the same flag that keeps Deep Wound's 412613 off the crit
// roll, which overrides the owner's "every true dot crits" rule. The logs
// are silent (3 Ignite ticks in every readable Forever log, none critical).
// A tick must not feed Ignite either: the talent's proc mask has no
// periodic bit, so only the triggering Fireball creates the burn and the
// burn runs its two ticks and ends.
func TestIgniteTicksNeverCrit(t *testing.T) {
	sim, built := newIgniteMageSim(t)
	target := sim.Encounter.TargetUnits[0]
	fireball := built.Fireball[core.MaxTrainerRank(FireballRanks)]
	fireball.BonusCritRating += 100 * core.CritRatingPerCritChance
	built.igniteTick.BonusCritRating += 100 * core.CritRatingPerCritChance

	fireball.ApplyEffects(sim, target, fireball)
	waitForOutcomes(sim)
	dot := built.igniteTick.Dot(target)
	if !dot.IsActive() {
		t.Fatal("a Fireball crit should apply Ignite")
	}

	deadline := sim.CurrentTime + 3*IgniteTicks*time.Second
	for sim.CurrentTime < deadline && !sim.Step() {
	}

	m := built.igniteTick.SpellMetrics[target.UnitIndex]
	if m.CritTicks != 0 || m.Ticks != IgniteTicks {
		t.Errorf("Ignite ticked %d plain and %d critical at a forced crit, want %d plain and none", m.Ticks, m.CritTicks, IgniteTicks)
	}
}

// Blizzard's beta note reads "Ignite no longer double dips on % damage
// increase modifiers". The crit Ignite shares already carries the
// target's percentage modifiers (Improved Scorch's Fire vulnerability,
// Curse of the Elements), so the burn's ticks must not take them again.
func TestIgniteTicksSkipTheTargetsDamageTakenModifiers(t *testing.T) {
	sim, built := newIgniteMageSim(t)
	target := sim.Encounter.TargetUnits[0]
	// The mage's own Fire Vulnerability: Forever's is the caster's own, so
	// the raid's Improved Scorch flag puts none on the target.
	vulnerability := built.ImprovedScorchAuras.Get(target)
	vulnerability.Activate(sim)
	vulnerability.SetStacks(sim, 5)

	attackTable := built.AttackTables[target.UnitIndex][built.igniteTick.CastType]
	if got := built.igniteTick.TargetDamageMultiplier(attackTable, true); got != 1 {
		t.Errorf("Ignite's tick takes a target damage multiplier of %v, want 1: the crit it shares already took them", got)
	}
}

// TestIgniteRollsCritsIntoOneBurn: Forever's Ignite is one aura that a
// new crit refreshes (client 412538: CumulativeAura 1, a 4 s, 2 s-tick
// burn of $w1/2; the beta logs show only debuff_applied and
// debuff_refreshed, never a second stack, with up to six crits in one
// burn). Each crit adds its 40% to what is left, so the whole burn pays
// 40% of every crit once, however the crits fall between ticks. The old
// stacking model restarted both ticks for every earlier stack on each
// refresh and paid an earlier crit again.
func TestIgniteRollsCritsIntoOneBurn(t *testing.T) {
	sim, built := newIgniteMageSim(t)
	target := sim.Encounter.TargetUnits[0]
	fireball := built.Fireball[core.MaxTrainerRank(FireballRanks)]
	fireball.BonusCritRating += 100 * core.CritRatingPerCritChance
	built.igniteTick.BonusCritRating -= 1000 * core.CritRatingPerCritChance

	crits := 0.0
	cast := func() {
		before := fireball.SpellMetrics[target.UnitIndex]
		fireball.ApplyEffects(sim, target, fireball)
		waitForOutcomes(sim)
		after := fireball.SpellMetrics[target.UnitIndex]
		crits += (after.TotalDamage - after.TotalTickDamage) - (before.TotalDamage - before.TotalTickDamage)
	}

	cast()
	dot := built.igniteTick.Dot(target)
	for dot.TickCount < 1 && !sim.Step() {
	} // one tick of the first burn lands, one is left
	if !dot.IsActive() || dot.TickCount != 1 {
		t.Fatalf("the first burn should have one tick left; active %v, ticks %d", dot.IsActive(), dot.TickCount)
	}
	cast()
	for deadline := sim.CurrentTime + 10*time.Second; sim.CurrentTime < deadline && !sim.Step(); {
	}

	ignite := built.igniteTick.SpellMetrics[target.UnitIndex].TotalTickDamage
	if want := 0.4 * crits; ignite < want*0.97 || ignite > want*1.03 {
		t.Errorf("two crits a tick apart burned %.1f, want 40%% of %.1f = %.1f", ignite, crits, want)
	}
}

// TestIgniteRegistersOnce: the burn is one spell under 12654. A second
// registration under the same id (the old trigger spell) made GetSpell,
// and so any APL line naming Ignite, resolve to a spell with no dot.
func TestIgniteRegistersOnce(t *testing.T) {
	sim, built := newIgniteMageSim(t)
	target := sim.Encounter.TargetUnits[0]
	count := 0
	for _, spell := range built.Spellbook {
		if spell.ActionID.SameAction(core.ActionID{SpellID: 12654}) {
			count++
		}
	}
	if count != 1 {
		t.Errorf("%d spells registered under Ignite's 12654, want 1", count)
	}
	if s := built.GetSpell(core.ActionID{SpellID: 12654}); s == nil || s.Dot(target) == nil {
		t.Error("GetSpell(12654) does not resolve to Ignite's burn")
	}
}
