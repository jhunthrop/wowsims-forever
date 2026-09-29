package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// counterattackLearnLevels are Counterattack's four rank learn levels;
// source: 1.60.1.70009 client spell data ("Counterattack", spells
// 19306/1242634/20909/20910). Ranks 1 and 2 both carry spell_level 30 in
// the client's own data (a duplicate, not a typo this file should paper
// over): core.HighestRankAtLevel resolves a tie to the later (higher)
// entry, so a level-30 hunter gets rank 2 -- the client's higher-damage
// row for that level -- which is the only reading of two equal levels
// that does not throw away a row.
var counterattackLearnLevels = []int{30, 30, 42, 54}

// counterattackBaseDamage and counterattackManaCost are ranks 1-4's flat
// bonus damage and mana cost; source: 1.60.1.70009 client spell data.
var counterattackBaseDamage = [5]float64{0, 26, 40, 70, 110}
var counterattackManaCost = [5]float64{0, 30, 45, 65, 85}

func (hunter *Hunter) getCounterattackConfig(rank int) core.SpellConfig {
	spellId := [5]int32{0, 19306, 1242634, 20909, 20910}[rank]
	baseDamage := counterattackBaseDamage[rank]
	manaCost := counterattackManaCost[rank]
	level := counterattackLearnLevels[rank-1]

	return core.SpellConfig{
		SpellCode:     SpellCode_HunterCounterattack,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
		Flags:         core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		Rank:          rank,
		RequiredLevel: level,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: time.Second * 5,
			},
		},

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget <= core.MaxMeleeAttackDistance && hunter.CounterattackProcAura.IsActive()
		},

		CritDamageBonus:  hunter.mortalShots() + hunter.predatorsEdgeCritDamage(),
		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// Client tooltip: "Counterattack cannot be blocked, dodged,
			// or parried" -- OutcomeMeleeWeaponSpecialNoBlockDodgeParry's
			// weapon-damage-aware sibling doesn't exist, but Counterattack's
			// own damage is a flat amount rather than weapon-scaled (the
			// client's effect is a school-damage bonus, not a weapon-damage
			// one, unlike Raptor Strike/Mongoose Bite), so the plain
			// no-block/dodge/parry outcome is the right one here.
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialNoBlockDodgeParry)
			hunter.CounterattackProcAura.Deactivate(sim)
		},
	}
}

func (hunter *Hunter) registerCounterattackSpell() {
	if !hunter.Talents.Counterattack {
		return
	}

	hunter.CounterattackProcAura = hunter.RegisterAura(core.Aura{
		Label:    "Counterattack",
		ActionID: core.ActionID{SpellID: 19306},
		Duration: time.Second * 5,
	})

	hunter.RegisterAura(core.Aura{
		Label:    "Counterattack Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.DidParry() {
				hunter.CounterattackProcAura.Activate(sim)
			}
		},
	})

	rank := core.HighestRankAtLevel(counterattackLearnLevels, hunter.Level)
	if rank == 0 {
		return
	}

	config := hunter.getCounterattackConfig(rank)
	hunter.Counterattack = hunter.GetOrRegisterSpell(config)
}
