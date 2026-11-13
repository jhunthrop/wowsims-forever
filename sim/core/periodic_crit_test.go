package core

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// Forever: a channel's ticks crit only where the client marks the spell
// PERIODIC_CAN_CRIT. Blizzard is a channel without the flag, so it must
// never crit, however much crit the caster has.
//
// The assertion is on the REGISTERED spell's behaviour, not on the zero
// value of a struct the test just declared: reading back a field nobody
// set proves only that Go zeroes memory.
func TestUnflaggedChannelDoesNotCrit(t *testing.T) {
	sim, caster, target := newPeriodicCritFixture(t, 100 /* percent crit */)
	spell := caster.RegisterSpell(SpellConfig{
		// Blizzard: a channel the client does not mark PERIODIC_CAN_CRIT.
		ActionID:         ActionID{SpellID: 10187},
		SpellSchool:      SpellSchoolFrost,
		DefenseType:      DefenseTypeMagic,
		Flags:            SpellFlagChanneled,
		ProcMask:         ProcMaskSpellDamage,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		Dot: DotConfig{
			Aura:          Aura{Label: "Test Dot"},
			NumberOfTicks: 5,
			TickLength:    time.Second * 3,
			OnTick: func(sim *Simulation, target *Unit, dot *Dot) {
				dot.Spell.CalcAndDealPeriodicDamage(sim, target, 100, dot.OutcomeTick)
			},
		},
	})
	if spell.Dot(target).CanCrit {
		t.Fatal("Dot.CanCrit is true on an unflagged channel; channels follow the client flag")
	}

	spell.Dot(target).Apply(sim)
	for i := 0; i < 5; i++ {
		advanceOneTick(sim, spell.Dot(target))
	}
	// 100% crit and every tick still flat: the channel is unflagged.
	m := spell.SpellMetrics[target.UnitIndex]
	if m.CritTicks != 0 {
		t.Errorf("an unflagged channel produced %d critical ticks at 100%% crit", m.CritTicks)
	}
	if m.Ticks != 5 {
		t.Errorf("Ticks = %d, want 5", m.Ticks)
	}
	if m.TotalTickDamage != 500 {
		t.Errorf("TotalTickDamage = %v, want 500 (five flat ticks of 100)", m.TotalTickDamage)
	}
}

// A dot that does opt in gets its own multiplier, because the periodic
// figure is unknown and may differ from the direct one. An unset
// multiplier falls back to the parent spell's, so opting in without
// choosing a multiplier is not silently a 1.0.
func TestDotCritMultiplierDefaultsToTheSpells(t *testing.T) {
	const parent = 2.0
	if got := dotCritMultiplier(0, parent); got != parent {
		t.Errorf("an unset Dot.CritMultiplier gave %v, want the parent's %v", got, parent)
	}
	if got := dotCritMultiplier(1.5, parent); got != 1.5 {
		t.Errorf("an explicit Dot.CritMultiplier gave %v, want 1.5", got)
	}
}

