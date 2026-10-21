package core

import (
	"time"

	"github.com/wowsims/classic/sim/core/proto"
)

// Flametongue Totem as the 1.60.1.70009 client states it (rows quoted in
// flametongue_totem_test.go): an aura on every party member, not a weapon
// enchant, that adds fire damage to each main-hand auto attack. The rank's
// amount is the proc spell's base points; the description's own range
// ("m1/77 to M1/25", the amount per second of weapon speed over 1.3 to 4.0
// seconds) makes a hit deal Amount x weapon speed / 100.

// FlametongueTotemRanks lists the ranks by the cast spell's id and learn
// level; Amount is the proc spell's base points.
var FlametongueTotemRanks = BuffRanks{
	{SpellID: 8227, Level: 28, Amount: 548},
	{SpellID: 8249, Level: 38, Amount: 781},
	{SpellID: 10526, Level: 48, Amount: 1061},
	{SpellID: 16387, Level: 58, Amount: 1363},
}

// FlametongueTotemProcSpellIDs are the proc spells (the ones that carry the
// amount), in rank order.
var FlametongueTotemProcSpellIDs = [...]int32{8253, 8248, 10523, 16389}

// FlametongueTotemSpeedDivisor turns base points into damage per second of
// weapon speed.
const FlametongueTotemSpeedDivisor = 100

// FlametongueTotemAuraLabel names the permanent raid-buff aura and the
// totem's own buff, so a shaman who drops the totem and a raid that brings
// one share a single aura.
const FlametongueTotemAuraLabel = "Flametongue Totem"

// meleeTotemCategory is the exclusive category the client's beta notes put
// the two weapon-hit totems in: "Windfury no longer stacks with ...
// Flametongue totem". The Windfury Totem outranks Flametongue Totem, the
// order the raid preset already chose.
const (
	meleeTotemCategory    = "Melee Weapon Totem"
	meleeTotemWindfury    = 2
	meleeTotemFlametongue = 1
)

// flametongueTotemDuration is the client's duration_ms (300000, spells 8227
// to 16387) on a standing totem.
const flametongueTotemDuration = 5 * time.Minute

// flametongueTotemRankIndex is the zero-based index of the highest rank a
// character of the level has learned, -1 before the first.
func flametongueTotemRankIndex(level int) int {
	rank, ok := FlametongueTotemRanks.Learned(level)
	if !ok {
		return -1
	}
	for i, candidate := range FlametongueTotemRanks {
		if candidate.SpellID == rank.SpellID {
			return i
		}
	}
	return -1
}

// FlametongueTotemHitDamage is the fire damage one main-hand hit adds at a
// character level for a weapon of the given speed; false when the level has
// not learned the totem.
func FlametongueTotemHitDamage(level int, weaponSpeed float64) (float64, bool) {
	index := flametongueTotemRankIndex(level)
	if index < 0 {
		return 0, false
	}
	return FlametongueTotemRanks[index].Amount * weaponSpeed / FlametongueTotemSpeedDivisor, true
}

// flametongueTotemReaches says whether the aura applies to the character:
// Flametongue Weapon on the main hand disables it (the weapon's text), and
// a druid in feral form has no main-hand weapon to hit with.
func flametongueTotemReaches(character *Character) bool {
	return weaponTotemReaches(character, proto.WeaponImbue_FlametongueWeapon)
}

// FlametongueTotemAura registers the character's Flametongue Totem aura
// and its proc spell, once, and returns the aura (inactive). The raid buff
// makes it permanent; a shaman's own cast activates it for the totem's
// life. The proc is skipped while the Windfury Totem effect holds the
// melee totem category.
func FlametongueTotemAura(character *Character) *Aura {
	index := flametongueTotemRankIndex(int(character.Level))
	if index < 0 {
		return nil
	}
	procSpell := character.GetOrRegisterSpell(SpellConfig{
		ActionID:    ActionID{SpellID: FlametongueTotemProcSpellIDs[index]},
		SpellSchool: SpellSchoolFire,
		DefenseType: DefenseTypeMagic,
		ProcMask:    ProcMaskSpellDamageProc,
		Flags:       SpellFlagNoOnCastComplete,

		RequiredLevel: int(FlametongueTotemRanks[index].Level),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		// The client's base points, before the weapon-speed factor the
		// hit applies: what the conformance report compares.
		ClientBaseDamage: [2]float64{FlametongueTotemRanks[index].Amount, FlametongueTotemRanks[index].Amount},

		ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
			if damage, ok := FlametongueTotemHitDamage(int(character.Level), character.AutoAttacks.MH().SwingSpeed); ok && damage > 0 {
				spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMagicHitAndCrit)
			}
		},
	})

	var effect *ExclusiveEffect
	aura := character.GetOrRegisterAura(Aura{
		Label:    FlametongueTotemAuraLabel,
		ActionID: ActionID{SpellID: FlametongueTotemRanks[index].SpellID},
		Duration: flametongueTotemDuration,
		OnSpellHitDealt: func(_ *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			if !result.Landed() || !spell.ProcMask.Matches(ProcMaskMeleeMHAuto) || spell.Flags.Matches(SpellFlagSuppressEquipProcs) {
				return
			}
			if effect != nil && effect.IsActive() {
				procSpell.Cast(sim, result.Target)
			}
		},
	})
	effect = aura.NewExclusiveEffect(meleeTotemCategory, false, ExclusiveEffect{Priority: meleeTotemFlametongue})
	return aura
}
