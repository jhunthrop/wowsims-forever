package core

import "slices"

// rotationCastSpells returns the spells the rotation can cast: those its
// cast, channel, multidot and multishield actions name, at any depth and in
// the prepull, plus the major cooldowns an autocast action fires. The hit
// profile reads only these, so a spell the character knows but never casts
// (a Beast Mastery hunter's traps and Volley) asks for no spell hit.
//
// ok is false when an action casts through code the profile cannot read (a
// custom rotation, or an action a spec defines itself); the caller then
// falls back to the whole spellbook.
func rotationCastSpells(rot *APLRotation) (spells []*Spell, ok bool) {
	add := func(spell *Spell) {
		if spell != nil && !slices.Contains(spells, spell) {
			spells = append(spells, spell)
		}
	}
	pending := slices.Concat(rot.prepullActions, rot.priorityList)
	for len(pending) > 0 {
		action := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if action == nil {
			continue
		}
		switch impl := action.impl.(type) {
		case *APLActionCastSpell:
			add(impl.spell)
		case *APLActionChannelSpell:
			add(impl.spell)
		case *APLActionMultidot:
			add(impl.spell)
		case *APLActionMultishield:
			add(impl.spell)
		case *APLActionAutocastOtherCooldowns:
			if impl.character != nil {
				for _, mcd := range impl.character.initialMajorCooldowns {
					add(mcd.Spell)
				}
			}
		case *APLActionSequence, *APLActionStrictSequence, *APLActionResetSequence,
			*APLActionWait, *APLActionWaitUntil, *APLActionSchedule,
			*APLActionChangeTarget, *APLActionActivateAura, *APLActionActivateAuraWithStacks,
			*APLActionCancelAura, *APLActionTriggerICD, *APLActionItemSwap, *APLActionMove,
			*APLActionAddComboPoints:
			// Casts nothing itself; its inner actions are walked below.
		default:
			return nil, false
		}
		pending = append(pending, action.impl.GetInnerActions()...)
	}
	return spells, true
}

// profiledSpells is the set the spell hit profile reads: the rotation's
// casts when they are readable, the whole spellbook otherwise.
func profiledSpells(character *Character) []*Spell {
	if character.Rotation != nil {
		if spells, ok := rotationCastSpells(character.Rotation); ok {
			return spells
		}
	}
	return character.Spellbook
}