// The missing variant: a magic dot that rolls crit on each tick rather
// than snapshotting at application. The distinction is observable -
// snapshotting makes every tick of one application crit or none - and
// which one Forever uses is what forever-measure answers.
//
// This asserts the BEHAVIOUR, not the method's existence: a bound method
// value is never nil, so `if d.OutcomeMagicCritPerTick == nil` is a test
// that cannot fail and an earlier draft of this task shipped exactly
// that. Compilation already proves the method exists.
func TestOutcomeMagicCritPerTickRollsEveryTick(t *testing.T) {
	// At 100% crit every tick crits; at 0% none does. Run both, because
	// "every tick crits" is also what a snapshot at application looks
	// like when the snapshot happens to crit.
	for _, tc := range []struct {
		crit                float64
		wantCrit, wantPlain int32
	}{
		{100, 5, 0},
		{0, 0, 5},
	} {
		sim, caster, target := newPeriodicCritFixture(t, tc.crit)
		spell := caster.RegisterSpell(SpellConfig{
			ActionID:         ActionID{SpellID: 25311},
			SpellSchool:      SpellSchoolShadow,
			DefenseType:      DefenseTypeMagic,
			ProcMask:         ProcMaskSpellDamage,
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			Dot: DotConfig{
				Aura:           Aura{Label: "Per-tick Crit Dot"},
				NumberOfTicks:  5,
				TickLength:     time.Second * 3,
				CanCrit:        true,
				CritMultiplier: 2,
				OnTick: func(sim *Simulation, target *Unit, dot *Dot) {
					dot.Spell.CalcAndDealPeriodicDamage(sim, target, 100, dot.OutcomeMagicCritPerTick)
				},
			},
		})
		spell.Dot(target).Apply(sim)
		for i := 0; i < 5; i++ {
			advanceOneTick(sim, spell.Dot(target))
		}
		m := spell.SpellMetrics[target.UnitIndex]
		if m.CritTicks != tc.wantCrit {
			t.Errorf("at %.0f%% crit: CritTicks = %d, want %d", tc.crit, m.CritTicks, tc.wantCrit)
		}
		if m.Ticks != tc.wantPlain {
			t.Errorf("at %.0f%% crit: plain Ticks = %d, want %d", tc.crit, m.Ticks, tc.wantPlain)
		}
	}
}

// A per-tick roll and a snapshot differ at an intermediate crit chance:
// the snapshot makes all five ticks agree, the per-tick roll does not.
// With a seeded sim this is deterministic, so the test can assert the
// mixture rather than a probability.
func TestOutcomeMagicCritPerTickIsNotASnapshot(t *testing.T) {
	sim, caster, target := newPeriodicCritFixture(t, 50)
	spell := caster.RegisterSpell(SpellConfig{
		ActionID:         ActionID{SpellID: 25311},
		SpellSchool:      SpellSchoolShadow,
		DefenseType:      DefenseTypeMagic,
		ProcMask:         ProcMaskSpellDamage,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		Dot: DotConfig{
			Aura:          Aura{Label: "Mixed Dot"},
			NumberOfTicks: 40,
			TickLength:    time.Second,
			CanCrit:       true,
			OnTick: func(sim *Simulation, target *Unit, dot *Dot) {
				dot.Spell.CalcAndDealPeriodicDamage(sim, target, 100, dot.OutcomeMagicCritPerTick)
			},
		},
	})
	spell.Dot(target).Apply(sim)
	for i := 0; i < 40; i++ {
		advanceOneTick(sim, spell.Dot(target))
	}
	m := spell.SpellMetrics[target.UnitIndex]
	if m.CritTicks == 0 || m.Ticks == 0 {
		t.Errorf("40 ticks at 50%% crit gave %d critical and %d plain; a per-tick roll produces both, a snapshot produces one", m.CritTicks, m.Ticks)
	}
}

