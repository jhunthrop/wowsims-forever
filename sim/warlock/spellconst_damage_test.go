package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// newBareWarlockForDamageTest builds a level-60, zero-talent, zero-gear
// warlock against the standard level-63 dummy, the same shape the
// rotation ladder uses for its bare characters. It exists so this
// file's tests can check a spell's own base damage against
// spellconst/warlock.json's per-rank "amount" without any spell power,
// crit rating or talent multiplier in the way.
func newBareWarlockForDamageTest(t *testing.T) (*core.Simulation, *Warlock, *core.Unit) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:     proto.Class_ClassWarlock,
			Race:      proto.Race_RaceOrc,
			Level:     60,
			Equipment: &proto.EquipmentSpec{},
			Buffs:     core.FullBuffs.Player,
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
	target := sim.Encounter.TargetUnits[0]
	return sim, built, target
}

// averageDamagePerCast casts spell n times against target (with a fresh
// sim.Reset each time so cooldowns/DoTs never gate a later cast) and
// returns the mean of SpellMetrics.TotalDamage across all n casts. At 0
// spell power and 0 talent bonuses this isolates the registered base
// damage (plus whatever crit/partial-resist variance the hit table
// itself adds), which is what distinguishes today's flat client
// "amount" from the roughly 2x larger classic tooltip roll every one of
// these spells carried before the rotation-accuracy audit.
func averageDamagePerCast(t *testing.T, n int, newSim func() (*core.Simulation, *core.Unit, *core.Spell)) float64 {
	t.Helper()
	total := 0.0
	for i := 0; i < n; i++ {
		sim, target, spell := newSim()
		spell.ApplyEffects(sim, target, spell)
		// Some of these spells (Shadow Bolt, Soul Fire) defer DealDamage
		// behind spell.WaitTravelTime, a pending action the sim only
		// resolves on Step - same as the Wrack regression test's own
		// step loop below.
		for step := 0; step < 50; step++ {
			if spell.SpellMetrics[target.UnitIndex].TotalDamage > 0 {
				break
			}
			if done := sim.Step(); done {
				break
			}
		}
		total += spell.SpellMetrics[target.UnitIndex].TotalDamage
	}
	return total / float64(n)
}

// TestShadowBoltRank10DamageMatchesSpellconst guards against
// shadowbolt.go's baseDamage regressing back to the classic tooltip
// roll ({482, 538}, average ~510) instead of spellconst/warlock.json's
// own flat rank-10 "amount" (25307: 268, sp_coefficient 0.857
// unchanged) - corroborated by wowhead's Forever page for 25307
// showing a single "Value: 269" with no min-max tooltip.
func TestShadowBoltRank10DamageMatchesSpellconst(t *testing.T) {
	const wantBase = 268.0
	const oldClassicAverage = 510.0

	got := averageDamagePerCast(t, 50, func() (*core.Simulation, *core.Unit, *core.Spell) {
		sim, built, target := newBareWarlockForDamageTest(t)
		if len(built.ShadowBolt) == 0 {
			t.Fatal("level-60 warlock has no Shadow Bolt registered")
		}
		return sim, target, built.ShadowBolt[len(built.ShadowBolt)-1]
	})

	if got < wantBase*0.9 || got > wantBase*2.1 {
		t.Errorf("Shadow Bolt rank 10 average damage = %.1f, want close to the client's flat %.1f", got, wantBase)
	}
	if got > oldClassicAverage*0.75 {
		t.Errorf("Shadow Bolt rank 10 average damage = %.1f, still in range of the old classic roll's ~%.1f average - the fix did not take", got, oldClassicAverage)
	}
}

// TestSearingPainRank6DamageMatchesSpellconst guards searing_pain.go's
// baseDamage against regressing to the classic roll ({208, 244}, average
// ~226) instead of spellconst's flat rank-6 amount (17923: 114,
// sp_coefficient 0.429 unchanged).
func TestSearingPainRank6DamageMatchesSpellconst(t *testing.T) {
	const wantBase = 114.0
	const oldClassicAverage = 226.0

	got := averageDamagePerCast(t, 50, func() (*core.Simulation, *core.Unit, *core.Spell) {
		sim, built, target := newBareWarlockForDamageTest(t)
		if len(built.SearingPain) == 0 {
			t.Fatal("level-60 warlock has no Searing Pain registered")
		}
		return sim, target, built.SearingPain[len(built.SearingPain)-1]
	})

	if got < wantBase*0.9 || got > wantBase*2.1 {
		t.Errorf("Searing Pain rank 6 average damage = %.1f, want close to the client's flat %.1f", got, wantBase)
	}
	if got > oldClassicAverage*0.75 {
		t.Errorf("Searing Pain rank 6 average damage = %.1f, still in range of the old classic roll's ~%.1f average - the fix did not take", got, oldClassicAverage)
	}
}

