package clientsetbonus

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// The thresholds of the Tier 1 utility (3-piece) and cooldown or duration
// (5-piece) bonuses.
const (
	ThreePieces int32 = 3
	FivePieces  int32 = 5
)

// SpellAt is the bonus spell a set grants at a piece count, read from the
// set's row, so a model names a bonus by its threshold and never retypes
// the id.
func SpellAt(setID, threshold int32) int32 {
	row, ok := core.ClientSetRow(setID)
	if !ok {
		panic(fmt.Sprintf("clientsetbonus: no client item set %d", setID))
	}
	for _, bonus := range row.Bonuses {
		if bonus.Threshold == threshold {
			return bonus.SpellID
		}
	}
	panic(fmt.Sprintf("clientsetbonus: set %d (%s) has no %d-piece bonus", setID, row.Name, threshold))
}

// TableMod is the spell mod for a bonus spell that is one add-modifier
// effect (cooldown, duration, cost, cast time, ...) on the spell families
// the row names. The class's table says which engine spells those families
// are; a family the table lacks stops the engine at init, so a bonus never
// counts as modelled while it reaches nothing.
func TableMod(spellID int32, table core.ClassMaskTable) core.SpellModConfig {
	effect := onlyEffect(spellID)
	mod := core.EquipSpellMod{
		ClientSpellID: spellID,
		Aura:          effect.Aura,
		Op:            effect.Misc0,
		Amount:        int32(effect.Points),
		Families:      effect.ClassMask,
	}
	engineMask := table.Resolve(mod.Families)
	if engineMask == 0 {
		panic(fmt.Sprintf("clientsetbonus: spell %d (%s) reaches no spell the engine models", spellID, core.MustClientSpellRow(spellID).Name))
	}
	config, err := mod.SpellModConfig(engineMask)
	if err != nil {
		panic(fmt.Sprintf("clientsetbonus: %v", err))
	}
	return config
}

// StaticMod applies the set's bonus at the threshold as a static spell mod
// built by TableMod.
func StaticMod(setID, threshold int32, table core.ClassMaskTable) core.ApplyEffect {
	config := TableMod(SpellAt(setID, threshold), table)
	return func(agent core.Agent) {
		agent.GetCharacter().AddStaticMod(config)
	}
}

// FivePieceMod is the Effects entry of a set whose 5-piece bonus is one
// spell mod (the Tier 1 cooldown reductions).
func FivePieceMod(setID int32, table core.ClassMaskTable) map[int32]core.ApplyEffect {
	return map[int32]core.ApplyEffect{
		SpellAt(setID, FivePieces): StaticMod(setID, FivePieces, table),
	}
}

// DummyDuration is a bonus spell's dummy-aura number read as milliseconds
// (a set bonus that changes how long a script-owned aura lasts).
func DummyDuration(spellID int32) time.Duration {
	effect := onlyEffect(spellID)
	if effect.Aura != auraDummy {
		panic(fmt.Sprintf("clientsetbonus: spell %d is aura %d, want the dummy aura %d", spellID, effect.Aura, auraDummy))
	}
	return time.Duration(effect.Points) * time.Millisecond
}

// NoSimThreePiece declares the 3-piece bonus of a set NoSim, with its reason:
// the Tier 1 sets' 3-piece bonuses are all utility abilities.
func NoSimThreePiece(setID int32, note string) map[int32]string {
	return map[int32]string{SpellAt(setID, ThreePieces): note}
}
