package item_effects_test

// The item procs the published BiS sets wear, rolled against the 70707
// client's SpellEffect rows (EffectBasePointsF as the centre, Variance as
// the whole width of the roll) and the server's own tooltip text on Wowhead
// Forever (nether.wowhead.com/forever/tooltip/spell/<id>, read 2026-11-02).
// Sim audit round 8, items lane: the engine still rolled Classic Era's
// numbers for most of them, and did not implement the rest.

import (
	"math"
	"testing"
	"time"

	engine "github.com/wowsims/classic/sim"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// bareTarget is a level-60 target with no armour and no resistance, so a
// landed non-critical hit is the proc's own roll.
var bareTarget = &proto.Target{
	Level:         60,
	Stats:         stats.Stats{}.ToFloatArray(),
	MobType:       proto.MobType_MobTypeHumanoid,
	SwingSpeed:    2,
	MinBaseDamage: 100,
	DamageSpread:  0.3333,
}

// unbuffedCharacter equips itemID in slot on a level-60 orc of class, with
// no buffs, consumables or talents.
func unbuffedCharacter(t *testing.T, class proto.Class, slot proto.ItemSlot, itemID int32) (*core.Simulation, *core.Character) {
	t.Helper()
	return unbuffedCharacterIn(t, class, slot, itemID, &proto.Encounter{Duration: 600, Targets: []*proto.Target{bareTarget}})
}

func unbuffedCharacterIn(t *testing.T, class proto.Class, slot proto.ItemSlot, itemID int32, encounter *proto.Encounter) (*core.Simulation, *core.Character) {
	t.Helper()
	engine.RegisterAll()

	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[slot] = &proto.ItemSpec{Id: itemID}

	player := &proto.Player{
		Name:      "Client Proc Test",
		Race:      proto.Race_RaceOrc,
		Class:     class,
		Level:     60,
		Equipment: &proto.EquipmentSpec{Items: items},
		Rotation:  &proto.APLRotation{},
	}
	switch class {
	case proto.Class_ClassWarrior:
		player.Spec = &proto.Player_Warrior{Warrior: &proto.Warrior{Options: &proto.Warrior_Options{}}}
	case proto.Class_ClassHunter:
		player.Spec = &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{}}}
	case proto.Class_ClassRogue:
		player.Spec = &proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.RogueOptions{}}}
	case proto.Class_ClassMage:
		player.Spec = &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{}}}
	default:
		t.Fatalf("unbuffedCharacter: no spec for %v", class)
	}

	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  encounter,
		SimOptions: &proto.SimOptions{RandomSeed: 7, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	return sim, sim.Raid.Parties[0].Players[0].GetCharacter()
}

// hitRange casts spell n times and returns the smallest and largest damage
// of a landed hit that was not a critical strike.
func hitRange(sim *core.Simulation, spell *core.Spell, n int) (lo, hi float64, hits int) {
	target := sim.Encounter.TargetUnits[0]
	lo, hi = math.Inf(1), math.Inf(-1)
	for i := 0; i < n; i++ {
		metrics := spell.SpellMetrics[target.UnitIndex]
		damage, hitCount, critCount := metrics.TotalDamage, metrics.Hits, metrics.Crits
		spell.Cast(sim, target)
		metrics = spell.SpellMetrics[target.UnitIndex]
		if metrics.Hits == hitCount+1 && metrics.Crits == critCount {
			dealt := metrics.TotalDamage - damage
			lo, hi = min(lo, dealt), max(hi, dealt)
			hits++
		}
		if spell.CD.Timer != nil {
			spell.CD.Reset()
		}
	}
	return lo, hi, hits
}

type clientProc struct {
	name    string
	class   proto.Class
	slot    proto.ItemSlot
	itemID  int32
	spellID int32
	// centre and variance are the client's EffectBasePointsF and Variance.
	centre, variance float64
}

