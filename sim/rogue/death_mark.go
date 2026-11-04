package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Death Mark is the Subtlety capstone (node 110866, 70755), taught in four
// ranks (SkillLineAbility 59901-59904: 1316709@40, 1325756@48, 1325757@56,
// 1325758@60). Each rank: 25 Energy and a combo-point cost (SpellPower),
// 10 sec cooldown and 1 sec GCD (SpellCooldowns), DurationIndex 32 (6 sec).
// Its effects are server dummies (aura 3 and aura 4, bp 0), so the whole
// mechanic is the rank text, with its variables from
// SpellDescriptionVariables 1021 (wago.tools, build 1.60.1.70755):
//
//	stored: combo points x $<ratio> % of the damage your attacks deal,
//	        Garrote and Rupture at twice the rate; $ratio = 12
//	cap:    ((base + cp x perPoint + cp x $<maxAP> x AP) x ($mcrit/100 + 1))
//	        x $<coup>; $maxAP = 0.04
//
// $<coup> is not in the variables table; Coup de Grace's e1 (aura 108 on
// Death Mark's mask_2 4) raises "the damage and storage rate", so it is
// modelled as 1 + Coup de Grace's bonus on both the storage rate and the
// cap, the same factor the client's untalented tooltip renders as 1.
//
// After 6 sec the stored damage, up to the cap, is dealt. What it stores
// is damage already dealt (past armor and modifiers), and the cap already
// carries crit, so the payout neither rolls nor is mitigated again. Damage
// from a spell whose procs the client suppresses (Shank, proc mask empty)
// is not stored. AP and crit are read when the mark is cast.
const (
	deathMarkRanks          = 4
	deathMarkEnergyCost     = 25.0
	deathMarkCooldown       = 10 * time.Second
	deathMarkDuration       = 6 * time.Second
	deathMarkRatioPerPoint  = 0.12
	deathMarkAPPerPoint     = 0.04
	deathMarkBleedRateScale = 2.0
	// deathMarkPayoutTag keeps the payout's metrics apart from the cast's
	// per-combo-point splits (tags 0-5).
	deathMarkPayoutTag = 10
)

var deathMarkLearnLevels = []int{40, 48, 56, 60}

// deathMarkCapBase and deathMarkCapPerPoint are each rank's cap terms, from
// its rank text (63+cp*102, 98+cp*158, 138+cp*217, 156+cp*245).
var deathMarkCapBase = [deathMarkRanks + 1]float64{0, 63, 98, 138, 156}
var deathMarkCapPerPoint = [deathMarkRanks + 1]float64{0, 102, 158, 217, 245}

// deathMarkCap is the most a mark can deal.
func deathMarkCap(rank int, comboPoints int32, attackPower, critPercent, coupDeGrace float64) float64 {
	cp := float64(comboPoints)
	return (deathMarkCapBase[rank] + cp*deathMarkCapPerPoint[rank] + cp*deathMarkAPPerPoint*attackPower) *
		(critPercent/100 + 1) * (1 + coupDeGrace)
}

// deathMarkStorageRate is the share of a hit a mark stores.
func deathMarkStorageRate(comboPoints int32, coupDeGrace float64, bleed bool) float64 {
	rate := float64(comboPoints) * deathMarkRatioPerPoint * (1 + coupDeGrace)
	if bleed {
		rate *= deathMarkBleedRateScale
	}
	return rate
}

type deathMarkState struct {
	stored      float64
	cap         float64
	comboPoints int32
}

func (rogue *Rogue) registerDeathMarkSpell() {
	if !rogue.Talents.DeathMark {
		return
	}
	rank := core.HighestRankAtLevel(deathMarkLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	actionID := core.ActionID{SpellID: DeathMarkSpellId[rank]}
	coup := rogue.coupDeGrace()
	states := make(map[*core.Unit]*deathMarkState)

	payout := rogue.RegisterSpell(core.SpellConfig{
		ActionID:    actionID.WithTag(deathMarkPayoutTag),
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskEmpty,
		Flags: core.SpellFlagMeleeMetrics | core.SpellFlagIgnoreResists | core.SpellFlagIgnoreModifiers |
			core.SpellFlagNoOnDamageDealt | core.SpellFlagPassiveSpell,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			state := states[target]
			spell.CalcAndDealDamage(sim, target, min(state.stored, state.cap), spell.OutcomeAlwaysHit)
		},
	})

	marks := rogue.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return target.RegisterAura(core.Aura{
			Label:    "Death Mark",
			ActionID: actionID,
			Duration: deathMarkDuration,
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				// Only a mark that runs its 6 sec pays out; one cut short
				// by the end of the fight deals nothing.
				if sim.CurrentTime >= aura.ExpiresAt() {
					payout.Cast(sim, aura.Unit)
				}
				delete(states, aura.Unit)
			},
		})
	})

	store := func(spell *core.Spell, result *core.SpellResult) {
		if result.Damage <= 0 || spell.ProcMask == core.ProcMaskEmpty || !marks.Get(result.Target).IsActive() {
			return
		}
		state := states[result.Target]
		bleed := spell == rogue.Garrote || spell == rogue.Rupture
		state.stored += result.Damage * deathMarkStorageRate(state.comboPoints, coup, bleed)
	}
	core.MakePermanent(rogue.RegisterAura(core.Aura{
		Label: "Death Mark Storage",
		OnSpellHitDealt: func(_ *core.Aura, _ *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			store(spell, result)
		},
		OnPeriodicDamageDealt: func(_ *core.Aura, _ *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			store(spell, result)
		},
	}))

	rogue.DeathMark = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:      SpellCode_RogueDeathMark,
		ClassSpellMask: RogueSpellMaskDeathMark,
		ActionID:       actionID,
		Flags:          core.SpellFlagAPL,
		MetricSplits:   6,
		RequiredLevel:  deathMarkLearnLevels[rank-1],
		Rank:           rank,

		EnergyCost: core.EnergyCostOptions{
			Cost: deathMarkEnergyCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: deathMarkCooldown,
			},
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				spell.SetMetricsSplit(spell.Unit.ComboPoints())
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return rogue.ComboPoints() > 0
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			comboPoints := rogue.ComboPoints()
			// The 10 sec cooldown outlasts the 6 sec mark, so a cast never
			// finds one running.
			mark := marks.Get(target)
			critPercent := rogue.GetStat(stats.Crit)/core.CritRatingPerCritChance + rogue.GetStat(stats.MeleeCrit)/core.CritRatingPerCritChance
			states[target] = &deathMarkState{
				cap:         deathMarkCap(rank, comboPoints, spell.MeleeAttackPower(target), critPercent, coup),
				comboPoints: comboPoints,
			}
			mark.Activate(sim)
			rogue.SpendComboPoints(sim, spell)
		},
	})
	rogue.Finishers = append(rogue.Finishers, rogue.DeathMark)
}
