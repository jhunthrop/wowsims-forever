package core

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
)

const (
	testMaskBloodthirst uint64 = 1 << 0
	testMaskWhirlwind   uint64 = 1 << 1
)

// modTestSpell registers a spell on a bare unit so a mod has something to
// bind to, without standing up a whole character. A zero cd omits Cast.CD
// entirely: RegisterSpell panics on a CD timer with no duration, and the
// damage/matching tests that pass cd=0 don't exercise the cooldown at all.
func modTestSpell(unit *Unit, mask uint64, cd time.Duration) *Spell {
	config := SpellConfig{
		ActionID:         ActionID{SpellID: 23894},
		ClassSpellMask:   mask,
		SpellSchool:      SpellSchoolPhysical,
		DefenseType:      DefenseTypeMelee,
		ProcMask:         ProcMaskMeleeMHSpecial,
		Flags:            SpellFlagMeleeMetrics | SpellFlagAPL,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		Cast: CastConfig{
			DefaultCast: Cast{GCD: GCDDefault},
		},
		ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {},
	}
	if cd > 0 {
		config.Cast.CD = Cooldown{Timer: unit.NewTimer(), Duration: cd}
	}
	return unit.RegisterSpell(config)
}

func TestSpellMatchesClassMask(t *testing.T) {
	unit := &Unit{Type: PlayerUnit}
	spell := modTestSpell(unit, testMaskBloodthirst, time.Second*6)
	if !spell.Matches(testMaskBloodthirst) {
		t.Error("spell does not match its own mask")
	}
	if spell.Matches(testMaskWhirlwind) {
		t.Error("spell matches a mask it does not carry")
	}
	if spell.Matches(0) {
		t.Error("an empty mask must match nothing")
	}
}

// The five damage helpers are re-expressed in classic's exported float
// multiplier fields rather than SoD's private percent accumulators. A
// percent of -50 must halve; +20 must add a fifth.
func TestApplyDamageBonusHelpers(t *testing.T) {
	unit := &Unit{Type: PlayerUnit}

	spell := modTestSpell(unit, testMaskBloodthirst, 0)
	spell.ApplyAdditiveDamageBonus(20)
	if got, want := spell.DamageMultiplierAdditive, 1.2; !closeEnough(got, want) {
		t.Errorf("after +20%%, DamageMultiplierAdditive = %v, want %v", got, want)
	}
	spell.ApplyAdditiveDamageBonus(-20)
	if got, want := spell.DamageMultiplierAdditive, 1.0; !closeEnough(got, want) {
		t.Errorf("after +20%% then -20%%, DamageMultiplierAdditive = %v, want %v", got, want)
	}

	spell.ApplyMultiplicativeDamageBonus(1.5)
	if got, want := spell.DamageMultiplier, 1.5; !closeEnough(got, want) {
		t.Errorf("after x1.5, DamageMultiplier = %v, want %v", got, want)
	}

	spell.ApplyAdditiveBaseDamageBonus(10)
	if got, want := spell.BaseDamageMultiplierAdditive, 1.1; !closeEnough(got, want) {
		t.Errorf("BaseDamageMultiplierAdditive = %v, want %v", got, want)
	}
	spell.ApplyAdditiveImpactDamageBonus(10)
	if got, want := spell.ImpactDamageMultiplierAdditive, 1.1; !closeEnough(got, want) {
		t.Errorf("ImpactDamageMultiplierAdditive = %v, want %v", got, want)
	}
	spell.ApplyAdditivePeriodicDamageBonus(10)
	if got, want := spell.PeriodicDamageMultiplierAdditive, 1.1; !closeEnough(got, want) {
		t.Errorf("PeriodicDamageMultiplierAdditive = %v, want %v", got, want)
	}
}

func TestCooldownMods(t *testing.T) {
	cd := Cooldown{Duration: time.Second * 30}
	cd.ApplyFlatCooldownMod(-time.Second * 5)
	if cd.Duration != time.Second*25 {
		t.Errorf("after -5s, Duration = %v, want 25s", cd.Duration)
	}
	cd.ApplyFlatPercentCooldownMod(-50)
	if cd.Duration != time.Second*12500/1000 {
		t.Errorf("after -50%%, Duration = %v, want 12.5s", cd.Duration)
	}
	// A cooldown never goes negative, however many mods pile on.
	cd.ApplyFlatCooldownMod(-time.Hour)
	if cd.Duration != 0 {
		t.Errorf("Duration = %v, want it clamped to 0", cd.Duration)
	}
}

// A static mod applies at registration and to every spell registered
// afterwards, which is what makes talents declarative.
func TestStaticModAppliesToMatchingSpells(t *testing.T) {
	unit := &Unit{Type: PlayerUnit}

	before := modTestSpell(unit, testMaskBloodthirst, 0)
	unit.AddStaticMod(SpellModConfig{
		Kind:      SpellMod_DamageDone_Flat,
		ClassMask: testMaskBloodthirst,
		IntValue:  30,
	})
	after := modTestSpell(unit, testMaskBloodthirst, 0)
	other := modTestSpell(unit, testMaskWhirlwind, 0)

	if got, want := before.DamageMultiplierAdditive, 1.3; !closeEnough(got, want) {
		t.Errorf("spell registered before the mod: %v, want %v", got, want)
	}
	if got, want := after.DamageMultiplierAdditive, 1.3; !closeEnough(got, want) {
		t.Errorf("spell registered after the mod: %v, want %v", got, want)
	}
	if got, want := other.DamageMultiplierAdditive, 1.0; !closeEnough(got, want) {
		t.Errorf("a non-matching spell was modified: %v, want %v", got, want)
	}
}

