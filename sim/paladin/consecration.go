package paladin

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// ConsecrationNumberOfTicks and ConsecrationTickLength: source
// 1.60.1.70009 client spell data (spellconst/paladin.json). Every rank's
// duration_ms is 8000 with a 1000 ms periodic-damage tick (effect index 2,
// effect 6 aura 226), so 8 ticks of 1 s each.
const (
	ConsecrationNumberOfTicks int32         = 8
	ConsecrationTickLength    time.Duration = time.Second * 1
)

// consecrationRanks: source 1.60.1.70009 client spell data. Consecration is
// baseline in Forever, not a talent -- talents/paladin.json's only
// Consecration-named entry is the Holy tree's "Consecrated Ground" (a
// synergy talent giving Holy spells bonus damage against enemies standing
// in Consecration); it does not gate Consecration itself.
//
// The cast spell carries only a periodic dummy (the third effect: 4 for
// the first targets, the ground effect's tick of 1 s). Its description
// prints the damage from a companion row per rank, damageSpellID:
// "doing ${$1280349m1*8} Holy damage over 8 sec to enemies who enter the
// area. The first $s3 enemies who enter the area will take an additional
// ${$1280349m2*8} damage". Effect 1 of the companion row is the damage a
// second to every enemy in the area (tickDamage) and effect 2 the extra a
// second the first consecrationFirstTargets enemies take
// (firstTargetsTickDamage, with a spell-power coefficient). Classic's
// 384 at rank 5 that this table once carried is the old design: a single
// target takes 12 + 27 = 39 a second, 312 over the eight seconds, and the
// area is heavy only on its first four.
var consecrationRanks = []struct {
	level                  int32
	spellID                int32
	damageSpellID          int32
	manaCost               float64
	tickDamage             float64
	firstTargetsTickDamage float64
}{
	{level: 20, spellID: 26573, damageSpellID: 1280345, manaCost: 135, tickDamage: 2, firstTargetsTickDamage: 4},
	{level: 30, spellID: 20116, damageSpellID: 1280346, manaCost: 235, tickDamage: 3, firstTargetsTickDamage: 7},
	{level: 40, spellID: 20922, damageSpellID: 1280347, manaCost: 320, tickDamage: 6, firstTargetsTickDamage: 11},
	{level: 50, spellID: 20923, damageSpellID: 1280348, manaCost: 435, tickDamage: 8, firstTargetsTickDamage: 20},
	{level: 60, spellID: 20924, damageSpellID: 1280349, manaCost: 565, tickDamage: 12, firstTargetsTickDamage: 27},
}

const (
	// consecrationFirstTargets is the cast spell's third effect: the
	// number of enemies that take the extra damage.
	consecrationFirstTargets = 4
	// consecrationFirstTargetsCoefficient is the spell-power coefficient of
	// the extra damage a second (the companion row's second effect).
	consecrationFirstTargetsCoefficient = 0.095
)

// consecrationTickDamage is what the nth enemy (from 0, in the order the
// encounter lists them) in the area takes each second.
func consecrationTickDamage(tickDamage, firstTargetsTickDamage, spellPower float64, nth int) float64 {
	if nth >= consecrationFirstTargets {
		return tickDamage
	}
	return tickDamage + firstTargetsTickDamage + consecrationFirstTargetsCoefficient*spellPower
}

func (paladin *Paladin) registerConsecration() {
	// category_cooldown_ms is 8000 for every rank, the same as the ground
	// effect's own duration -- one Consecration must finish before the next
	// can be cast.
	cd := core.Cooldown{
		Timer:    paladin.NewTimer(),
		Duration: time.Second * 8,
	}

	for i, rank := range consecrationRanks {
		rank := rank
		if paladin.Level < rank.level {
			break
		}

		consecration := paladin.RegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{SpellID: rank.spellID},
			SpellSchool: core.SpellSchoolHoly,
			DefenseType: core.DefenseTypeMagic,
			ProcMask:    core.ProcMaskSpellDamage,
			Flags:       core.SpellFlagPureDot | core.SpellFlagAPL,

			RequiredLevel: int(rank.level),
			Rank:          i + 1,

			SpellCode:      SpellCode_PaladinConsecration,
			ClassSpellMask: PaladinSpellMaskConsecration,
			ManaCost: core.ManaCostOptions{
				FlatCost: rank.manaCost,
			},
			Cast: core.CastConfig{
				DefaultCast: core.Cast{
					GCD: core.GCDDefault,
				},
				CD: cd,
			},

			DamageMultiplier: 1,
			ThreatMultiplier: 1,

			Dot: core.DotConfig{
				IsAOE: true,
				Aura: core.Aura{
					Label: "Consecration" + paladin.Label + strconv.Itoa(i+1),
				},
				NumberOfTicks: ConsecrationNumberOfTicks,
				TickLength:    ConsecrationTickLength,

				OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
					// dot.OutcomeTick, not spell.OutcomeMagicHit: the
					// latter is a Hits/Misses counter (sim/core/
					// spell_outcome.go), not a Ticks one, and every
					// other ground-AOE dot in this fork (Rain of Fire,
					// Blizzard) ticks through dot.OutcomeTick so its
					// casts show up as ticks in the metrics.
					for nth, aoeTarget := range sim.Encounter.TargetUnits {
						damage := consecrationTickDamage(rank.tickDamage, rank.firstTargetsTickDamage, dot.Spell.GetBonusDamage(aoeTarget), nth)
						dot.Spell.CalcAndDealPeriodicDamage(sim, aoeTarget, damage, dot.OutcomeTick)
					}
				},
			},

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.AOEDot().Apply(sim)
			},
		})

		// Consecration's ground effect is an AOE Dot (IsAOE: true,
		// above), so sim/core/dot.go's createDots stores its Aura on
		// spell.aoeDot rather than in spell.dots - the only two places
		// compare.go's engineDuration looks are RelatedSelfBuff and
		// Dot(target). The aura is on the CASTER (dot.go's
		// `caster.GetOrRegisterAura` for IsAOE dots), so RelatedSelfBuff
		// is the semantically correct field, not a workaround: its
		// Duration (TickLength*NumberOfTicks = 8s, set at registration
		// by newDot) already matches the client's duration_ms (8000,
		// every rank).
		consecration.RelatedSelfBuff = consecration.AOEDot().Aura
	}
}
