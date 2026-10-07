package shaman

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// Fire Nova is a Forever spell, not the vanilla Fire Nova Totem: five
// trainable ranks (SkillLineAbility learn rows 8734-8738 at 12/22/32/
// 42/52) of an instant 1.5 s GCD spell on a 10 s cooldown that "instantly
// inflicts fire damage to enemies within 10 yd of your active Fire
// totem". The vanilla totem ids (1535, 8498, 8499, 11314, 11315) have no
// learn row, so that registration is gone. The cast spells carry the
// mana, cooldown and totem condition; the damage rides on the linked
// spells 8349/8502/8503/11306/11307, whose effect is the client's roll
// and spell-power coefficient.
const FireNovaLearnRanks = 5

var FireNovaLearnSpellId = [FireNovaLearnRanks + 1]int32{0, 408341, 408342, 408343, 408344, 408345}
var FireNovaDamageSpellId = [FireNovaLearnRanks + 1]int32{0, 8349, 8502, 8503, 11306, 11307}
var FireNovaDamage = [FireNovaLearnRanks + 1]clientdamage.Effect{
	{},
	{Amount: 52, Variance: 0.153846, PerLevel: 1.1, SpellLevel: 12, MaxLevel: 17},
	{Amount: 109, Variance: 0.12844, PerLevel: 1.6, SpellLevel: 22, MaxLevel: 27},
	{Amount: 196, Variance: 0.122449, PerLevel: 2.2, SpellLevel: 32, MaxLevel: 37},
	{Amount: 299, Variance: 0.120401, PerLevel: 2.8, SpellLevel: 42, MaxLevel: 47},
	{Amount: 419, Variance: 0.109785, PerLevel: 3.4, SpellLevel: 52, MaxLevel: 57},
}
var FireNovaDamageCoeff = [FireNovaLearnRanks + 1]float64{0, .1, .143, .143, .143, .143}
var FireNovaLearnManaCost = [FireNovaLearnRanks + 1]float64{0, 95, 170, 280, 395, 520}
var FireNovaLearnLevel = [FireNovaLearnRanks + 1]int{0, 12, 22, 32, 42, 52}

// FireNovaCooldown is the client's category cooldown on every rank.
const FireNovaCooldown = 10 * time.Second

// Improved Fire Nova (spell 16086, 2 points): "Increases the damage done
// by your Fire Nova spell by 20% and reduces its cooldown by 2 sec."
// The client states one effect for the pair of ranks, so each point is
// taken as half of it.
const (
	improvedFireNovaDamagePerPoint   = 0.10
	improvedFireNovaCooldownPerPoint = time.Second
)

func (shaman *Shaman) registerFireNovaSpell() {
	shaman.FireNova = make([]*core.Spell, FireNovaLearnRanks+1)
	shaman.FireNovaBlast = make([]*core.Spell, FireNovaLearnRanks+1)

	for rank := 1; rank <= FireNovaLearnRanks; rank++ {
		if FireNovaLearnLevel[rank] > int(shaman.Level) {
			continue
		}
		blast := shaman.registerFireNovaBlast(rank)
		shaman.FireNovaBlast[rank] = blast
		shaman.FireNova[rank] = shaman.RegisterSpell(shaman.newFireNovaCastConfig(rank, blast))
	}
}

// fireNovaDamageMultiplier adds Call of Flame (+5% a point) and Improved
// Fire Nova (+10% a point) the way the client's percent modifiers stack.
func (shaman *Shaman) fireNovaDamageMultiplier() float64 {
	return shaman.callOfFlameMultiplier() + improvedFireNovaDamagePerPoint*float64(shaman.Talents.ImprovedFireNova)
}

func (shaman *Shaman) registerFireNovaBlast(rank int) *core.Spell {
	damage := FireNovaDamage[rank]
	casterLevel := int(shaman.Level)

	return shaman.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_ShamanFireNova,
		ActionID:    core.ActionID{SpellID: FireNovaDamageSpellId[rank]},
		SpellSchool: core.SpellSchoolFire,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       SpellFlagShaman,

		RequiredLevel: FireNovaLearnLevel[rank],

		DamageMultiplier: shaman.fireNovaDamageMultiplier(),
		ThreatMultiplier: 1,
		BonusCoefficient: FireNovaDamageCoeff[rank],
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			for _, enemy := range sim.Encounter.TargetUnits {
				spell.CalcAndDealDamage(sim, enemy, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
			}
		},
	})
}

func (shaman *Shaman) newFireNovaCastConfig(rank int, blast *core.Spell) core.SpellConfig {
	return core.SpellConfig{
		SpellCode:     SpellCode_ShamanFireNova,
		ActionID:      core.ActionID{SpellID: FireNovaLearnSpellId[rank]},
		SpellSchool:   core.SpellSchoolFire,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskSpellDamage,
		Flags:         SpellFlagShaman | core.SpellFlagAPL,
		RequiredLevel: FireNovaLearnLevel[rank],
		Rank:          rank,

		ManaCost: core.ManaCostOptions{FlatCost: FireNovaLearnManaCost[rank]},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: core.GCDDefault},
			CD: core.Cooldown{
				Timer:    shaman.NewTimer(),
				Duration: FireNovaCooldown - improvedFireNovaCooldownPerPoint*time.Duration(shaman.Talents.ImprovedFireNova),
			},
		},

		ExtraCastCondition: func(sim *core.Simulation, _ *core.Unit) bool {
			return shaman.ActiveTotems[FireTotem] != nil && sim.CurrentTime < shaman.TotemExpirations[FireTotem]
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
			blast.ApplyEffects(sim, target, blast)
		},
	}
}
