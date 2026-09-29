package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (hunter *Hunter) getFreezingTrapConfig(rank int, timer *core.Timer) core.SpellConfig {
	// Ids, mana cost and level match spellconst/hunter.json's own
	// spells table exactly (1499/14310/14311; the old fake id 409510
	// does not exist in the client at all). spellranks.json also lists
	// a fourth entry, 27753 (rank 3, level 60), but it carries a
	// different family_mask (missing the hunter trap flag 67108864)
	// and a 15000ms category_cooldown_ms unlike every other rank's
	// 30000ms - a stale/inert duplicate, not the live spell, so it is
	// not used here. The shared "Traps" category cooldown is 30s for
	// every rank (confirmed on Wowhead's Forever pages: "Cooldown: 30
	// seconds"), matching Immolation/Explosive Trap's fix, not the old
	// 15s.
	spellId := [4]int32{0, 1499, 14310, 14311}[rank]
	manaCost := [4]float64{0, 50, 75, 100}[rank]
	level := [4]int{0, 20, 40, 60}[rank]

	return core.SpellConfig{
		SpellCode:     SpellCode_HunterFreezingTrap,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolFrost,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskSpellDamage,
		Flags:         core.SpellFlagAPL | SpellFlagTrap,
		Rank:          rank,
		RequiredLevel: level,
		MissileSpeed:  24,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    timer,
				Duration: time.Second * 30,
			},
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true, // Hunter GCD is locked at 1.5s
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		},
	}
}

func (hunter *Hunter) registerFreezingTrapSpell(timer *core.Timer) {
	maxRank := 3
	for i := 1; i <= maxRank; i++ {
		config := hunter.getFreezingTrapConfig(i, timer)

		if config.RequiredLevel <= int(hunter.Level) {
			hunter.FreezingTrap = hunter.GetOrRegisterSpell(config)
		}
	}
}
