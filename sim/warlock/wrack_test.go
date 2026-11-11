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

// TestWrackLevel60DealsPeriodicDamage casts Wrack on a level-60 warlock
// with the talent and checks that the spell is registered
// (registerWrackSpell is gated on warlock.Talents.Wrack) and that its DoT
// actually ticks for damage, not just a registered *Spell wired to nothing.
func TestWrackLevel60DealsPeriodicDamage(t *testing.T) {
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

	// Run one tick through the DoT's own handler: Wrack is a channel, and
	// stepping the sim to its ticks would need a rotation to end it.
	dot := built.Wrack.Dot(target)
	dot.OnTick(sim, target, dot)

	if got := built.Wrack.SpellMetrics[target.UnitIndex].TotalDamage; got <= 0 {
		t.Errorf("Wrack dealt %v total damage, want > 0", got)
	}
}

// TestWrackTickReadsThe70888Push pins the 10 Nov hotfix ("Wrack damage
// reduced by 30%"), pushed to 1.60.1.70888 as push 113766 on SpellEffect
// 1349556 (1316697 effect 0): 25.2 a tick at the spell's level 40, +1.75 a
// caster level up to MaxLevel 60 (60.2 a tick at 60), and a 0.2002
// spell-power coefficient. Each is 70% of the 70755 row (36, 2.5, 0.286).
func TestWrackTickReadsThe70888Push(t *testing.T) {
	_, warlock, target := newWarlockForDamageTest(t, WrackOnlyTalentsString)
	if warlock.Wrack == nil {
		t.Fatal("level-60 warlock with the Wrack talent has no Wrack registered")
	}
	if got, want := WrackTickDamage.Center(60), 60.2; !floatsNearlyEqual(got, want) {
		t.Errorf("Wrack tick at level 60 = %v, want %v", got, want)
	}
	if got, want := WrackTickDamage.Center(40), 25.2; !floatsNearlyEqual(got, want) {
		t.Errorf("Wrack tick at level 40 = %v, want %v", got, want)
	}
	if got := warlock.Wrack.ClientBaseDamage; !floatsNearlyEqual(got[0], 60.2) || !floatsNearlyEqual(got[1], 60.2) {
		t.Errorf("Wrack ClientBaseDamage at 60 = %v, want 60.2-60.2", got)
	}
	if got, want := warlock.Wrack.Dot(target).BonusCoefficient, 0.2002; !floatsNearlyEqual(got, want) {
		t.Errorf("Wrack tick coefficient = %v, want %v", got, want)
	}
}