// TestSoulFireRank2DamageMatchesSpellconst guards soul_fire.go's
// baseDamage against regressing to the classic roll ({715, 894},
// average ~804.5) instead of spellconst's flat rank-2 amount (17924:
// 431, sp_coefficient 1.0 unchanged).
func TestSoulFireRank2DamageMatchesSpellconst(t *testing.T) {
	const wantBase = 431.0
	const oldClassicAverage = 804.5

	got := averageDamagePerCast(t, 50, func() (*core.Simulation, *core.Unit, *core.Spell) {
		sim, built, target := newBareWarlockForDamageTest(t)
		if len(built.SoulFire) == 0 {
			t.Fatal("level-60 warlock has no Soul Fire registered")
		}
		return sim, target, built.SoulFire[len(built.SoulFire)-1]
	})

	if got < wantBase*0.9 || got > wantBase*2.1 {
		t.Errorf("Soul Fire rank 2 average damage = %.1f, want close to the client's flat %.1f", got, wantBase)
	}
	if got > oldClassicAverage*0.75 {
		t.Errorf("Soul Fire rank 2 average damage = %.1f, still in range of the old classic roll's ~%.1f average - the fix did not take", got, oldClassicAverage)
	}
}

// TestShadowburnRank6DamageMatchesSpellconst guards shadowburn.go's
// baseDamage against regressing to the classic roll ({462, 514},
// average ~488) instead of spellconst's flat rank-6 amount (18871: 266,
// sp_coefficient 0.429 unchanged), the Destruction talent Shadowburn
// needed to register at all.
func TestShadowburnRank6DamageMatchesSpellconst(t *testing.T) {
	const wantBase = 266.0
	const oldClassicAverage = 488.0

	got := averageDamagePerCast(t, 50, func() (*core.Simulation, *core.Unit, *core.Spell) {
		sim, built, target := newBareWarlockForDamageTest(t)
		built.Talents.Shadowburn = true
		built.registerShadowBurnSpell()
		if len(built.Shadowburn) == 0 {
			t.Fatal("level-60 warlock with the Shadowburn talent has no Shadowburn registered")
		}
		return sim, target, built.Shadowburn[len(built.Shadowburn)-1]
	})

	if got < wantBase*0.9 || got > wantBase*2.1 {
		t.Errorf("Shadowburn rank 6 average damage = %.1f, want close to the client's flat %.1f", got, wantBase)
	}
	if got > oldClassicAverage*0.75 {
		t.Errorf("Shadowburn rank 6 average damage = %.1f, still in range of the old classic roll's ~%.1f average - the fix did not take", got, oldClassicAverage)
	}
}

// TestConflagrateRank6DamageMatchesSpellconst guards conflagrate.go's
// baseDamage against regressing to the classic roll ({447, 557},
// average ~502) instead of spellconst's flat rank-6 amount (18932: 282,
// sp_coefficient 0.429 unchanged).
func TestConflagrateRank6DamageMatchesSpellconst(t *testing.T) {
	const wantBase = 282.0
	const oldClassicAverage = 502.0

	got := averageDamagePerCast(t, 50, func() (*core.Simulation, *core.Unit, *core.Spell) {
		sim, built, target := newBareWarlockForDamageTest(t)
		built.Talents.Conflagrate = true
		built.registerConflagrateSpell()
		if len(built.Conflagrate) == 0 {
			t.Fatal("level-60 warlock with the Conflagrate talent has no Conflagrate registered")
		}
		// Conflagrate requires Immolate up on the target; apply the
		// max-rank Immolate first so ExtraCastCondition allows the cast.
		built.registerImmolateSpell()
		immolate := built.Immolate[len(built.Immolate)-1]
		immolate.ApplyEffects(sim, target, immolate)
		return sim, target, built.Conflagrate[len(built.Conflagrate)-1]
	})

	if got < wantBase*0.9 || got > wantBase*2.1 {
		t.Errorf("Conflagrate rank 6 average damage = %.1f, want close to the client's flat %.1f", got, wantBase)
	}
	if got > oldClassicAverage*0.75 {
		t.Errorf("Conflagrate rank 6 average damage = %.1f, still in range of the old classic roll's ~%.1f average - the fix did not take", got, oldClassicAverage)
	}
}

// TestCorruptionRank7SnapshotMatchesSpellconst checks the exact
// per-tick snapshot value (no crit/hit-table noise to average out for a
// DoT): corruption.go's baseDamage against regressing to the classic
// total-over-ticks value (822/6 = 137/tick) instead of spellconst's own
// flat rank-7 per-tick amount (25311: 73, period_ms 3000 unchanged).
func TestCorruptionRank7SnapshotMatchesSpellconst(t *testing.T) {
	const want = 73.0

	// A hit-table miss would leave the DoT unapplied, so retry with a
	// fresh sim (new RNG draw) until it lands - the snapshot value
	// itself does not depend on the roll, only whether one landed.
	for attempt := 0; attempt < 20; attempt++ {
		sim, built, target := newBareWarlockForDamageTest(t)
		if len(built.Corruption) == 0 {
			t.Fatal("level-60 warlock has no Corruption registered")
		}
		spell := built.Corruption[len(built.Corruption)-1]
		spell.ApplyEffects(sim, target, spell)

		dot := spell.Dot(target)
		if !dot.IsActive() {
			continue
		}
		if got := dot.SnapshotBaseDamage; got != want {
			t.Errorf("Corruption rank 7 per-tick snapshot damage = %.2f, want %.2f (spellconst 25311 amount)", got, want)
		}
		return
	}
	t.Fatal("Corruption never landed in 20 attempts")
}