// newPeriodicCritFixture builds the smallest sim that has a caster with
// a known crit chance and one target, and returns it started. crit is a
// percentage: 100 means every roll crits.
//
// Built from the same raid/encounter shape as dot_test.go's SetupFakeSim
// (a lone Elemental Shaman versus a single target). See the comment below
// on why it stops short of what SetupFakeSim does (a full NewSim, which
// finalizes the environment before returning). UseLabeledRands is set so
// each spell_outcome.go RandomFloat call site gets its own deterministic
// stream, isolating the crit roll from whatever other rolls (partial
// resist, and so on) happen to run first.
func newPeriodicCritFixture(t *testing.T, crit float64) (*Simulation, *Unit, *Unit) {
	t.Helper()

	// The environment (and so every Unit) is already finalized by the
	// time NewSim returns - env.finalize() panics if a new aura is
	// registered afterward, and every test in this file registers its
	// dot spell on caster only after this fixture returns. So this
	// builds the environment through the construct+initialize phases
	// only (the same phases NewSim runs before finalizing), leaving it
	// open for the test's RegisterSpell call. pendingFixtureFinalize (see
	// below) completes the setup once that registration has happened.
	raidProto := &proto.Raid{
		Parties: []*proto.Party{
			{
				Players: []*proto.Player{
					{
						Name:      "Caster",
						Class:     proto.Class_ClassShaman,
						Consumes:  &proto.Consumes{},
						Buffs:     &proto.IndividualBuffs{},
						Spec:      &proto.Player_ElementalShaman{},
						Equipment: &proto.EquipmentSpec{},
					},
				},
				Buffs: &proto.PartyBuffs{},
			},
		},
	}
	encounterProto := &proto.Encounter{
		Targets: []*proto.Target{
			{Name: "target", Level: 63, MobType: proto.MobType_MobTypeDemon},
		},
		Duration: 180,
	}

	env := &Environment{State: Created}
	env.construct(raidProto, encounterProto)
	raidStats := env.initialize(raidProto, encounterProto)

	sim := newSimWithEnv(env, &proto.SimOptions{
		RandomSeed:      100,
		UseLabeledRands: true,
	}, simsignals.CreateSignals())

	// Apply() (called by every test right after it registers its spell,
	// before the environment is finalized) activates the dot's aura,
	// which schedules its first tick via sim.AddPendingAction. That
	// indexes into sim.pendingActions[1:], so it needs the same sentinel
	// entry sim.Reset() would normally seed.
	sim.pendingActions = append(sim.pendingActions, sentinelPendingAction)

	caster := &sim.Raid.Parties[0].Players[0].GetCharacter().Unit

	// AddStats (unlike AddStatsDynamic) requires an unfinalized unit,
	// which is exactly what caster still is here - this is the fixture
	// the brief's own comment describes.
	caster.AddStats(stats.Stats{stats.Crit: crit * CritRatingPerCritChance})

	target := sim.Encounter.TargetUnits[0]

	pendingFixtureFinalize[sim] = func() {
		env.finalize(raidProto, encounterProto, raidStats, false)
	}
	// A test that fails before its first advanceOneTick would otherwise
	// leave its entry in the package-level map for the life of the
	// binary, holding the whole environment alive.
	t.Cleanup(func() { delete(pendingFixtureFinalize, sim) })

	return sim, caster, target
}

// pendingFixtureFinalize holds the completion step newPeriodicCritFixture
// defers past the test's RegisterSpell call; advanceOneTick runs it, once,
// on its first call for a given sim. See newPeriodicCritFixture.
var pendingFixtureFinalize = map[*Simulation]func(){}

// advanceOneTick runs the sim forward to the dot's next tick and no
// further, so a test can count ticks deterministically.
func advanceOneTick(sim *Simulation, dot *Dot) {
	if finish, ok := pendingFixtureFinalize[sim]; ok {
		delete(pendingFixtureFinalize, sim)
		finish()
		// A channelled dot asks its caster's rotation whether to
		// interrupt after each tick; finalize leaves the fixture's
		// caster one that dereferences a nil unit.
		caster := dot.Spell.Unit
		caster.Rotation = &APLRotation{unit: caster}
	}

	sim.CurrentTime = dot.NextTickAt()
	dot.TickCount++
	dot.TickOnce(sim)
}

