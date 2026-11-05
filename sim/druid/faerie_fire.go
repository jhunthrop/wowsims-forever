package druid

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (druid *Druid) registerFaerieFireSpell() {
	spellCode := SpellCode_DruidFaerieFire
	actionID := core.ActionID{SpellID: 9907}
	manaCostOptions := core.ManaCostOptions{
		FlatCost: 115,
	}
	gcd := core.GCDDefault
	ignoreHaste := false
	cd := core.Cooldown{}
	flatThreatBonus := 2. * 54
	flags := core.SpellFlagNone
	formMask := Humanoid | Moonkin

	druid.FaerieFireAuras = druid.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.FaerieFireAura(target)
	})

	/*
		if druid.InForm(Cat|Bear) && druid.Talents.FaerieFireFeral {
			spellCode = SpellCode_DruidFaerieFireFeral
			actionID = core.ActionID{SpellID: 17392}
			manaCostOptions = core.ManaCostOptions{}
			gcd = time.Second
			ignoreHaste = true
			formMask = Cat | Bear
			cd = core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: time.Second * 6,
			}
			druid.FaerieFireAuras = druid.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
				return core.FaerieFireFeralAura(target)
			})
		}
	*/
	// No swing reset: "Faerie Fire no longer resets your swing timer when
	// used" (Blizzard's Forever notes, 1 October 2026).
	flags |= core.SpellFlagAPL

	druid.FaerieFire = druid.RegisterSpell(formMask, core.SpellConfig{
		SpellCode:   spellCode,
		ActionID:    actionID,
		SpellSchool: core.SpellSchoolNature,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       flags,

		// This file always registers the top (rank 4, id 9907) Faerie
		// Fire; FaerieFireLevel[4] (constants_auto_gen.go) is that
		// rank's required level, 54.
		RequiredLevel: FaerieFireLevel[4],

		ManaCost: manaCostOptions,
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: gcd,
			},
			IgnoreHaste: ignoreHaste,
			CD:          cd,
		},

		ThreatMultiplier: 1,
		FlatThreatBonus:  flatThreatBonus,
		DamageMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				druid.FaerieFireAuras.Get(target).Activate(sim)
			}

			if druid.InForm(Humanoid | Moonkin) {
				druid.AutoAttacks.StopMeleeUntil(sim, sim.CurrentTime, false)
			}
		},

		RelatedAuras: []core.AuraArray{druid.FaerieFireAuras},
	})
}

func (druid *Druid) ShouldFaerieFire(sim *core.Simulation, target *core.Unit) bool {
	if druid.FaerieFire == nil {
		return false
	}

	if !druid.FaerieFire.IsReady(sim) {
		return false
	}

	debuff := druid.FaerieFireAuras.Get(target)
	return !debuff.IsActive() || debuff.RemainingDuration(sim) < time.Second*4
}

// faerieFireArmor is each Faerie Fire rank's armor reduction (effect 0,
// aura 674, client 1.60.1.70707: 770, 778, 9749, 9907), index 0 unused.
var faerieFireArmor = [FaerieFireRanks + 1]float64{0, 175, 285, 395, 505}

// Forever's feral forms cast Faerie Fire as Era's Faerie Fire (Feral) was:
// Cat Form (Passive) 3025 and Bear Form (Passive) 1178 take 100% off its
// cost (aura 108, SpellModOp 14, class mask 0x400) and add 6000 ms to its
// cooldown (aura 107, op 11); Cat Form also takes 500 ms off its global
// cooldown (op 21). Blizzard's 15 October beta notes: "Omen of Clarity no
// longer procs from a free Faerie Fire cast while using Cat Form, Bear Form,
// or Dire Bear Form" and "Faerie Fire now generates threat when used in
// Bear Form and Dire Bear Form".
const (
	faerieFireFeralCooldown = 6 * time.Second
	faerieFireFeralGCD      = time.Second
	faerieFireDuration      = 40 * time.Second
	faerieFireFeralThreat   = 2. * 54
	faerieFireFeralTag      = 1
)

// faerieFireRankAura is a rank's armor reduction on target. Rank 4 is the
// raid preset's own Faerie Fire aura, so a druid never recasts over it.
func faerieFireRankAura(target *core.Unit, rank int) *core.Aura {
	if rank == FaerieFireRanks {
		return core.FaerieFireAura(target)
	}
	return core.MinorArmorReductionAura(target, "Faerie Fire (rank "+strconv.Itoa(rank)+")",
		FaerieFireSpellId[rank], faerieFireArmor[rank], faerieFireDuration)
}

// registerFaerieFireFeralSpell registers Faerie Fire as Cat Form and Bear
// Form cast it: free, a 6 s cooldown and a 1 s global cooldown (1.5 s in Bear Form), no swing
// reset, and no Omen of Clarity roll (it carries no proc mask). The rank is
// the highest the level has learned.
func (druid *Druid) registerFaerieFireFeralSpell() {
	rank := core.HighestRankAtLevel(FaerieFireLevel[1:], druid.Level)
	if rank == 0 {
		return
	}

	druid.FaerieFireFeralAuras = druid.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return faerieFireRankAura(target, rank)
	})

	druid.FaerieFireFeral = druid.RegisterSpell(Cat|Bear, core.SpellConfig{
		// Tag 1 keeps it apart from the caster-form spell of the same id.
		ActionID:    core.ActionID{SpellID: FaerieFireSpellId[rank], Tag: faerieFireFeralTag},
		SpellSchool: core.SpellSchoolNature,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagAPL,

		RequiredLevel: FaerieFireLevel[rank],
		Rank:          rank,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: faerieFireFeralGCD,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: faerieFireFeralCooldown,
			},
			// Only Cat Form's passive shortens the global cooldown.
			ModifyCast: func(_ *core.Simulation, _ *core.Spell, cast *core.Cast) {
				if druid.InForm(Bear) {
					cast.GCD = core.GCDDefault
				}
			},
		},

		ThreatMultiplier: 1,
		FlatThreatBonus:  faerieFireFeralThreat,
		DamageMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				druid.FaerieFireFeralAuras.Get(target).Activate(sim)
			}
		},

		RelatedAuras: []core.AuraArray{druid.FaerieFireFeralAuras},
	})
}
