package clientsetbonus

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// modifierGlobalCooldown is the client's flat-modifier property for the
// global cooldown, which core.EquipSpellMod has no op for.
const modifierGlobalCooldown int32 = 21

// pvpUtilityReasons says, by bonus spell, why the sim leaves a PvP rank
// set's utility bonus unmodelled. Every spell here changes an ability the
// sim never casts for damage, healing, threat or mana.
var pvpUtilityReasons = map[int32]string{
	22738:  "modifies Intercept's cooldown, a gap closer the sim never casts",
	23025:  "modifies Blink's cooldown, an escape the sim never casts",
	23044:  "modifies Psychic Scream's duration, crowd control the sim never casts",
	23048:  "modifies Gouge's cooldown, crowd control the sim never casts",
	23158:  "modifies Concussive Shot's cooldown, a snare the sim never casts",
	23218:  "movement speed in Bear, Cat or Travel Form, which the sim does not model",
	23302:  "modifies Hammer of Justice's cooldown, crowd control the sim never casts",
	459584: "modifies Wing Clip's duration, a snare the sim never casts",
}

// RegisterPvPSet registers a PvP rank set from the client's row. Flat stat
// bonuses are applied from their rows; the bonuses in effects are modelled
// by the caller; every other bonus must be a utility ability in
// pvpUtilityReasons and is declared NoSim with that reason.
func RegisterPvPSet(id int32, effects map[int32]core.ApplyEffect) *core.ItemSet {
	row, ok := core.ClientSetRow(id)
	if !ok {
		panic(fmt.Sprintf("clientsetbonus: no client item set %d", id))
	}
	noSim := map[int32]string{}
	for _, bonus := range row.Bonuses {
		if _, modelled := effects[bonus.SpellID]; modelled {
			continue
		}
		if _, flat := core.DecodeClientFlatBonus(core.MustClientSpellRow(bonus.SpellID)); flat {
			continue
		}
		reason, known := pvpUtilityReasons[bonus.SpellID]
		if !known {
			panic(fmt.Sprintf("clientsetbonus: %s (%d) bonus spell %d is neither flat, modelled nor a known PvP utility",
				row.Name, id, bonus.SpellID))
		}
		noSim[bonus.SpellID] = reason
	}
	return core.NewClientItemSet(core.ClientSetModel{ID: id, Effects: effects, NoSim: noSim})
}

// RegisterPvPSets registers each set with no modelled bonus.
func RegisterPvPSets(ids ...int32) {
	for _, id := range ids {
		RegisterPvPSet(id, nil)
	}
}

// AddStaticMods adds, to the unit, the spell mods of a bonus spell made of
// flat-modifier effects (crit chance, cast time, global cooldown or any op
// core expresses). The spells each effect reaches are its client family
// mask resolved through the class's table. Panics on any other effect, or
// on a family the engine does not model.
func AddStaticMods(unit *core.Unit, spellID int32, table core.ClassMaskTable) {
	spell := core.MustClientSpellRow(spellID)
	for _, effect := range spell.Effects {
		if effect.Aura != core.ClientAuraAddFlatModifier {
			panic(fmt.Sprintf("clientsetbonus: spell %d (%s) has aura %d, want the flat modifier %d",
				spellID, spell.Name, effect.Aura, core.ClientAuraAddFlatModifier))
		}
		engineMask := table.Resolve(core.ClientClassMask(effect.ClassMask))
		if engineMask == 0 {
			panic(fmt.Sprintf("clientsetbonus: spell %d (%s) reaches no spell the engine models", spellID, spell.Name))
		}
		unit.AddStaticMod(modConfig(spellID, effect, engineMask))
	}
}

func modConfig(spellID int32, effect core.ClientEffect, engineMask uint64) core.SpellModConfig {
	if effect.Misc0 == modifierGlobalCooldown {
		return core.SpellModConfig{
			Kind:      core.SpellMod_GlobalCooldown_Flat,
			ClassMask: engineMask,
			TimeValue: time.Duration(effect.Points) * time.Millisecond,
		}
	}
	config, err := core.EquipSpellMod{
		ClientSpellID: spellID,
		Aura:          effect.Aura,
		Op:            effect.Misc0,
		Amount:        int32(effect.Points),
	}.SpellModConfig(engineMask)
	if err != nil {
		panic("clientsetbonus: " + err.Error())
	}
	return config
}
