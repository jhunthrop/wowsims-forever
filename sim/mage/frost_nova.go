package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Frost Nova is a baseline trainer ability, not a talent - every mage
// learns it, so it registers from Initialize like Frostbolt and
// Fireball rather than from ApplyTalents. It carries the mask
// MageSpellMaskFrostNova, which talents.go's Improved Frost Nova
// (cooldown) and the Frost damage group already reference; both were
// inert until this file existed.
//
// Ranks and numbers come from constants_auto_gen.go's generated
// FrostNova* arrays (spellranks.json "Frost Nova": 122/865/6131/10230
// at levels 10/26/40/54; no id collision here, unlike Arcane Blast, so
// the generated arrays are used directly rather than hand-copied).
// Every rank agrees on cost_type 0 (mana), duration_ms 8000 and
// category_cooldown_ms 25000 with cooldown_ms 0 -
// FrostNovaCooldownMS/EffectiveCooldownMS (sim/core/spellconst) is what
// reads the category cooldown when the spell's own is zero, and
// Improved Frost Nova's -2s/rank targets exactly that 25s.
const frostNovaCooldown = time.Second * 25
const frostNovaFrozenDuration = time.Second * 8

func (mage *Mage) registerFrostNovaSpell() {
	mage.FrostNova = make([]*core.Spell, FrostNovaRanks+1)
	mage.FrozenAuras = mage.NewEnemyAuraArray(mage.newFrozenAura)

	cdTimer := mage.NewTimer()
	for rank := 1; rank <= FrostNovaRanks; rank++ {
		config := mage.getFrostNovaConfig(rank, cdTimer)

		if config.RequiredLevel <= int(mage.Level) {
			mage.FrostNova[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) getFrostNovaConfig(rank int, cdTimer *core.Timer) core.SpellConfig {
	roll := mage.clientRoll(FrostNovaBaseDamage[rank], FrostNovaPointsPerLevel[rank], FrostNovaLevel[rank], FrostNovaMaxLevel[rank])
	baseDamageLow, baseDamageHigh := roll[0], roll[1]

	return core.SpellConfig{
		ActionID:         core.ActionID{SpellID: FrostNovaSpellId[rank]},
		ClassSpellMask:   MageSpellMaskFrostNova,
		SpellCode:        SpellCode_MageFrostNova,
		SpellSchool:      core.SpellSchoolFrost,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: roll,
		Flags:            SpellFlagMage | core.SpellFlagAPL,

		RequiredLevel: FrostNovaLevel[rank],
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: FrostNovaManaCost[rank],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    cdTimer,
				Duration: frostNovaCooldown,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: FrostNovaSpellCoeff[rank],

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			for _, aoeTarget := range sim.Encounter.TargetUnits {
				baseDamage := sim.Roll(baseDamageLow, baseDamageHigh)
				result := spell.CalcAndDealDamage(sim, aoeTarget, baseDamage, spell.OutcomeMagicHitAndCrit)

				if result.Landed() && mage.canFreeze(aoeTarget) {
					mage.FrozenAuras.Get(aoeTarget).Activate(sim)
				}
			}
		},
	}
}

// canFreeze reports whether Frost Nova's root (and so the Frozen state
// Ice Lance and Shatter read) can land on target at all.
//
// The engine has no dedicated CC-immunity flag on core.Unit or
// core.Target (proto/common.proto's Target message carries only level
// and MobType), and no ability file in this fork gates a root or stun
// on either yet, so there is no existing pattern to match beyond the
// obvious one: every raid boss in the sim is built above
// core.CharacterMaxLevel (target.go's defaultRaidBossLevel is
// CharacterMaxLevel+3), the way every WoW boss has always resisted or
// been immune to snares, roots and disorients. A target at or below the
// player level cap can be frozen; a boss cannot.
func (mage *Mage) canFreeze(target *core.Unit) bool {
	return target.Level <= core.CharacterMaxLevel
}

// newFrozenAura is Frost Nova's root, the Frozen state ice_lance.go's
// isTargetFrozen (and Shatter, talents.go) read. It shares Frost Nova
// rank 1's spell id (122) as a fixed marker regardless of which rank
// applied it, the way core.WintersChillAura and core.ImprovedScorchAura
// (sim/core/debuffs.go) each use one id for every source rank of their
// own talent.
//
// Duration is the client's duration_ms (8000, every rank agrees); the
// client's second effect on Frost Nova (effect 6, aura 26) is the root
// itself and carries no separate number to read - it is a boolean
// snare/root, which this Aura's presence models directly.
func (mage *Mage) newFrozenAura(target *core.Unit) *core.Aura {
	return target.GetOrRegisterAura(core.Aura{
		Label:    "Frozen",
		ActionID: core.ActionID{SpellID: FrostNovaSpellId[1]},
		Duration: frostNovaFrozenDuration,
	})
}
