package warlock

import (
	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

const LifeTapRanks = 6

var LifeTapSpellId = [LifeTapRanks + 1]int32{0, 1454, 1455, 1456, 11687, 11688, 11689}

// LifeTapAmount is each rank's amount as the client states it (effect 0,
// a dummy effect that parks the flat the spell's script reads), checked
// rank by rank against spellconst/warlock.json by
// TestLifeTapAmountTableMatchesTheClient. The spell text reads "Converts
// ${($m1+$SPI*1)*(1+$18182m1/100)} Health into ${...} Mana for you. Spirit
// increases the amount converted.": the amount is the effect plus one
// point per point of Spirit, times Improved Life Tap. The effect grows by
// one a caster level above the spell's own, so rank 6 reads 424 at level
// 60 (the client's older table said 420, the value at the spell's own
// level).
//
// The engine used to scale this with spell power at a 0.8 coefficient,
// which neither the effect (coefficient 0) nor the text states.
var LifeTapAmount = [LifeTapRanks + 1]clientdamage.Effect{
	{},
	{Amount: 20, PerLevel: 1, SpellLevel: 6, MaxLevel: 16},
	{Amount: 65, PerLevel: 1, SpellLevel: 16, MaxLevel: 26},
	{Amount: 130, PerLevel: 1, SpellLevel: 26, MaxLevel: 36},
	{Amount: 210, PerLevel: 1, SpellLevel: 36, MaxLevel: 46},
	{Amount: 300, PerLevel: 1, SpellLevel: 46, MaxLevel: 56},
	{Amount: 420, PerLevel: 1, SpellLevel: 56, MaxLevel: 66},
}

// improvedLifeTapPerRank is the talent's share of the amount a rank:
// 10% and 20% in the live tree.
const improvedLifeTapPerRank = 0.1

// lifeTapMana is the mana (and, to the caster, the health) one cast
// converts: the rank's amount at the caster's level plus the caster's
// Spirit, raised by Improved Life Tap.
func (warlock *Warlock) lifeTapMana(rank int) float64 {
	amount := LifeTapAmount[rank].Center(int(warlock.Level)) + warlock.GetStat(stats.Spirit)
	return amount * (1 + improvedLifeTapPerRank*float64(warlock.Talents.ImprovedLifeTap))
}

func (warlock *Warlock) getLifeTapBaseConfig(rank int) core.SpellConfig {
	spellId := LifeTapSpellId[rank]
	level := [LifeTapRanks + 1]int{0, 6, 16, 26, 36, 46, 56}[rank]

	actionID := core.ActionID{SpellID: spellId}

	manaMetrics := warlock.NewManaMetrics(actionID)
	for _, pet := range warlock.BasePets {
		pet.LifeTapManaMetrics = pet.NewManaMetrics(actionID)
	}

	return core.SpellConfig{
		ActionID:      actionID,
		SpellSchool:   core.SpellSchoolShadow,
		SpellCode:     SpellCode_WarlockLifeTap,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskSpellDamage,
		Flags:         core.SpellFlagAPL | core.SpellFlagResetAttackSwing | core.SpellFlagBinary | WarlockFlagAffliction,
		RequiredLevel: level,
		Rank:          rank,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			restore := warlock.lifeTapMana(rank)

			if warlock.IsTanking() {
				spell.DealDamage(sim, spell.CalcDamage(sim, spell.Unit, restore, spell.OutcomeAlwaysHit))
			}

			warlock.AddMana(sim, restore, manaMetrics)
			warlock.shareDemonicEnergiesMana(sim, restore)
		},
	}
}

func (warlock *Warlock) registerLifeTapSpell() {
	warlock.LifeTap = make([]*core.Spell, 0)
	for i := 1; i <= LifeTapRanks; i++ {
		config := warlock.getLifeTapBaseConfig(i)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.LifeTap = append(warlock.LifeTap, warlock.GetOrRegisterSpell(config))
		}
	}
}
