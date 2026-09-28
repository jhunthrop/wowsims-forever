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

	// Step the sim so the DoT's periodic ticks (1 s apart, 6 of them)
	// actually fire and deal damage.
	for i := 0; i < 1000; i++ {
		if built.Wrack.SpellMetrics[target.UnitIndex].TotalDamage > 0 {
			break
		}
		if done := sim.Step(); done {
			break
		}
	}

	if got := built.Wrack.SpellMetrics[target.UnitIndex].TotalDamage; got <= 0 {
		t.Errorf("Wrack dealt %v total damage, want > 0", got)
	}
}
