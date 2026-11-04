package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// WrackOnlyTalentsString sets only the Affliction capstone Wrack (the
// last field in the Affliction tree, 17 nodes) so the test isolates the
// spell this lane added rather than depending on a full leveling build.
const WrackOnlyTalentsString = "00000000000000001"

// TestWrackLevel60DealsPeriodicDamageAndAppliesVulnerability casts Wrack
// on a level-60 warlock with the talent and checks: the spell is
// registered (registerWrackSpell is gated on warlock.Talents.Wrack), its
// DoT actually ticks for damage (not just a registered *Spell wired to
// nothing), and the companion vulnerability aura (the talent's "+10%
// damage taken from your other Shadow DoTs") comes up alongside it.
func TestWrackLevel60DealsPeriodicDamageAndAppliesVulnerability(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassWarlock,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			Buffs:         core.FullBuffs.Player,
			TalentsString: WrackOnlyTalentsString,
		},
		&proto.Player_Warlock{
			Warlock: &proto.Warlock{
				Options: &proto.WarlockOptions{
					Armor:       proto.WarlockOptions_NoArmor,
					Summon:      proto.WarlockOptions_NoSummon,
					WeaponImbue: proto.WarlockOptions_NoWeaponImbue,
				},
			},
		},
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

	agent, ok := sim.Raid.Parties[0].Players[0].(WarlockAgent)
	if !ok {
		t.Fatal("the raid's first player is not a warlock agent")
	}
	built := agent.GetWarlock()

	if built.Wrack == nil {
		t.Fatal("level-60 warlock with the Wrack talent has no Wrack spell registered")
	}
	if got, want := built.Wrack.ActionID.SpellID, int32(WrackSpellID); got != want {
		t.Errorf("Wrack spell ID = %d, want %d", got, want)
	}

	target := sim.Encounter.TargetUnits[0]

	// Drive ApplyEffects directly, same as the hunter Aimed Shot
	// regression test, so the assertion isn't gated on GCD/mana
	// bookkeeping.
	built.Wrack.ApplyEffects(sim, target, built.Wrack)

	if !built.Wrack.Dot(target).IsActive() {
		t.Fatal("Wrack landed but its DoT is not active on the target")
	}
	if aura := built.WrackVulnerabilityAuras.Get(target); aura == nil || !aura.IsActive() {
		t.Error("Wrack landed but its vulnerability aura is not active on the target")
	}

	// Run one tick through the DoT's own handler: Wrack is a channel, and
	// stepping the sim to its ticks would need a rotation to end it.
	dot := built.Wrack.Dot(target)
	dot.OnTick(sim, target, dot)

	if got := built.Wrack.SpellMetrics[target.UnitIndex].TotalDamage; got <= 0 {
		t.Errorf("Wrack dealt %v total damage, want > 0", got)
	}
}

// TestWrackTickReadsThe70755Row pins the 70755 hotfix push on 1316697
// effect 0: 36 a tick at the spell's level 40, +2.5 a caster level up to
// MaxLevel 60 (86 a tick at 60), and a 0.286 spell-power coefficient
// (was 0.143, with no per-level growth). Effect 1's +10% vulnerability is
// unchanged.
func TestWrackTickReadsThe70755Row(t *testing.T) {
	_, warlock, target := newWarlockForDamageTest(t, WrackOnlyTalentsString)
	if warlock.Wrack == nil {
		t.Fatal("level-60 warlock with the Wrack talent has no Wrack registered")
	}
	if got, want := WrackTickDamage.Center(60), 86.0; !floatsNearlyEqual(got, want) {
		t.Errorf("Wrack tick at level 60 = %v, want %v", got, want)
	}
	if got, want := WrackTickDamage.Center(40), 36.0; !floatsNearlyEqual(got, want) {
		t.Errorf("Wrack tick at level 40 = %v, want %v", got, want)
	}
	if got := warlock.Wrack.ClientBaseDamage; !floatsNearlyEqual(got[0], 86) || !floatsNearlyEqual(got[1], 86) {
		t.Errorf("Wrack ClientBaseDamage at 60 = %v, want 86-86", got)
	}
	if got, want := warlock.Wrack.Dot(target).BonusCoefficient, 0.286; !floatsNearlyEqual(got, want) {
		t.Errorf("Wrack tick coefficient = %v, want %v", got, want)
	}
	if got, want := WrackVulnerabilityBonus, 0.10; got != want {
		t.Errorf("Wrack vulnerability = %v, want %v", got, want)
	}
}
