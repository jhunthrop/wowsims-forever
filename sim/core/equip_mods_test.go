package core

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestClassMaskTableResolveUnionsEveryFamilyTheModifierReaches(t *testing.T) {
	table := ClassMaskTable{
		{Client: ClientClassMask{1 << 0}, Engine: 1 << 0},
		{Client: ClientClassMask{0, 1 << 3}, Engine: 1 << 1},
		{Client: ClientClassMask{0, 0, 1 << 5}, Engine: 1 << 2},
	}
	if got := table.Resolve(ClientClassMask{1<<0 | 1<<7, 0, 1 << 5}); got != 1<<0|1<<2 {
		t.Errorf("Resolve = %b, want the first and third spells (101)", got)
	}
	if got := table.Resolve(ClientClassMask{0, 1 << 4}); got != 0 {
		t.Errorf("a family no entry names resolved to %b, want 0", got)
	}
}

func TestEquipSpellModTranslatesEachClientProperty(t *testing.T) {
	families := ClientClassMask{1}
	cases := []struct {
		name string
		mod  EquipSpellMod
		want SpellModConfig
	}{
		{"flat damage is additive percent points", EquipSpellMod{Aura: 107, Op: ClientModOpDamage, Amount: 30, Families: families},
			SpellModConfig{Kind: SpellMod_DamageDone_Flat, ClassMask: 8, IntValue: 30}},
		{"percent damage multiplies", EquipSpellMod{Aura: 108, Op: ClientModOpDamage, Amount: 4, Families: families},
			SpellModConfig{Kind: SpellMod_DamageDone_Pct, ClassMask: 8, FloatValue: 1.04}},
		{"flat duration lengthens dots", EquipSpellMod{Aura: 107, Op: ClientModOpDuration, Amount: 3000, Families: families},
			SpellModConfig{Kind: SpellMod_DotDuration_Flat, ClassMask: 8, TimeValue: 3 * time.Second}},
		{"flat crit chance is percent points", EquipSpellMod{Aura: 107, Op: ClientModOpCritChance, Amount: 6, Families: families},
			SpellModConfig{Kind: SpellMod_BonusCrit_Flat, ClassMask: 8, FloatValue: 6 * CritRatingPerCritChance}},
		{"flat cast time is milliseconds", EquipSpellMod{Aura: 107, Op: ClientModOpCastTime, Amount: -150, Families: families},
			SpellModConfig{Kind: SpellMod_CastTime_Flat, ClassMask: 8, TimeValue: -150 * time.Millisecond}},
		{"flat cooldown is milliseconds", EquipSpellMod{Aura: 107, Op: ClientModOpCooldown, Amount: -3000, Families: families},
			SpellModConfig{Kind: SpellMod_Cooldown_Flat, ClassMask: 8, TimeValue: -3 * time.Second}},
		{"flat cost is mana", EquipSpellMod{Aura: 107, Op: ClientModOpCost, Amount: -25, Families: families},
			SpellModConfig{Kind: SpellMod_PowerCost_Flat, ClassMask: 8, IntValue: -25}},
		{"percent cost is percent", EquipSpellMod{Aura: 108, Op: ClientModOpCost, Amount: -5, Families: families},
			SpellModConfig{Kind: SpellMod_PowerCost_Pct, ClassMask: 8, IntValue: -5}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.mod.SpellModConfig(8)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("config = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestEquipSpellModRefusesWhatTheEngineCannotExpress(t *testing.T) {
	for _, mod := range []EquipSpellMod{
		{ClientSpellID: 28852, Aura: 107, Op: 3, Amount: 48},              // effect 1 points: hand-written per item
		{ClientSpellID: 1, Aura: 108, Op: ClientModOpCooldown, Amount: 5}, // no percent cooldown mod is table-driven
		{ClientSpellID: 2, Aura: 112, Op: ClientModOpDamage, Amount: 33},  // class script, not a modifier
	} {
		if _, err := mod.SpellModConfig(1); err == nil || !strings.Contains(err.Error(), "not modelled") {
			t.Errorf("%+v: err = %v, want a not-modelled error", mod, err)
		}
	}
}

func TestNewEquipModItemEffectPanicsForAFamilyTheEngineDoesNotModel(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("registering a modifier that reaches no engine spell did not panic")
		}
	}()
	NewEquipModItemEffect(1, []EquipSpellMod{{Aura: 107, Op: ClientModOpDamage, Amount: 1, Families: ClientClassMask{1 << 4}}}, ClassMaskTable{})
}

func TestDotDurationModAddsWholeTicksAndRemovesThem(t *testing.T) {
	sim := SetupFakeSim()
	fa := sim.Raid.Parties[0].Players[0].(*FakeAgent)
	if fa.Dot.NumberOfTicks != 6 {
		t.Fatalf("fake dot has %d ticks, want 6", fa.Dot.NumberOfTicks)
	}

	mod := fa.AddDynamicMod(SpellModConfig{Kind: SpellMod_DotDuration_Flat, TimeValue: 3 * time.Second})
	mod.Activate()
	if fa.Dot.NumberOfTicks != 7 || fa.Dot.ModNumberOfTicks != 1 {
		t.Errorf("after +3s: ticks %d, mod ticks %d, want 7 and 1", fa.Dot.NumberOfTicks, fa.Dot.ModNumberOfTicks)
	}
	if fa.Dot.Aura.Duration != 7*fa.Dot.TickPeriod() {
		t.Errorf("aura duration %v is not 7 ticks of %v", fa.Dot.Aura.Duration, fa.Dot.TickPeriod())
	}
	mod.Deactivate()
	if fa.Dot.NumberOfTicks != 6 || fa.Dot.ModNumberOfTicks != 0 {
		t.Errorf("after removal: ticks %d, mod ticks %d, want 6 and 0", fa.Dot.NumberOfTicks, fa.Dot.ModNumberOfTicks)
	}
}

func TestDotDurationModPanicsOnAPartialTick(t *testing.T) {
	sim := SetupFakeSim()
	fa := sim.Raid.Parties[0].Players[0].(*FakeAgent)
	defer func() {
		if recover() == nil {
			t.Fatal("+2s on a 3s dot did not panic")
		}
	}()
	fa.AddStaticMod(SpellModConfig{Kind: SpellMod_DotDuration_Flat, TimeValue: 2 * time.Second})
}

func TestChangedSpellsNamesOnlyTheSpellsAModMoved(t *testing.T) {
	unit := &Unit{Type: PlayerUnit}
	modTestSpell(unit, testMaskBloodthirst, 0)
	modTestSpell(unit, testMaskWhirlwind, 0)
	before := SpellFingerprints(unit)

	unit.AddStaticMod(SpellModConfig{Kind: SpellMod_DamageDone_Flat, ClassMask: testMaskBloodthirst, IntValue: 10})
	changed := ChangedSpells(before, SpellFingerprints(unit))
	if len(changed) != 1 || !strings.HasSuffix(changed[0].Key, "#0") {
		t.Errorf("changed = %v, want exactly the first (Bloodthirst-masked) spell", changed)
	}
}
