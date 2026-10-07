package warlock

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const (
	unstableAfflictionTickLength = 3 * time.Second
	unstableAfflictionNumTicks   = 6 // the client's 18 s duration at a 3 s period
	// unstableAfflictionTickCoefficient is the client's sp_coefficient on the
	// periodic effect, per tick like Corruption's.
	unstableAfflictionTickCoefficient = 0.2
)

// Ids, levels, costs and cast time are constants_auto_gen.go's
// UnstableAffliction* arrays. UnstableAfflictionTickDamage is spellconst/warlock.json's own per-tick
// amount for ids 427717, 1242971 and 1242972 (86, 123 and 174 every 3 s, no
// growth: the 516, 738 and 1044 the tooltips state over 18 s). The client
// learns rank 2 at 60 and rank 3 at 50, which UnstableAfflictionLevel keeps.
var UnstableAfflictionTickDamage = [UnstableAfflictionRanks + 1]clientdamage.Effect{
	{},
	{Amount: 86, SpellLevel: 40},
	{Amount: 123, SpellLevel: 60},
	{Amount: 174, SpellLevel: 50},
}

func (warlock *Warlock) getUnstableAfflictionConfig(rank int) core.SpellConfig {
	damage := UnstableAfflictionTickDamage[rank]
	casterLevel := int(warlock.Level)
	baseDamage := damage.Center(casterLevel)

	return core.SpellConfig{
		SpellCode:        SpellCode_WarlockUnstableAffliction,
		ActionID:         core.ActionID{SpellID: UnstableAfflictionSpellId[rank]},
		SpellSchool:      core.SpellSchoolShadow,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		Flags:            core.SpellFlagAPL | core.SpellFlagResetAttackSwing | core.SpellFlagPureDot | WarlockFlagAffliction,
		Rank:             rank,
		RequiredLevel:    UnstableAfflictionLevel[rank],
		ClientBaseDamage: damage.Range(casterLevel),

		ManaCost: core.ManaCostOptions{
			FlatCost: UnstableAfflictionManaCost[rank],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				CastTime: time.Duration(UnstableAfflictionCastTime[rank]) * time.Millisecond,
				GCD:      core.GCDDefault,
			},
		},

		BonusCoefficient: unstableAfflictionTickCoefficient, // the report compares the spell's, which a pure DoT never reads
		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "UnstableAffliction-" + warlock.Label + strconv.Itoa(rank),
			},
			NumberOfTicks:    unstableAfflictionNumTicks,
			TickLength:       unstableAfflictionTickLength,
			BonusCoefficient: unstableAfflictionTickCoefficient,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
			if result.Landed() {
				dot := spell.Dot(target)
				warlock.cancelExclusiveDots(sim, target, dot)
				dot.Apply(sim)
			}
			spell.DealOutcome(sim, result)
		},
		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, useSnapshot bool) *core.SpellResult {
			if useSnapshot {
				dot := spell.Dot(target)
				return dot.CalcSnapshotDamage(sim, target, dot.Spell.OutcomeExpectedMagicAlwaysHit)
			}
			return spell.CalcPeriodicDamage(sim, target, baseDamage, spell.OutcomeExpectedMagicAlwaysHit)
		},
	}
}

func (warlock *Warlock) registerUnstableAfflictionSpell() {
	warlock.UnstableAffliction = make([]*core.Spell, 0, UnstableAfflictionRanks)
	for rank := 1; rank <= UnstableAfflictionRanks; rank++ {
		config := warlock.getUnstableAfflictionConfig(rank)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.UnstableAffliction = append(warlock.UnstableAffliction, warlock.GetOrRegisterSpell(config))
		}
	}
}

// cancelExclusiveDots enforces the live text shared by Unstable Affliction and
// Immolate, "Only one Unstable Affliction or Immolate per Warlock can be
// active on any one target": applying keep drops every other Immolate and
// Unstable Affliction dot on the target.
func (warlock *Warlock) cancelExclusiveDots(sim *core.Simulation, target *core.Unit, keep *core.Dot) {
	for _, group := range [][]*core.Spell{warlock.Immolate, warlock.UnstableAffliction} {
		for _, spell := range group {
			if dot := spell.Dot(target); dot != keep && dot.IsActive() {
				dot.Deactivate(sim)
			}
		}
	}
}
