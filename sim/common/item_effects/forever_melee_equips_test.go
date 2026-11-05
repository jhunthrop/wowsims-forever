package item_effects_test

// Forever melee equip effects the engine did not implement (sim audit
// round 21, items lane), against the 70790 client rows and Wowhead
// Forever's tooltips (nether.wowhead.com/forever/tooltip/spell/<id>, read
// 2026-11-05).

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// hitWith drives trigger's hit callback with n landed hits of a spell
// carrying procMask, as the engine's proc tests do.
func hitWith(t *testing.T, sim *core.Simulation, character *core.Character, trigger string, procMask core.ProcMask, n int) {
	t.Helper()
	aura := character.GetAura(trigger)
	if aura == nil || aura.OnSpellHitDealt == nil {
		t.Fatalf("no %s trigger aura", trigger)
	}
	spell := &core.Spell{ProcMask: procMask}
	result := &core.SpellResult{Outcome: core.OutcomeHit, Target: sim.Encounter.TargetUnits[0], Damage: 1}
	for i := 0; i < n; i++ {
		aura.OnSpellHitDealt(aura, sim, spell, result)
	}
}

func TestForeverMeleeEquipsRollTheClientsDamage(t *testing.T) {
	mh, hands := proto.ItemSlot_ItemSlotMainHand, proto.ItemSlot_ItemSlotHands
	warrior := proto.Class_ClassWarrior
	for _, proc := range []clientProc{
		// 1298499: Effect 2, base 27.72, variance 0.4. "Melee attacks with
		// this weapon deal 22 to 33 Frost damage."
		{"Warblade of Caer Darrow", warrior, mh, 13982, 1298499, 27.71999931335, 0.40000000596},
		// 16614 Lightning Strike: Effect 2, base 3. "Adds 3 Lightning damage
		// to your melee attacks."
		{"Storm Gauntlets", warrior, hands, 12632, 16614, 3, 0},
	} {
		t.Run(proc.name, func(t *testing.T) {
			sim, character := unbuffedCharacter(t, proc.class, proc.slot, proc.itemID)
			spell := character.GetSpell(core.ActionID{SpellID: proc.spellID})
			if spell == nil {
				t.Fatalf("%s equipped, but its client spell %d is not registered", proc.name, proc.spellID)
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

// Both ride every landed melee hit: SpellAuraOptions 246336 (Warblade) and
// 128107 (Storm Gauntlets) are ProcChance 100 on ProcTypeMask 20, auto
// attacks and melee specials.
func TestForeverMeleeEquipsProcOnEveryLandedHit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		slot    proto.ItemSlot
		itemID  int32
		spellID int32
	}{
		{"Warblade of Caer Darrow", proto.ItemSlot_ItemSlotMainHand, 13982, 1298499},
		{"Storm Gauntlets", proto.ItemSlot_ItemSlotHands, 12632, 16614},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim, character := unbuffedCharacter(t, proto.Class_ClassWarrior, tc.slot, tc.itemID)
			spell := character.GetSpell(core.ActionID{SpellID: tc.spellID})
			if spell == nil {
				t.Fatalf("no %s spell %d", tc.name, tc.spellID)
			}
			hitWith(t, sim, character, tc.name, core.ProcMaskMeleeMHAuto, 200)
			hitWith(t, sim, character, tc.name, core.ProcMaskMeleeMHSpecial, 100)
			if got := spell.SpellMetrics[sim.Encounter.TargetUnits[0].UnitIndex].Casts; got != 300 {
				t.Errorf("%d procs on 200 landed swings and 100 landed specials, want 300", got)
			}
		})
	}
}

// 1298499's SpellCategories.DefenseType is 2 (melee) and its SpellMisc
// Attributes_3 carries ALWAYS_HIT (0x40000): it never misses, and crits on
// the melee table.
func TestWarbladeFrostNeverMissesAndCritsOnTheMeleeTable(t *testing.T) {
	sim, character := unbuffedCharacter(t, proto.Class_ClassWarrior, proto.ItemSlot_ItemSlotMainHand, 13982)
	spell := character.GetSpell(core.ActionID{SpellID: 1298499})
	if spell == nil {
		t.Fatal("no Warblade of Caer Darrow spell")
	}
	if spell.DefenseType != core.DefenseTypeMelee || spell.SpellSchool != core.SpellSchoolFrost {
		t.Fatalf("defense %v school %v, want the client's melee defense and Frost", spell.DefenseType, spell.SpellSchool)
	}
	character.AddStatDynamic(sim, stats.MeleeCrit, 40*core.CritRatingPerCritChance)
	target := sim.Encounter.TargetUnits[0]
	for i := 0; i < 2000; i++ {
		spell.Cast(sim, target)
	}
	metrics := spell.SpellMetrics[target.UnitIndex]
	if landed := metrics.Hits + metrics.Crits; landed != 2000 {
		t.Errorf("%d of 2000 casts landed, want every one", landed)
	}
	if rate := float64(metrics.Crits) / 2000; rate < 0.3 {
		t.Errorf("crit %.1f%% with 40%% melee crit, want the melee table", 100*rate)
	}
}

// Blade of Hanna (2801): ItemEffect 238936, 1325535 Improving Form, adds a
// stack of 1325509 on each landed auto attack (SpellAuraOptions 248750,
// ProcTypeMask 4); at 30 stacks (1325509 aura 542, misc 30) it casts
// 1325508 Perfected Form, 490 Physical (melee defense) and 20 to all stats
// (aura 29, misc -1) for 20 sec (duration 18), and clears the stacks
// (effect 164). "At 30 stacks perfect your form, inflicting 490 Physical
// damage to your current target and increasing all your stats by 20 for
// 20 sec."
func TestBladeOfHannaPerfectsItsFormEveryThirtyHits(t *testing.T) {
	sim, character := unbuffedCharacter(t, proto.Class_ClassWarrior, proto.ItemSlot_ItemSlotMainHand, 2801)
	stacks := character.GetAura("Improving Form")
	form := character.GetAura("Perfected Form")
	strike := character.GetSpell(core.ActionID{SpellID: 1325508})
	if stacks == nil || form == nil || strike == nil {
		t.Fatalf("Blade of Hanna registered stacks %v, form %v, strike %v", stacks != nil, form != nil, strike != nil)
	}
	if strike.DefenseType != core.DefenseTypeMelee || strike.SpellSchool != core.SpellSchoolPhysical {
		t.Fatalf("Perfected Form defense %v school %v, want melee Physical", strike.DefenseType, strike.SpellSchool)
	}

	strength := character.GetStat(stats.Strength)
	hitWith(t, sim, character, "Blade of Hanna", core.ProcMaskMeleeMHAuto, 29)
	if form.IsActive() || stacks.GetStacks() != 29 || strike.SpellMetrics[0].Casts != 0 {
		t.Fatalf("after 29 landed swings: form %v, %d stacks, %d strikes; want 29 stacks and nothing yet",
			form.IsActive(), stacks.GetStacks(), strike.SpellMetrics[0].Casts)
	}
	hitWith(t, sim, character, "Blade of Hanna", core.ProcMaskMeleeMHAuto, 1)
	if !form.IsActive() || form.Duration.Seconds() != 20 {
		t.Fatalf("after 30 landed swings: form active %v for %v, want 20 sec", form.IsActive(), form.Duration)
	}
	if got := character.GetStat(stats.Strength) - strength; got < 20 {
		t.Errorf("Perfected Form adds %.1f Strength, want at least the client's 20", got)
	}
	if stacks.GetStacks() != 0 {
		t.Errorf("%d stacks left after the form, want the client's reset", stacks.GetStacks())
	}
	if strike.SpellMetrics[0].Casts != 1 {
		t.Errorf("%d Perfected Form strikes, want 1", strike.SpellMetrics[0].Casts)
	}
	lo, hi, hits := hitRange(sim, strike, 500)
	if hits < 100 || lo != 490 || hi != 490 {
		t.Errorf("Perfected Form hits %.1f to %.1f (%d plain hits) on an unarmoured target, want the client's 490", lo, hi, hits)
	}
}

// Melee specials do not build Improving Form: 248750's ProcTypeMask is 4,
// auto attacks only.
func TestBladeOfHannaStacksOnlyOnAutoAttacks(t *testing.T) {
	sim, character := unbuffedCharacter(t, proto.Class_ClassWarrior, proto.ItemSlot_ItemSlotMainHand, 2801)
	hitWith(t, sim, character, "Blade of Hanna", core.ProcMaskMeleeMHSpecial, 10)
	if got := character.GetAura("Improving Form").GetStacks(); got != 0 {
		t.Errorf("%d stacks from 10 melee specials, want none", got)
	}
}
