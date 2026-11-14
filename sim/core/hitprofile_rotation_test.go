package core

import (
	"slices"
	"testing"
)

func rotationOf(actions ...APLActionImpl) *APLRotation {
	rot := &APLRotation{}
	for _, impl := range actions {
		rot.priorityList = append(rot.priorityList, &APLAction{impl: impl})
	}
	return rot
}

// A ranged hunter's spellbook holds traps and Volley, which carry spell
// damage, but the Beast Mastery and Marksmanship rotations never cast them:
// their spell hit is not a stat the character needs.
func TestSpellHitProfileReadsOnlyTheSpellsTheRotationCasts(t *testing.T) {
	hunter, boss := spellHitTestUnits(3)
	trap := registerDamageSpell(hunter, SpellSchoolFire, 0)
	shot := hunter.RegisterSpell(SpellConfig{
		ActionID: ActionID{SpellID: 2643}, SpellSchool: SpellSchoolPhysical, ProcMask: ProcMaskRangedSpecial,
		Flags: SpellFlagAPL, DamageMultiplier: 1, ThreatMultiplier: 1, ApplyEffects: func(*Simulation, *Unit, *Spell) {},
	})

	cast, ok := rotationCastSpells(rotationOf(&APLActionCastSpell{spell: shot}))
	if !ok || !slices.Equal(cast, []*Spell{shot}) {
		t.Fatalf("rotation casts = %v (ok %v), want the shot alone", cast, ok)
	}
	if _, spell := computeSpellHitProfile(hunter, boss, cast); spell {
		t.Fatal("a rotation that casts no damaging spell has no spell profile")
	}

	withTrap, _ := rotationCastSpells(rotationOf(&APLActionCastSpell{spell: shot}, &APLActionCastSpell{spell: trap}))
	if p, spell := computeSpellHitProfile(hunter, boss, withTrap); !spell || p.Hit != 3 {
		t.Fatalf("profile = %+v (spell %v), want the trap's hit 3", p, spell)
	}
}

func TestRotationCastSpellsReadsEveryCastingAction(t *testing.T) {
	unit, _ := spellHitTestUnits(0)
	cast := registerDamageSpell(unit, SpellSchoolFire, 0)
	channel := registerDamageSpell(unit, SpellSchoolShadow, 0)
	dot := registerDamageSpell(unit, SpellSchoolNature, 0)
	inner := registerDamageSpell(unit, SpellSchoolFrost, 0)
	rot := rotationOf(
		&APLActionCastSpell{spell: cast},
		&APLActionChannelSpell{spell: channel},
		&APLActionMultidot{spell: dot},
		&APLActionSequence{subactions: []*APLAction{{impl: &APLActionCastSpell{spell: inner}}}},
		&APLActionWait{},
	)
	got, ok := rotationCastSpells(rot)
	if !ok {
		t.Fatal("a rotation of plain actions is readable")
	}
	for _, want := range []*Spell{cast, channel, dot, inner} {
		if !slices.Contains(got, want) {
			t.Errorf("rotation casts %v, missing %v", got, want.ActionID)
		}
	}
}

func TestRotationCastSpellsCountsThePrepull(t *testing.T) {
	unit, _ := spellHitTestUnits(0)
	opener := registerDamageSpell(unit, SpellSchoolFire, 0)
	rot := &APLRotation{prepullActions: []*APLAction{{impl: &APLActionCastSpell{spell: opener}}}}
	if got, ok := rotationCastSpells(rot); !ok || !slices.Equal(got, []*Spell{opener}) {
		t.Fatalf("rotation casts = %v (ok %v), want the prepull opener", got, ok)
	}
}

// A custom rotation, or an action a spec defines itself, casts through code
// the profile cannot read, so the caller keeps the whole spellbook.
func TestRotationCastSpellsIsUnreadableThroughOpaqueActions(t *testing.T) {
	if _, ok := rotationCastSpells(rotationOf(&APLActionCustomRotation{})); ok {
		t.Error("a custom rotation is not readable")
	}
	if _, ok := rotationCastSpells(rotationOf(opaqueSpecAction{})); ok {
		t.Error("a spec's own action is not readable")
	}
}

type opaqueSpecAction struct{ defaultAPLActionImpl }

func (opaqueSpecAction) IsReady(*Simulation) bool { return false }
func (opaqueSpecAction) Execute(*Simulation)      {}
func (opaqueSpecAction) String() string           { return "opaque" }