// A dynamic mod is what a proc or a temporary buff uses: it toggles.
func TestDynamicModActivatesAndDeactivates(t *testing.T) {
	unit := &Unit{Type: PlayerUnit}
	spell := modTestSpell(unit, testMaskBloodthirst, time.Second*6)

	mod := unit.AddDynamicMod(SpellModConfig{
		Kind:      SpellMod_Cooldown_Flat,
		ClassMask: testMaskBloodthirst,
		TimeValue: -time.Second * 2,
	})
	if spell.CD.Duration != time.Second*6 {
		t.Fatalf("a dynamic mod applied before Activate: %v", spell.CD.Duration)
	}
	mod.Activate()
	if spell.CD.Duration != time.Second*4 {
		t.Errorf("after Activate, CD = %v, want 4s", spell.CD.Duration)
	}
	mod.Deactivate()
	if spell.CD.Duration != time.Second*6 {
		t.Errorf("after Deactivate, CD = %v, want 6s", spell.CD.Duration)
	}
}

// A spell can opt out entirely, which the engine needs for auto attacks
// and for spells a talent must not touch.
func TestSpellFlagNoSpellModsOptsOut(t *testing.T) {
	unit := &Unit{Type: PlayerUnit}
	spell := unit.RegisterSpell(SpellConfig{
		ActionID:         ActionID{SpellID: 1},
		ClassSpellMask:   testMaskBloodthirst,
		SpellSchool:      SpellSchoolPhysical,
		ProcMask:         ProcMaskMeleeMHSpecial,
		Flags:            SpellFlagNoSpellMods,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ApplyEffects:     func(sim *Simulation, target *Unit, spell *Spell) {},
	})
	unit.AddStaticMod(SpellModConfig{Kind: SpellMod_DamageDone_Flat, ClassMask: testMaskBloodthirst, IntValue: 50})
	if got, want := spell.DamageMultiplierAdditive, 1.0; !closeEnough(got, want) {
		t.Errorf("a SpellFlagNoSpellMods spell was modified: %v, want %v", got, want)
	}
}

func TestUnknownModKindPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("an unimplemented mod kind did not panic")
		}
	}()
	unit := &Unit{Type: PlayerUnit}
	// SpellModType is uint32 (ported unchanged from SoD); 1<<31 is its top
	// bit and is unimplemented, same as the brief's 1<<62 intended.
	unit.AddStaticMod(SpellModConfig{Kind: SpellModType(1 << 31), ClassMask: testMaskBloodthirst})
}

func closeEnough(a, b float64) bool {
	d := a - b
	return d < 1e-9 && d > -1e-9
}

var _ = proto.CastType_CastTypeUnknown

// RemoveSpellByClassMask deletes from the slice it walks. Walking it
// forwards, each removal shifted the tail down under the loop index and
// the next element was skipped, so two adjacent matching spells left the
// second attached and still modified. No engine code calls this today —
// it arrived with the upstream spell-mod system — which is why nothing
// caught it.
func TestRemoveSpellByClassMaskRemovesAdjacentMatches(t *testing.T) {
	unit := &Unit{Type: PlayerUnit}

	mod := unit.AddDynamicMod(SpellModConfig{
		Kind:      SpellMod_DamageDone_Flat,
		ClassMask: testMaskBloodthirst | testMaskWhirlwind,
		IntValue:  30,
	})
	mod.Activate()

	// Three spells, all matching, registered back to back: with the
	// forward loop the second survived the sweep.
	first := modTestSpell(unit, testMaskBloodthirst, 0)
	second := modTestSpell(unit, testMaskBloodthirst, 0)
	third := modTestSpell(unit, testMaskBloodthirst, 0)
	kept := modTestSpell(unit, testMaskWhirlwind, 0)

	if len(mod.AffectedSpells) != 4 {
		t.Fatalf("the mod bound to %d spells, want 4", len(mod.AffectedSpells))
	}

	mod.RemoveSpellByClassMask(testMaskBloodthirst)

	if len(mod.AffectedSpells) != 1 {
		t.Errorf("after removing the Bloodthirst-masked spells the mod still holds %d, want 1", len(mod.AffectedSpells))
	}
	for _, spell := range []*Spell{first, second, third} {
		if got, want := spell.DamageMultiplierAdditive, 1.0; !closeEnough(got, want) {
			t.Errorf("a removed spell still carries the mod: %v, want %v", got, want)
		}
	}
	if got, want := kept.DamageMultiplierAdditive, 1.3; !closeEnough(got, want) {
		t.Errorf("the non-matching spell lost the mod: %v, want %v", got, want)
	}
}