func clientProcs() []clientProc {
	mh, ranged, hands := proto.ItemSlot_ItemSlotMainHand, proto.ItemSlot_ItemSlotRanged, proto.ItemSlot_ItemSlotHands
	warrior, hunter := proto.Class_ClassWarrior, proto.Class_ClassHunter
	return []clientProc{
		// Wowhead Forever: "Sends a shadowy bolt at the enemy causing 161 to 301 Shadow damage."
		{"Nightblade", warrior, mh, 1982, 18211, 231, 0.60869562626},
		// "Launches a bolt of frost ... causing 40 to 60 Frost damage"
		{"Hammer of the Northern Wind", warrior, mh, 810, 13439, 50, 0.40000000596},
		// Forever's own 1293790: "causing 34 to 50 Frost damage"
		{"Coldrage Dagger", warrior, mh, 10761, 1293790, 42, 0.40000000596},
		// "Hurls a fiery ball that causes 231 to 293 Fire damage and an additional 60 damage over 6 sec."
		{"Fiery War Axe", warrior, mh, 870, 18796, 262, 0.23863635957},
		// "Fires a Searing Arrow at the target for 23 to 33 Fire damage."
		{"Bow of Searing Arrows", hunter, ranged, 2825, 29638, 28, 0.36363637447},
		// "Delivers a fatal wound for 92 to 118 Physical damage."
		{"Gut Ripper", warrior, mh, 2164, 18107, 105, 0.24074074626},
		// Unimplemented until round 8.
		// "Sends a shadowy bolt at the enemy causing 77 Shadow damage."
		{"Black Menace", warrior, mh, 6831, 1293407, 77, 0},
		// "Sends a shadowy bolt at the enemy causing 35 Shadow damage."
		{"Claw of the Shadowmancer", warrior, mh, 2912, 16409, 35, 0},
		// "dealing 39 to 59 Holyfire damage"
		{"Verigan's Fist", warrior, mh, 6953, 1322218, 49, 0.41999998689},
		// "Smite your target for 101 to 123 Holy damage."
		{"Gloves of Holy Might", hunter, hands, 867, 1325229, 112, 0.20000000298},
	}
}

func TestItemProcsRollTheClientsDamage(t *testing.T) {
	for _, proc := range clientProcs() {
		t.Run(proc.name, func(t *testing.T) {
			sim, character := unbuffedCharacter(t, proc.class, proc.slot, proc.itemID)
			spell := character.GetSpell(core.ActionID{SpellID: proc.spellID})
			if spell == nil {
				t.Fatalf("%s equipped, but its client proc spell %d is not registered", proc.name, proc.spellID)
			}
			want := [2]float64{proc.centre * (1 - proc.variance/2), proc.centre * (1 + proc.variance/2)}
			lo, hi, hits := hitRange(sim, spell, 3000)
			if hits < 100 {
				t.Fatalf("only %d plain hits in 3000 casts", hits)
			}
			slack := 0.5 + 0.1*(want[1]-want[0])
			if lo < want[0]-0.5 || hi > want[1]+0.5 || lo > want[0]+slack || hi < want[1]-slack {
				t.Errorf("hits rolled %.1f to %.1f, want the client's %.1f to %.1f", lo, hi, want[0], want[1])
			}
		})
	}
}

// Fiery War Axe's Fireball also burns for 20 every 2 sec for 6 sec
// (18796 effect 1: aura 3, base 20, period 2000).
func TestFieryWarAxeBurnsForTheClientsTwentyATick(t *testing.T) {
	sim, character := unbuffedCharacter(t, proto.Class_ClassWarrior, proto.ItemSlot_ItemSlotMainHand, 870)
	spell := character.GetSpell(core.ActionID{SpellID: 18796})
	if spell == nil {
		t.Fatal("no Fiery War Axe proc spell")
	}
	target := sim.Encounter.TargetUnits[0]
	for i := 0; i < 50 && !spell.Dot(target).IsActive(); i++ {
		spell.Cast(sim, target)
	}
	dot := spell.Dot(target)
	if !dot.IsActive() {
		t.Fatal("the Fireball never applied its burn")
	}
	if dot.SnapshotBaseDamage != 20 || dot.NumberOfTicks != 3 {
		t.Errorf("burn is %v x %d ticks, want 20 x 3", dot.SnapshotBaseDamage, dot.NumberOfTicks)
	}
}

