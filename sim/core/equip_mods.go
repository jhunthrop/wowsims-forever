package core

import (
	"fmt"
	"time"
)

// Equip-effect spell modifiers.
//
// A relic (libram, idol, totem) or similar item states its value as an
// equip spell whose effect is a spell-family modifier: aura 107
// (ADD_FLAT_MODIFIER) or 108 (ADD_PCT_MODIFIER), a property it changes
// (SpellModOp, the effect's EffectMiscValue_0), an amount
// (EffectBasePointsF) and the family of spells it reaches
// (EffectSpellClassMask_0..3). EquipSpellMod carries those four numbers
// straight from the client rows; a class package supplies the table that
// says which of its engine spells each client family bit is, and
// NewEquipModItemEffect turns the pair into core.SpellModConfig static mods
// on the wearer.
//
// The generated tables live in each class package (relic_mods_auto_gen.go,
// from tools/relicmods/gen.py); this file is the one place that decides what
// a (aura, property) pair means.

// ClientClassMask is the client's 128-bit spell family mask, one 32-bit word
// per EffectSpellClassMask_N column.
type ClientClassMask [4]uint32

// Intersects reports whether the two masks share a family bit.
func (mask ClientClassMask) Intersects(other ClientClassMask) bool {
	for word := range mask {
		if mask[word]&other[word] != 0 {
			return true
		}
	}
	return false
}

// The client's modifier auras (SpellEffect.EffectAura).
const (
	ClientAuraAddFlatModifier int32 = 107
	ClientAuraAddPctModifier  int32 = 108
)

// The client's SpellModOp values (SpellEffect.EffectMiscValue_0) this
// package knows how to apply. Anything else is refused rather than guessed.
const (
	ClientModOpDamage     int32 = 0
	ClientModOpDuration   int32 = 1
	ClientModOpCritChance int32 = 7
	ClientModOpCastTime   int32 = 10
	ClientModOpCooldown   int32 = 11
	ClientModOpCost       int32 = 14
)

// EquipSpellMod is one modifier effect of an equip spell, as the client
// states it.
type EquipSpellMod struct {
	ClientSpellID int32 // the equip spell that carries the effect
	Aura          int32 // ClientAuraAddFlatModifier or ClientAuraAddPctModifier
	Op            int32 // ClientModOp*
	Amount        int32 // EffectBasePointsF: percent points, milliseconds or a flat count, per Op
	Families      ClientClassMask
}

// ClassMaskEntry names which engine spells one client family is.
type ClassMaskEntry struct {
	Client ClientClassMask
	Engine uint64
}

// ClassMaskTable maps a class's client spell families to its engine
// ClassSpellMask bits.
type ClassMaskTable []ClassMaskEntry

// Resolve returns the engine mask of every spell the client families reach.
// Zero means none of them is a spell the engine models.
func (table ClassMaskTable) Resolve(families ClientClassMask) uint64 {
	var engine uint64
	for _, entry := range table {
		if entry.Client.Intersects(families) {
			engine |= entry.Engine
		}
	}
	return engine
}

// SpellModConfig translates the modifier into the engine's static mod
// config for the given engine spell mask. It returns an error for an
// (aura, op) pair no engine mod expresses.
func (mod EquipSpellMod) SpellModConfig(engineMask uint64) (SpellModConfig, error) {
	flat := mod.Aura == ClientAuraAddFlatModifier
	pct := mod.Aura == ClientAuraAddPctModifier
	config := SpellModConfig{ClassMask: engineMask}

	switch {
	case mod.Op == ClientModOpDamage && flat:
		config.Kind, config.IntValue = SpellMod_DamageDone_Flat, int64(mod.Amount)
	case mod.Op == ClientModOpDamage && pct:
		config.Kind, config.FloatValue = SpellMod_DamageDone_Pct, 1+float64(mod.Amount)/100
	case mod.Op == ClientModOpDuration && flat:
		config.Kind, config.TimeValue = SpellMod_DotDuration_Flat, time.Duration(mod.Amount)*time.Millisecond
	case mod.Op == ClientModOpCritChance && flat:
		config.Kind, config.FloatValue = SpellMod_BonusCrit_Flat, float64(mod.Amount)*CritRatingPerCritChance
	case mod.Op == ClientModOpCastTime && flat:
		config.Kind, config.TimeValue = SpellMod_CastTime_Flat, time.Duration(mod.Amount)*time.Millisecond
	case mod.Op == ClientModOpCooldown && flat:
		config.Kind, config.TimeValue = SpellMod_Cooldown_Flat, time.Duration(mod.Amount)*time.Millisecond
	case mod.Op == ClientModOpCost && flat:
		config.Kind, config.IntValue = SpellMod_PowerCost_Flat, int64(mod.Amount)
	case mod.Op == ClientModOpCost && pct:
		config.Kind, config.IntValue = SpellMod_PowerCost_Pct, int64(mod.Amount)
	default:
		return SpellModConfig{}, fmt.Errorf("equip spell %d: aura %d with spell mod op %d is not modelled",
			mod.ClientSpellID, mod.Aura, mod.Op)
	}
	return config, nil
}

// NewEquipModItemEffect registers an item whose whole effect is the given
// modifiers. It resolves every modifier against the class's table now, at
// registration, and panics when a modifier names a spell family the engine
// does not model or an (aura, op) pair it cannot express: an item must not
// count as modelled while a part of it does nothing.
func NewEquipModItemEffect(itemID int32, mods []EquipSpellMod, table ClassMaskTable) {
	configs := make([]SpellModConfig, 0, len(mods))
	for _, mod := range mods {
		engineMask := table.Resolve(mod.Families)
		if engineMask == 0 {
			panic(fmt.Sprintf("item %d: equip spell %d reaches no spell the engine models", itemID, mod.ClientSpellID))
		}
		config, err := mod.SpellModConfig(engineMask)
		if err != nil {
			panic(fmt.Sprintf("item %d: %v", itemID, err))
		}
		configs = append(configs, config)
	}
	NewItemEffect(itemID, func(agent Agent) {
		character := agent.GetCharacter()
		for _, config := range configs {
			character.AddStaticMod(config)
		}
	})
}
