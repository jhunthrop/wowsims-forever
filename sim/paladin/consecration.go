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
// tickDamage is Classic's per-rank Consecration damage over 8 s divided by
// its 8 ticks (64/120/192/280/384 at ranks 1-5). The Forever client's
// periodic effect (index 2) carries only a "periodic dummy" of 4-5 per
// second with a server-side script (wowhead.com/forever/spell=20924 shows
// the same), so the client does not state the damage at all, and the one
// number it does carry is not it. Classic's values are the best-known
// answer until Forever's own numbers are measured in a log.
var consecrationRanks = []struct {
	level      int32
	spellID    int32
	manaCost   float64
	tickDamage float64
}{
	{level: 20, spellID: 26573, manaCost: 135, tickDamage: 64 / 8},
	{level: 30, spellID: 20116, manaCost: 235, tickDamage: 120 / 8},
	{level: 40, spellID: 20922, manaCost: 320, tickDamage: 192 / 8},
	{level: 50, spellID: 20923, manaCost: 435, tickDamage: 280 / 8},
	{level: 60, spellID: 20924, manaCost: 565, tickDamage: 384 / 8},
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

				OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
					dot.Snapshot(target, rank.tickDamage, isRollover)
				},
				OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
					// dot.OutcomeTick, not spell.OutcomeMagicHit: the
					// latter is a Hits/Misses counter (sim/core/
					// spell_outcome.go), not a Ticks one, and every
					// other ground-AOE dot in this fork (Rain of Fire,
					// Blizzard) ticks through dot.OutcomeTick so its
					// casts show up as ticks in the metrics.
					for _, aoeTarget := range sim.Encounter.TargetUnits {
						dot.CalcAndDealPeriodicSnapshotDamage(sim, aoeTarget, dot.OutcomeTick)
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