// Manual Crowd Pummeler: ItemEffect 98990 is 13494 on a 3 minute cooldown
// with 3 charges ("(3 Min Cooldown) 3 Charges"), not Era's 30 sec.
func TestManualCrowdPummelerIsOnAThreeMinuteCooldown(t *testing.T) {
	_, character := unbuffedCharacter(t, proto.Class_ClassWarrior, proto.ItemSlot_ItemSlotMainHand, 9449)
	spell := character.GetSpell(core.ActionID{SpellID: 13494})
	if spell == nil {
		t.Fatal("no Manual Crowd Pummeler use")
	}
	if spell.CD.Duration != 3*time.Minute {
		t.Errorf("cooldown is %v, want 3m", spell.CD.Duration)
	}
}

// The periodic and on-use effects, against their client rows.
func TestClientPeriodicItemEffects(t *testing.T) {
	cases := []struct {
		name     string
		class    proto.Class
		slot     proto.ItemSlot
		itemID   int32
		spellID  int32
		tick     float64
		ticks    int32
		period   time.Duration
		cooldown time.Duration
	}{
		// 11790: 14 Nature every 3 sec for 12 sec.
		{"Toxic Revenger", proto.Class_ClassWarrior, proto.ItemSlot_ItemSlotMainHand, 9453, 11790, 14, 4, 3 * time.Second, 0},
		// 12257: 75 Fire every 1 sec for 5 sec, on a 5 minute use.
		{"Thunderbrew's Boot Flask", proto.Class_ClassMage, proto.ItemSlot_ItemSlotTrinket1, 744, 12257, 75, 5, time.Second, 5 * time.Minute},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sim, character := unbuffedCharacter(t, c.class, c.slot, c.itemID)
			spell := character.GetSpell(core.ActionID{SpellID: c.spellID})
			if spell == nil {
				t.Fatalf("%s equipped, but %d is not registered", c.name, c.spellID)
			}
			if c.cooldown != 0 && spell.CD.Duration != c.cooldown {
				t.Errorf("cooldown %v, want %v", spell.CD.Duration, c.cooldown)
			}
			target := sim.Encounter.TargetUnits[0]
			for i := 0; i < 50 && !spell.Dot(target).IsActive(); i++ {
				if spell.CD.Timer != nil {
					spell.CD.Reset()
				}
				spell.Cast(sim, target)
			}
			dot := spell.Dot(target)
			if !dot.IsActive() {
				t.Fatal("the effect never applied")
			}
			if dot.SnapshotBaseDamage != c.tick || dot.NumberOfTicks != c.ticks || dot.TickLength != c.period {
				t.Errorf("%v x %d every %v, want %v x %d every %v", dot.SnapshotBaseDamage, dot.NumberOfTicks, dot.TickLength, c.tick, c.ticks, c.period)
			}
		})
	}
}

// 9174: "Increase Rage by 30", once an hour.
func TestRageHauberksGrantThirtyRage(t *testing.T) {
	for _, itemID := range []int32{6972, 7133} {
		sim, character := unbuffedCharacter(t, proto.Class_ClassWarrior, proto.ItemSlot_ItemSlotChest, itemID)
		spell := character.GetSpell(core.ActionID{SpellID: 9174})
		if spell == nil {
			t.Fatalf("item %d equipped, but its rage use is not registered", itemID)
		}
		if spell.CD.Duration != time.Hour {
			t.Errorf("item %d cooldown %v, want 1h", itemID, spell.CD.Duration)
		}
		before := character.CurrentRage()
		spell.Cast(sim, sim.Encounter.TargetUnits[0])
		if got := character.CurrentRage() - before; got != 30 {
			t.Errorf("item %d gave %v rage, want 30", itemID, got)
		}
	}
}

// Aegis of the Scarlet Commander: 1293539, 14 Holy (variance 0.2) on a block.
func TestAegisOfTheScarletCommanderHitsBack(t *testing.T) {
	_, character := unbuffedCharacter(t, proto.Class_ClassWarrior, proto.ItemSlot_ItemSlotOffHand, 7726)
	spell := character.GetSpell(core.ActionID{SpellID: 1293539})
	if spell == nil {
		t.Fatal("no Scarlet Retribution spell")
	}
	// SpellCooldowns RecoveryTime 5000 on 1293539; Wowhead Forever's
	// tooltip prints "5 sec cooldown".
	if spell.CD.Timer == nil || spell.CD.Duration != 5*time.Second {
		t.Errorf("Scarlet Retribution's cooldown is %v, want the client's 5 sec", spell.CD.Duration)
	}
}

