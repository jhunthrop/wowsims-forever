package core

import "time"

// HarmfulSpellProc is a "chance on harmful spell cast" equip effect: Forever
// items such as Kindling Stave's Ignition and Necropile Raiment's Necropile
// Drain. The client rows give these ProcChance 100 with no
// SpellProcsPerMinute row and a short ProcCategoryRecovery, while the text
// says "a chance"; the beta logs show such rows proccing at a few percent
// (see the round-8 warlock report), so the rate is the server's and the
// item states it as a rate per minute.
type HarmfulSpellProc struct {
	Name     string
	ActionID ActionID
	PPM      float64
	// ICD is the client's ProcCategoryRecovery. It stops one cast's several
	// hit results from proccing twice, so it is kept per spell: the engine
	// can land a cast-time spell in the same instant as the next instant
	// cast starts.
	ICD     time.Duration
	Handler func(sim *Simulation, target *Unit)
}

// UnstatedProcsPerMinute is the rate of a "chance on" bonus whose client
// row states none (ProcChance 100, no SpellProcsPerMinute row): one a
// minute. The beta logs put Forever's Dream Venom (1291807) at 10 in 358
// auto attacks and Rotmending (1321587) at 10 in about 258 heals, both near
// one a minute; it is still an estimate from two small samples.
const UnstatedProcsPerMinute = 1.0

// SpellPPMChance is the chance a spell cast procs an effect that fires ppm
// times a minute: ppm times the cast time, an instant counting as the GCD.
func SpellPPMChance(ppm float64, spell *Spell) float64 {
	castTime := max(spell.CurCast.CastTime, GCDDefault)
	return min(1, ppm*castTime.Minutes())
}

// MakeHarmfulSpellProcAura registers the proc on unit. A harmful cast is a
// damage spell's hit result on an enemy, landed or missed; a self-cast and
// a periodic tick are not casts.
func MakeHarmfulSpellProcAura(unit *Unit, config HarmfulSpellProc) *Aura {
	lastProc := make(map[*Spell]time.Duration)
	return MakePermanent(unit.RegisterAura(Aura{
		Label:    config.Name,
		ActionID: config.ActionID,
		// Each iteration starts with no recovery running, so a run's mean
		// does not depend on how its iterations are split between sims.
		OnReset: func(*Aura, *Simulation) { clear(lastProc) },
		OnSpellHitDealt: func(aura *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			if !spell.ProcMask.Matches(ProcMaskSpellDamage) || result.Target == nil || result.Target.Type != EnemyUnit {
				return
			}
			if last, ok := lastProc[spell]; ok && sim.CurrentTime >= last && sim.CurrentTime < last+config.ICD {
				return
			}
			if sim.RandomFloat(config.Name) >= SpellPPMChance(config.PPM, spell) {
				return
			}
			lastProc[spell] = sim.CurrentTime
			config.Handler(sim, result.Target)
		},
	}))
}
