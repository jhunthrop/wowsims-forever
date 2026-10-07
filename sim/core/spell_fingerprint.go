package core

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
)

// SpellFingerprint is the modifier-visible state of one registered spell:
// every number a SpellMod changes. Tests snapshot a unit with and without an
// item, then ask which spells the item moved (ChangedSpells), so "the
// modifier shows on the right spell and on no other" is one assertion.
type SpellFingerprint struct {
	ClassSpellMask           uint64
	DamageMultiplier         float64
	DamageMultiplierAdditive float64
	CostMultiplier           int32
	CostFlatModifier         int32
	BonusCritRating          float64
	CastTime                 time.Duration
	CooldownDuration         time.Duration
	DotTicks                 int32
}

func fingerprint(spell *Spell) SpellFingerprint {
	fp := SpellFingerprint{
		ClassSpellMask:           spell.ClassSpellMask,
		DamageMultiplier:         spell.DamageMultiplier,
		DamageMultiplierAdditive: spell.DamageMultiplierAdditive,
		BonusCritRating:          spell.BonusCritRating,
		CastTime:                 spell.DefaultCast.CastTime,
		CooldownDuration:         spell.CD.Duration,
	}
	if spell.Cost != nil {
		fp.CostMultiplier = spell.Cost.Multiplier
		fp.CostFlatModifier = spell.Cost.FlatModifier
	}
	spell.eachDot(func(dot *Dot) { fp.DotTicks += dot.NumberOfTicks })
	return fp
}

// SpellFingerprints snapshots every spell the unit has registered, keyed by
// action id (suffixed #n when several spells share one, in registration
// order).
func SpellFingerprints(unit *Unit) map[string]SpellFingerprint {
	out := make(map[string]SpellFingerprint, len(unit.Spellbook))
	seen := map[string]int{}
	for _, spell := range unit.Spellbook {
		base := spell.ActionID.String()
		key := fmt.Sprintf("%s#%d", base, seen[base])
		seen[base]++
		out[key] = fingerprint(spell)
	}
	return out
}

// SpellChange is one spell a snapshot pair disagrees on. Before is the zero
// value for a spell the first snapshot lacks.
type SpellChange struct {
	Key           string
	Before, After SpellFingerprint
}

// ChangedSpells returns, sorted by key, every spell whose fingerprint differs
// between the two snapshots, including a spell present in only one of them.
func ChangedSpells(before, after map[string]SpellFingerprint) []SpellChange {
	var changed []SpellChange
	for key, fp := range after {
		if old, ok := before[key]; !ok || old != fp {
			changed = append(changed, SpellChange{Key: key, Before: before[key], After: fp})
		}
	}
	for key, fp := range before {
		if _, ok := after[key]; !ok {
			changed = append(changed, SpellChange{Key: key, Before: fp})
		}
	}
	slices.SortFunc(changed, func(a, b SpellChange) int { return strings.Compare(a.Key, b.Key) })
	return changed
}

// RelicEquipment is an equipment spec that wears only the given relic in the
// ranged slot (zero for none), for tests that compare a unit with and
// without it.
func RelicEquipment(relicID int32) *proto.EquipmentSpec {
	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotRanged] = &proto.ItemSpec{Id: relicID}
	return &proto.EquipmentSpec{Items: items}
}