// Gut Ripper doubles below 25% health (18107 effects 1 and 2).
func TestGutRipperDoublesInTheExecutePhase(t *testing.T) {
	encounter := &proto.Encounter{Duration: 600, Targets: []*proto.Target{bareTarget}, ExecuteProportion_20: 1, ExecuteProportion_25: 1, ExecuteProportion_35: 1}
	sim, character := unbuffedCharacterIn(t, proto.Class_ClassWarrior, proto.ItemSlot_ItemSlotMainHand, 2164, encounter)
	// The phases open on the first step past the pull.
	for i := 0; i < 5 && !sim.IsExecutePhase25(); i++ {
		sim.Step()
	}
	if !sim.IsExecutePhase25() {
		t.Fatal("the encounter did not open in its execute phase")
	}
	lo, hi, hits := hitRange(sim, character.GetSpell(core.ActionID{SpellID: 18107}), 2000)
	if hits < 100 || lo < 2*92.36 || hi > 2*117.64+0.5 {
		t.Errorf("execute-phase hits rolled %.1f to %.1f, want double 92.4 to 117.6", lo, hi)
	}
}

// Spire of Hakkar's Blood Siphon is 28 every 2 sec for 8 sec, Wall of the
// Dead's Bone Armor 150 armour for 20 sec.
func TestSpireOfHakkarAndWallOfTheDead(t *testing.T) {
	sim, character := unbuffedCharacter(t, proto.Class_ClassMage, proto.ItemSlot_ItemSlotMainHand, 10844)
	siphon := character.GetSpell(core.ActionID{SpellID: 1299943})
	if siphon == nil {
		t.Fatal("no Blood Siphon")
	}
	target := sim.Encounter.TargetUnits[0]
	siphon.Cast(sim, target)
	if dot := siphon.Dot(target); dot.SnapshotBaseDamage != 28 || dot.NumberOfTicks != 4 || dot.TickLength != 2*time.Second {
		t.Errorf("Blood Siphon is %v x %d every %v, want 28 x 4 every 2s", dot.SnapshotBaseDamage, dot.NumberOfTicks, dot.TickLength)
	}

	_, tank := unbuffedCharacter(t, proto.Class_ClassWarrior, proto.ItemSlot_ItemSlotOffHand, 1979)
	bone := tank.GetAuraByID(core.ActionID{SpellID: 18828})
	if bone == nil || bone.Duration != 20*time.Second {
		t.Fatal("no 20 sec Bone Armor aura")
	}
}

// Gloves of Holy Might's smite (1325229) is SpellCategories DefenseType 3,
// ranged, in the 70707 client, as Hammer of Wrath (24239) is: it rolls the
// ranged table, melee and ranged crit and the physical crit multiplier, not
// the spell table.
func TestGlovesOfHolyMightRollsTheRangedTable(t *testing.T) {
	sim, character := unbuffedCharacter(t, proto.Class_ClassHunter, proto.ItemSlot_ItemSlotHands, 867)
	spell := character.GetSpell(core.ActionID{SpellID: 1325229})
	if spell == nil {
		t.Fatal("no Gloves of Holy Might proc spell")
	}
	if spell.DefenseType != core.DefenseTypeRanged {
		t.Errorf("defense type %v, want ranged (client DefenseType 3)", spell.DefenseType)
	}
	target := sim.Encounter.TargetUnits[0]
	maxCrit := 0.0
	for i := 0; i < 4000; i++ {
		before := spell.SpellMetrics[target.UnitIndex]
		spell.Cast(sim, target)
		after := spell.SpellMetrics[target.UnitIndex]
		if after.Crits == before.Crits+1 {
			maxCrit = max(maxCrit, after.TotalDamage-before.TotalDamage)
		}
	}
	// The roll tops out at 123.2; a ranged crit doubles it, a spell crit
	// adds half.
	if maxCrit < 112*1.1*1.8 {
		t.Errorf("largest crit %.1f, want a physical double of the 101-123 roll", maxCrit)
	}
}
