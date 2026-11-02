package item_effects_test

import (
	"math"
	"testing"
	"time"

	engine "github.com/wowsims/classic/sim"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// newKindlingStaveWarlock is a level-60 warlock wielding Kindling Stave
// (11750) with no talents, no demon and no buffs.
func newKindlingStaveWarlock(t *testing.T) (*core.Simulation, *core.Character) {
	t.Helper()
	engine.RegisterAll()

	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: 11750}

	player := core.WithSpec(&proto.Player{
		Name:      "Kindling Stave Test",
		Race:      proto.Race_RaceOrc,
		Class:     proto.Class_ClassWarlock,
		Level:     60,
		Equipment: &proto.EquipmentSpec{Items: items},
		Rotation:  &proto.APLRotation{},
	}, &proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.WarlockOptions{
		Armor:  proto.WarlockOptions_NoArmor,
		Summon: proto.WarlockOptions_NoSummon,
	}}})

	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	return sim, sim.Raid.Parties[0].Players[0].GetCharacter()
}

// Kindling Stave's equip "Ignition" (client ItemXItemEffect -> 1300781:
// aura 42 on harmful spell casts, ProcChance 100, no SpellProcsPerMinute
// row, 10 ms recovery; text "have a chance") triggers 1300782, 70 Fire
// damage at Variance 0.3 ("60 to 81", Wowhead Forever). It reads the same
// unstated rate as Necropile Drain: one a minute, so 5% of 3 s casts.
func TestKindlingStaveIgnitionFiresAtOneAMinute(t *testing.T) {
	sim, character := newKindlingStaveWarlock(t)
	target := sim.Encounter.TargetUnits[0]

	proc := character.GetAura("Ignition")
	if proc == nil {
		t.Fatal("Kindling Stave carries no Ignition proc")
	}
	ignition := character.GetSpell(core.ActionID{SpellID: 1300782})
	if ignition == nil || ignition.SpellSchool != core.SpellSchoolFire {
		t.Fatal("Kindling Stave registers no Fire Ignition (1300782) spell")
	}

	const trials = 40_000
	bolt := &core.Spell{ProcMask: core.ProcMaskSpellDamage, CurCast: core.Cast{CastTime: 3 * time.Second}}
	start := sim.CurrentTime
	for i := 0; i < trials; i++ {
		sim.CurrentTime = start + time.Duration(1+i%100)*10*time.Millisecond
		proc.OnSpellHitDealt(proc, sim, bolt, &core.SpellResult{Outcome: core.OutcomeHit, Target: target, Damage: 1})
	}

	metrics := ignition.SpellMetrics[target.UnitIndex]
	want := trials * 0.05
	if got := float64(metrics.Casts); math.Abs(got-want) > want*0.1 {
		t.Errorf("%d harmful 3 s casts fired Ignition %v times, want about %v (one a minute)", trials, got, want)
	}
	if metrics.Hits == 0 {
		t.Fatal("Ignition never landed")
	}
	if avg := metrics.TotalDamage / float64(metrics.Hits); avg < 50 || avg > 110 {
		t.Errorf("Ignition averages %.1f a landed hit, want about 70 (60 to 81, crits 1.5x, less partial resists)", avg)
	}
}