// CanCrit's doc comment calls it the opt-in, and until this test existed
// nothing read it: OutcomeMagicCritPerTick rolled crit unconditionally,
// so whether a dot critted depended solely on which outcome function its
// OnTick happened to pass. A dot with CanCrit false that uses the
// per-tick outcome — the shape the next spec author writes when
// forever-measure reports a spell's ticks never crit — must tick flat.
func TestOutcomeMagicCritPerTickHonoursCanCrit(t *testing.T) {
	sim, caster, target := newPeriodicCritFixture(t, 100 /* percent crit */)
	spell := caster.RegisterSpell(SpellConfig{
		// Blizzard: a channel the client does not mark PERIODIC_CAN_CRIT.
		ActionID:         ActionID{SpellID: 10187},
		SpellSchool:      SpellSchoolShadow,
		DefenseType:      DefenseTypeMagic,
		Flags:            SpellFlagChanneled,
		ProcMask:         ProcMaskSpellDamage,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		Dot: DotConfig{
			Aura:          Aura{Label: "Opted-out Per-tick Dot"},
			NumberOfTicks: 5,
			TickLength:    time.Second * 3,
			// CanCrit deliberately left false while the outcome
			// function is the per-tick crit one.
			CritMultiplier: 2,
			OnTick: func(sim *Simulation, target *Unit, dot *Dot) {
				dot.Spell.CalcAndDealPeriodicDamage(sim, target, 100, dot.OutcomeMagicCritPerTick)
			},
		},
	})

	spell.Dot(target).Apply(sim)
	for i := 0; i < 5; i++ {
		advanceOneTick(sim, spell.Dot(target))
	}

	m := spell.SpellMetrics[target.UnitIndex]
	if m.CritTicks != 0 {
		t.Errorf("a dot with CanCrit false produced %d critical ticks at 100%% crit through OutcomeMagicCritPerTick", m.CritTicks)
	}
	if m.Ticks != 5 {
		t.Errorf("plain Ticks = %d, want 5", m.Ticks)
	}
	if m.TotalTickDamage != 500 {
		t.Errorf("TotalTickDamage = %v, want 500 (five flat ticks of 100)", m.TotalTickDamage)
	}
}

// Forever's periodic crit rule (owner, 2026-10-31): a periodic damage
// effect crits if it is a bleed, OR a dot that is not channelled, OR the
// client marks its spell PERIODIC_CAN_CRIT (SpellMisc Attributes_8 0x200).
// An unflagged bleed or true dot crits under the rule: on the melee table
// with the physical multiplier, or on the spell table with the spell
// multiplier. Channels follow the flag: Mind Flay crits, an unflagged
// channel does not. (Deep Wounds' and Ignite's own configs set NeverCrits,
// Blizzard's CanCrit; this test pins the rule underneath them.) A crit uses the spell's own multiplier, so talents
// that raise a spell's crit damage (Pandemic) reach its ticks.
func TestPeriodicCritRule(t *testing.T) {
	for _, tc := range []struct {
		name     string
		spellID  int32
		school   SpellSchool
		defense  DefenseType
		flags    SpellFlag
		wantCrit bool
	}{
		{"Corruption", 25311, SpellSchoolShadow, DefenseTypeMagic, 0, true},
		{"Rend", 11574, SpellSchoolPhysical, DefenseTypeMelee, 0, true},
		// The warrior's Rend declares no defense type; its ticks crit on
		// the melee table.
		{"Rend without a defense type", 11574, SpellSchoolPhysical, DefenseTypeNone, 0, true},
		{"Deep Wounds, an unflagged bleed", 12162, SpellSchoolPhysical, DefenseTypeNone, 0, true},
		{"a channelled bleed crits as a bleed", 12162, SpellSchoolPhysical, DefenseTypeNone, SpellFlagChanneled, true},
		{"Ignite, an unflagged true dot", 12654, SpellSchoolFire, DefenseTypeMagic, 0, true},
		{"Mind Flay, a flagged channel", 15407, SpellSchoolShadow, DefenseTypeMagic, SpellFlagChanneled, true},
		{"Blizzard, an unflagged channel", 10187, SpellSchoolFrost, DefenseTypeMagic, SpellFlagChanneled, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim, caster, target := newPeriodicCritFixture(t, 100)
			spell := caster.RegisterSpell(SpellConfig{
				ActionID:         ActionID{SpellID: tc.spellID},
				SpellSchool:      tc.school,
				DefenseType:      tc.defense,
				Flags:            tc.flags,
				ProcMask:         ProcMaskSpellDamage,
				DamageMultiplier: 1,
				ThreatMultiplier: 1,
				CritDamageBonus:  1,
				Dot: DotConfig{
					Aura:          Aura{Label: "Rule Dot " + tc.name},
					NumberOfTicks: 4,
					TickLength:    time.Second * 3,
					OnTick: func(sim *Simulation, target *Unit, dot *Dot) {
						dot.Spell.CalcAndDealPeriodicDamage(sim, target, 100, dot.OutcomeTick)
					},
				},
			})
			if got := spell.Dot(target).CanCrit; got != tc.wantCrit {
				t.Fatalf("CanCrit = %v, want %v", got, tc.wantCrit)
			}
			spell.Dot(target).Apply(sim)
			for i := 0; i < 4; i++ {
				advanceOneTick(sim, spell.Dot(target))
			}
			m := spell.SpellMetrics[target.UnitIndex]
			wantCritTicks := int32(0)
			if tc.wantCrit {
				wantCritTicks = 4
			}
			if m.CritTicks != wantCritTicks {
				t.Errorf("CritTicks = %d, want %d at 100%% crit", m.CritTicks, wantCritTicks)
			}
			// The spell's own multiplier for its defense type: 1.5x base
			// for spells, 2x for melee, before talents and meta gems.
			wantMult := 1.0
			if tc.wantCrit {
				defense := tc.defense
				if defense == DefenseTypeNone {
					defense = DefenseTypeMelee
				}
				wantMult = spell.critMultiplierAs(defense, caster.AttackTables[target.UnitIndex][spell.CastType])
			}
			if want := 400 * wantMult; math.Abs(m.TotalTickDamage-want) > 1e-6 {
				t.Errorf("tick damage = %v, want %v (4 ticks of 100 x %v)", m.TotalTickDamage, want, wantMult)
			}
		})
	}
}

