package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Missile Barrage is the Arcane tree's tier-4 bool talent (node 105802,
// max_rank 1): "Gives your Arcane Blast spell a 40% chance, and your
// Fireball, Frostbolt, and Frostfire Bolt spells a 20% chance to reduce
// the channeled duration of your next Arcane Missiles spell by 50%,
// reduce the Mana cost by 100%, and missiles fire every 0.5 sec."
// Frostfire Bolt has no ability file in this package, so its 20% chance
// is inert until one exists; Arcane Blast and Frostbolt are both live.
//
// The buff is spell 400589 ("Missile Barrage", duration_ms 15000);
// arcane_missiles.go's ApplyEffects is what reads it to halve the tick
// length, and this file's own Cost.Multiplier swap is what makes the
// next Arcane Missiles free - both only take effect on the very next
// cast, per the tooltip's "your next Arcane Missiles".
const (
	missileBarrageBuffSpellId       int32 = 400589
	missileBarrageDuration                = time.Second * 15
	missileBarrageArcaneBlastChance       = 0.40
	missileBarrageOtherChance             = 0.20

	// "missiles fire every 0.5 sec" instead of the normal 1 sec tick
	// (arcane_missiles.go's tickLength), with the same tick count, which
	// is what "reduce the channeled duration ... by 50%" comes out to.
	missileBarrageTickLength = time.Millisecond * 500
)

func (mage *Mage) registerMissileBarrage() {
	if !mage.Talents.MissileBarrage {
		return
	}

	// Arcane Missiles is registered from Initialize, which runs after
	// ApplyTalents, so this captures it as it registers rather than
	// reading mage.ArcaneMissiles here (nil at this point).
	var arcaneMissilesSpells []*core.Spell
	mage.OnSpellRegistered(func(spell *core.Spell) {
		if spell.SpellCode == SpellCode_MageArcaneMissiles {
			arcaneMissilesSpells = append(arcaneMissilesSpells, spell)
		}
	})

	mage.MissileBarrageAura = mage.RegisterAura(core.Aura{
		Label:    "Missile Barrage",
		ActionID: core.ActionID{SpellID: missileBarrageBuffSpellId},
		Duration: missileBarrageDuration,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range arcaneMissilesSpells {
				if spell.Cost != nil {
					spell.Cost.Multiplier = 0
				}
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range arcaneMissilesSpells {
				if spell.Cost != nil {
					spell.Cost.Multiplier = 100
				}
			}
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			// Mirrors ClearcastingAura's own guard (applyArcaneConcentration,
			// talents.go): OnCastComplete runs for every active aura after
			// the triggering cast too, so don't consume the buff on the
			// same event that granted it.
			if aura.RemainingDuration(sim) == aura.Duration {
				return
			}
			if spell.SpellCode != SpellCode_MageArcaneMissiles {
				return
			}
			aura.Deactivate(sim)
		},
	})

	mage.RegisterAura(core.Aura{
		Label:    "Missile Barrage Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnCastComplete: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell) {
			var chance float64
			switch spell.SpellCode {
			case SpellCode_MageArcaneBlast:
				chance = missileBarrageArcaneBlastChance
			case SpellCode_MageFireball, SpellCode_MageFrostbolt:
				chance = missileBarrageOtherChance
			default:
				return
			}

			if sim.Proc(chance, "Missile Barrage") {
				mage.MissileBarrageAura.Activate(sim)
			}
		},
	})
}
