package shaman

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Lava Burst is Elemental's tier-6 capstone (talents/shaman.json node
// 104758, prereq Lightning Overload rank 3): a single-rank talent that
// grants a real per-level nuke, gated the same way every other Shaman
// nuke is (registerLavaBurstSpell only runs once the talent is spent),
// with its own castable ranks from spellranks.json. The guide calls it
// the tree's capstone nuke, but the engine had no lava_burst.go and the
// talent wasn't even in the generated proto, so a shaman who spent the
// point had a dead rotation line.
//
// Client numbers (spellconst/shaman.json), all three ranks share
// cast_time_ms 2500, gcd_ms 1500, cooldown_ms 0, category_cooldown_ms
// 10000 (a 10s cooldown, not the 8s Classic remembers -- the client
// wins):
//   - rank 1 (408490, level 40): cost 165, amount 164, sp_coefficient 0.714
//   - rank 2 (1238299, level 50): cost 230, amount 196, sp_coefficient 0.714
//   - rank 3 (1238300, level 60): cost 265, amount 220, sp_coefficient 0.714
//   - every rank's effect 1 is effect=3 (Dummy), amount 20 -- the client's
//     own talent text spells this out exactly: "If your Flame Shock is on
//     the target, Lava Burst deals 20% increased damage." Classic/SoD's
//     guaranteed-crit-on-Flame-Shocked-targets behaviour is NOT what this
//     client's tooltip says, so the client wins: a flat +20% multiplier,
//     not a crit guarantee.
//
// Like every other Shaman nuke, Convection's mana discount ("Shock,
// Lightning Bolt, Lava Burst, and Chain Lightning") and Call of Flame's
// damage bonus ("Fire Totems and ... Flame Shock, Fire Nova, and Lava
// Burst") both name Lava Burst explicitly, so both apply here the same
// way electric_spell.go and fire_totems.go already apply them. Elemental
// Fury's crit-damage bonus applies automatically via the SpellFlagShaman
// + DefenseTypeMagic hook in talents.go and needs no special case here.
const LavaBurstRanks = 3

var LavaBurstSpellId = [LavaBurstRanks + 1]int32{0, 408490, 1238299, 1238300}
var LavaBurstBaseDamage = [LavaBurstRanks + 1]float64{0, 164, 196, 220}
var LavaBurstManaCost = [LavaBurstRanks + 1]float64{0, 165, 230, 265}
var LavaBurstLevel = [LavaBurstRanks + 1]int{0, 40, 50, 60}

const LavaBurstSpellCoefficient = 0.71399998665

// lavaBurstFlameShockDamageMultiplier is the talent's flat +20% bonus
// when the caster's own Flame Shock is on the target.
const lavaBurstFlameShockDamageMultiplier = 1.20

// LavaBurstCooldown is the client's category_cooldown_ms (10000) for
// every rank -- exported so the elemental package's test (Lava Burst's
// only spec, since a shaman who hasn't spent the talent has no spell to
// test) can assert against it without duplicating the number.
const LavaBurstCooldown = time.Second * 10

func (shaman *Shaman) registerLavaBurstSpell() {
	if !shaman.Talents.LavaBurst {
		return
	}

	shaman.LavaBurst = make([]*core.Spell, 0)
	for rank := 1; rank <= LavaBurstRanks; rank++ {
		config := shaman.newLavaBurstSpellConfig(rank)

		if config.RequiredLevel <= int(shaman.Level) {
			shaman.LavaBurst = append(shaman.LavaBurst, shaman.RegisterSpell(config))
		}
	}
}

func (shaman *Shaman) newLavaBurstSpellConfig(rank int) core.SpellConfig {
	spellId := LavaBurstSpellId[rank]
	baseDamage := LavaBurstBaseDamage[rank]
	manaCost := LavaBurstManaCost[rank]
	level := LavaBurstLevel[rank]

	return core.SpellConfig{
		SpellCode:     SpellCode_ShamanLavaBurst,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolFire,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskSpellDamage,
		Flags:         SpellFlagShaman | core.SpellFlagAPL,
		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost:   manaCost,
			Multiplier: 100 - 2*shaman.Talents.Convection,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond * 2500,
			},
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				castTime := shaman.ApplyCastSpeedForSpell(cast.CastTime, spell)
				shaman.AutoAttacks.StopMeleeUntil(sim, sim.CurrentTime+castTime, false)
			},
			CD: core.Cooldown{
				Timer:    shaman.NewTimer(),
				Duration: LavaBurstCooldown,
			},
		},

		DamageMultiplier: shaman.callOfFlameMultiplier(),
		ThreatMultiplier: 1,
		BonusCoefficient: LavaBurstSpellCoefficient,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			oldMultiplier := spell.DamageMultiplier
			if shaman.GetActiveFlameShockSpell(target) != nil {
				spell.DamageMultiplier *= lavaBurstFlameShockDamageMultiplier
			}

			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			spell.DamageMultiplier = oldMultiplier

			spell.DealDamage(sim, result)
		},
	}
}

// GetActiveFlameShockSpell returns whichever rank of the caster's own
// Flame Shock has an active DoT on target, or nil -- the same
// "does my own DoT own the target right now" lookup warlock's
// getActiveImmolateSpell uses for Incinerate/Conflagrate. Exported (Lava
// Burst is the only caller so far, but it lives in a different package
// from every Shaman spec that could talent into it) rather than
// unexported like the warlock version, whose test shares its package.
func (shaman *Shaman) GetActiveFlameShockSpell(target *core.Unit) *core.Spell {
	for _, flameShockSpell := range shaman.FlameShock {
		if flameShockSpell != nil && flameShockSpell.Dot(target).IsActive() {
			return flameShockSpell
		}
	}
	return nil
}