// Forever's heals over time crit where the client marks the spell
// PERIODIC_CAN_CRIT. Every trained rank of Rejuvenation, Regrowth,
// Tranquility, Wild Growth, Renew, Riptide and Healing Stream carries the
// flag in 1.60.1.70707, and Blizzard's beta development notes of 24
// September 2026 say: "Corrected an issue and now Rejuvenation can land
// critical hits." A crit uses the healing crit chance and the spell's own
// multiplier; an unflagged heal over time (8070, an NPC Rejuvenation)
// ticks flat.
func TestHotsFollowTheClientFlag(t *testing.T) {
	for _, tc := range []struct {
		name     string
		spellID  int32
		wantCrit bool
	}{
		{"Rejuvenation rank 10, flagged", 9841, true},
		{"an NPC Rejuvenation, unflagged", 8070, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim, caster, _ := newPeriodicCritFixture(t, 100)
			spell := caster.RegisterSpell(SpellConfig{
				ActionID:         ActionID{SpellID: tc.spellID},
				SpellSchool:      SpellSchoolNature,
				DefenseType:      DefenseTypeMagic,
				ProcMask:         ProcMaskSpellHealing,
				Flags:            SpellFlagHelpful,
				DamageMultiplier: 1,
				ThreatMultiplier: 1,
				Hot: DotConfig{
					Aura:          Aura{Label: "Rule Hot " + tc.name},
					NumberOfTicks: 4,
					TickLength:    time.Second * 3,
					OnTick: func(sim *Simulation, target *Unit, dot *Dot) {
						dot.Spell.DealPeriodicHealing(sim, dot.Spell.CalcHealing(sim, target, 100, dot.OutcomeTick))
					},
				},
			})
			hot := spell.Hot(caster)
			if got := hot.CanCrit; got != tc.wantCrit {
				t.Fatalf("CanCrit = %v, want %v", got, tc.wantCrit)
			}
			hot.Apply(sim)
			for i := 0; i < 4; i++ {
				advanceOneTick(sim, hot)
			}
			m := spell.SpellMetrics[caster.UnitIndex]
			wantCritTicks := int32(0)
			if tc.wantCrit {
				wantCritTicks = 4
			}
			if m.CritTicks != wantCritTicks {
				t.Errorf("CritTicks = %d, want %d at 100%% crit", m.CritTicks, wantCritTicks)
			}
		})
	}
}
