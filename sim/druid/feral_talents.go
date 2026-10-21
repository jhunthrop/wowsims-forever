package druid

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Heart of the Wild (node 104939, five ranks), rank text: "Increases your
// Intellect by 2%. In addition, while in Bear Form or Dire Bear Form your
// Stamina is increased by 4% and while in Cat Form your Strength is
// increased by 2%." Intellect and Strength gain 2% a rank, Stamina 4%
// (heartOfTheWildBearStaminaPerRank, bear_form.go).
const (
	heartOfTheWildMaxRank  = 5
	heartOfTheWildPerRank  = 0.02
	feralSwiftnessMaxRank  = 2
	naturalReactionMaxRank = 5
)

func (druid *Druid) applyHeartOfTheWildIntellect() {
	rank := clampRank(druid.Talents.HeartOfTheWild, heartOfTheWildMaxRank)
	if rank == 0 {
		return
	}
	druid.MultiplyStat(stats.Intellect, 1+heartOfTheWildPerRank*float64(rank))
}

// Dodge from the Feral tree, as percentage points of dodge chance (the
// engine's Dodge stat is flat percent):
//
//   - Feral Swiftness (node 104943): "Increases your movement speed while
//     in Cat Form by 15%, and increases your chance to Dodge by 2%" a rank.
//   - Natural Reaction (node 104954): "Increases your dodge chance by 1%,
//     and gives you a 20% chance to gain 5 Rage each time you dodge" a rank.
//
// Neither text names a form for the dodge, so it is permanent;
// the vanilla versions were Cat and Bear only.
const (
	feralSwiftnessDodgePerRank  = 2
	naturalReactionDodgePerRank = 1

	naturalReactionRageChancePerRank = 0.2
	naturalReactionRage              = 5.0
)

func (druid *Druid) applyFeralDodge() {
	dodge := feralSwiftnessDodgePerRank*float64(clampRank(druid.Talents.FeralSwiftness, feralSwiftnessMaxRank)) +
		naturalReactionDodgePerRank*float64(clampRank(druid.Talents.NaturalReaction, naturalReactionMaxRank))
	if dodge > 0 {
		druid.AddStat(stats.Dodge, dodge)
	}
}

// applyNaturalReaction implements the rage half of Natural Reaction: a bear
// that dodges gains 5 rage on 20% a rank of those dodges.
func (druid *Druid) applyNaturalReaction() {
	rank := clampRank(druid.Talents.NaturalReaction, naturalReactionMaxRank)
	if rank == 0 {
		return
	}

	chance := naturalReactionRageChancePerRank * float64(rank)
	rageMetrics := druid.NewRageMetrics(core.ActionID{SpellID: 417051})

	core.MakePermanent(druid.RegisterAura(core.Aura{
		Label: "Natural Reaction",
		OnSpellHitTaken: func(_ *core.Aura, sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			if !druid.InForm(Bear) || !result.Outcome.Matches(core.OutcomeDodge) {
				return
			}
			if sim.Proc(chance, "Natural Reaction") {
				druid.AddRage(sim, naturalReactionRage, rageMetrics)
			}
		},
	}))
}

// Rend and Tear (node 104953, five ranks): "Increases damage done by your
// melee abilities on Bleeding targets by 2%" a rank.
//
// unconfirmed: whether the client applies it with the other damage
// modifiers (additively) or after them; it is applied here as its own
// multiplier. Scoped to the Bear Form abilities this package registers: the
// Cat Form abilities (claw.go, rake.go, rip.go, shred.go, ravage.go and
// ferocious_bite.go) do not read it yet.
const (
	rendAndTearMaxRank       = 5
	rendAndTearDamagePerRank = 0.02
)

// RendAndTearMultiplier is Rend and Tear's multiplier on a bear
// ability's damage against target: the target counts as bleeding while the
// druid's own Lacerate is on it, or when the request says the raid keeps a
// bleed up (Druid.AssumeBleedActive).
func (druid *Druid) RendAndTearMultiplier(target *core.Unit) float64 {
	rank := clampRank(druid.Talents.RendAndTear, rendAndTearMaxRank)
	if rank == 0 || !druid.targetIsBleeding(target) {
		return 1
	}
	return 1 + rendAndTearDamagePerRank*float64(rank)
}

func (druid *Druid) targetIsBleeding(target *core.Unit) bool {
	return druid.AssumeBleedActive || (druid.Lacerate != nil && druid.Lacerate.Dot(target).IsActive())
}
