package core

// Poised is Forever's Energy opener discount (new in 1.60.1.70753), trained
// by rogues (passive 1328825, level 18) and druids (passive 1328829, level
// 32): "While at maximum Energy, and out of combat, reduce the Energy cost
// of your next ability by 10." The passive is a dummy (aura 4); what it
// gives is a buff (rogue 1328826, druid 1328827) with no duration, one
// charge (SpellAuraOptions ProcCharges 1) and one effect: aura 107
// (ADD_FLAT_MODIFIER), SpellMod 14 (cost), -10, on the class's
// EffectSpellClassMask.
//
// An encounter starts out of combat at full Energy (energyBar.reset), and
// the buff has no duration, so every fight opens with it up and the first
// ability in its mask spends it. Combat never ends inside a fight, so it
// does not come back.

// PoisedConfig is one class's Poised buff, from the client rows.
type PoisedConfig struct {
	BuffSpellID   int32  // the buff the passive applies
	CostReduction int64  // the buff's EffectBasePointsF (negative: Energy off the cost)
	ClassMask     uint64 // the engine spells the buff's EffectSpellClassMask reaches
}

// RegisterPoised registers the buff and raises it at the start of every
// fight. The caller decides whether the character has learned Poised (its
// level against the passive's).
func (character *Character) RegisterPoised(config PoisedConfig) *Aura {
	if config.ClassMask == 0 {
		panic("Poised reaches no spell the engine models")
	}
	costMod := character.AddDynamicMod(SpellModConfig{
		Kind:      SpellMod_PowerCost_Flat,
		ClassMask: config.ClassMask,
		IntValue:  config.CostReduction,
	})
	return MakePermanent(character.RegisterAura(Aura{
		Label:    "Poised",
		ActionID: ActionID{SpellID: config.BuffSpellID},
		OnGain: func(_ *Aura, _ *Simulation) {
			costMod.Activate()
		},
		OnExpire: func(_ *Aura, _ *Simulation) {
			costMod.Deactivate()
		},
		OnCastComplete: func(aura *Aura, sim *Simulation, spell *Spell) {
			if spell.Matches(config.ClassMask) {
				aura.Deactivate(sim)
			}
		},
	}))
}
