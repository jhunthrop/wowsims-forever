package item_sets_test

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

const (
	procTrials = 40_000

	// The trials land within this fraction of the expected proc count.
	procTolerance = 0.15
)

// resource reads and empties one of a character's resources, so each
// trial sees only its own gain.
type resource struct {
	current func() float64
	spend   func(*core.Simulation, float64)
}

func resourceOf(character *core.Character, power proto.ResourceType) resource {
	id := core.ActionID{SpellID: 1}
	switch power {
	case proto.ResourceType_ResourceTypeMana:
		metrics := character.NewManaMetrics(id)
		return resource{character.CurrentMana, func(sim *core.Simulation, n float64) { character.SpendMana(sim, n, metrics) }}
	case proto.ResourceType_ResourceTypeEnergy:
		metrics := character.NewEnergyMetrics(id)
		return resource{character.CurrentEnergy, func(sim *core.Simulation, n float64) { character.SpendEnergy(sim, n, metrics) }}
	case proto.ResourceType_ResourceTypeRage:
		metrics := character.NewRageMetrics(id)
		return resource{character.CurrentRage, func(sim *core.Simulation, n float64) { character.SpendRage(sim, n, metrics) }}
	}
	panic("unhandled resource")
}

func (r resource) drain(sim *core.Simulation) {
	if current := r.current(); current > 0 {
		r.spend(sim, current)
	}
}

// gains runs trials of fire and returns how many restored the resource,
// and the amount each restored.
func (r resource) gains(w wornSet, trials int, fire func()) (events int, perEvent float64) {
	for i := 0; i < trials; i++ {
		r.drain(w.sim)
		fire()
		if gained := r.current(); gained > 0 {
			events++
			perEvent = gained
		}
	}
	return events, perEvent
}

func withinTolerance(got int, want float64) bool {
	return math.Abs(float64(got)-want) <= want*procTolerance
}

func triggerPoints(spellID int32) float64 {
	return core.MustClientSpellRow(spellID).Effects[0].Points
}

type energizeCase struct {
	name    string
	host    hostClass
	setID   int32
	pieces  int
	label   string
	event   string
	power   proto.ResourceType
	amount  float64
	chance  float64
	trigger func(w wornSet, aura *core.Aura)
}

func spellcast(w wornSet, aura *core.Aura) { w.cast(aura) }
func autoattack(w wornSet, aura *core.Aura) {
	w.swing(aura, core.ProcMaskRangedAuto)
}
func meleeHit(w wornSet, aura *core.Aura) { w.swing(aura, core.ProcMaskMeleeMHAuto) }

// The trigger spells and bonus the energize cases read their numbers from.
const (
	manaProcBonus            int32 = 450527
	manaProcAmountSpell      int32 = 450554
	energyProcAmountSpell    int32 = 27788
	warriorsResolveRageSpell int32 = 450589
	revitalizeBonus          int32 = 23863
	revitalizeAmountSpell    int32 = 23864
	rageClientTenths               = 10.0
	percentDivisor                 = 100.0

	// The client states no chance for the melee energize bonuses; the
	// model keeps the engine's earlier one proc per minute.
	meleeProcsPerMinute = 1.0
	secondsPerMinute    = 60.0
)

func energizeCases() []energizeCase {
	mana := triggerPoints(manaProcAmountSpell)
	manaProcChance := float64(core.MustClientSpellRow(manaProcBonus).ProcChance) / percentDivisor
	energy := triggerPoints(energyProcAmountSpell)
	rage := core.MustClientSpellRow(warriorsResolveRageSpell).Effects[1].Points / rageClientTenths
	perSwing := testWeaponSpeed * meleeProcsPerMinute / secondsPerMinute
	return []energizeCase{
		{"Magister's 5P", mageHost, 181, 5, "S03 - Mana Proc on Cast - Magister's Regalia (spellcast)", "spellcast", proto.ResourceType_ResourceTypeMana, mana, manaProcChance, spellcast},
		{"Devout 5P", mageHost, 182, 5, "S03 - Mana Proc on Cast - Vestments of the Devout (spellcast)", "spellcast", proto.ResourceType_ResourceTypeMana, mana, manaProcChance, spellcast},
		{"Sorcerer's 4P", mageHost, 1671, 4, "S03 - Mana Proc on Cast - Magister's Regalia (spellcast)", "spellcast", proto.ResourceType_ResourceTypeMana, mana, manaProcChance, spellcast},
		{"Virtuous 4P", mageHost, 1675, 4, "S03 - Mana Proc on Cast - Vestments of the Devout (spellcast)", "spellcast", proto.ResourceType_ResourceTypeMana, mana, manaProcChance, spellcast},
		{"Beaststalker 5P", mageHost, 186, 5, "S03 - Mana Proc on Cast - Beaststalker Armor (autoattack)", "autoattack", proto.ResourceType_ResourceTypeMana, mana, manaProcChance, autoattack},
		{"Beastmaster 4P", mageHost, 1669, 4, "S03 - Mana Proc on Cast - Beaststalker Armor (autoattack)", "autoattack", proto.ResourceType_ResourceTypeMana, mana, manaProcChance, autoattack},
		{"Shadowcraft 5P", rogueHost, 184, 5, "Rogue Armor Energize (melee hit)", "melee hit", proto.ResourceType_ResourceTypeEnergy, energy, perSwing, meleeHit},
		{"Darkmantle 4P", rogueHost, 1676, 4, "Rogue Armor Energize (melee hit)", "melee hit", proto.ResourceType_ResourceTypeEnergy, energy, perSwing, meleeHit},
		{"Stormshroud 3P", rogueHost, 142, 3, "Revitalize (melee hit)", "melee hit", proto.ResourceType_ResourceTypeEnergy, triggerPoints(revitalizeAmountSpell), float64(core.MustClientSpellRow(revitalizeBonus).ProcChance) / percentDivisor, meleeHit},
		{"Battlegear of Valor 5P", warriorHost, 189, 5, "S03 - Warrior Armor Heal Trigger - Battlegear of Valor (melee hit)", "melee hit", proto.ResourceType_ResourceTypeRage, rage, perSwing, meleeHit},
		{"Heroism 4P", warriorHost, 1778, 4, "S03 - Warrior Armor Heal Trigger - Battlegear of Valor (melee hit)", "melee hit", proto.ResourceType_ResourceTypeRage, rage, perSwing, meleeHit},
	}
}

func TestEnergizeBonusesRestoreTheClientAmount(t *testing.T) {
	for _, tc := range energizeCases() {
		t.Run(tc.name, func(t *testing.T) {
			if aura := wear(t, tc.host, tc.setID, tc.pieces-1).character.GetAura(tc.label); aura != nil {
				t.Fatalf("%d pieces already carry %q", tc.pieces-1, tc.label)
			}
			w := wear(t, tc.host, tc.setID, tc.pieces)
			aura := w.character.GetAura(tc.label)
			if aura == nil {
				t.Fatalf("%d pieces do not carry %q", tc.pieces, tc.label)
			}
			probe := resourceOf(w.character, tc.power)
			events, perEvent := probe.gains(w, procTrials, func() { tc.trigger(w, aura) })
			if perEvent != tc.amount {
				t.Errorf("each proc restores %v, want the client's %v", perEvent, tc.amount)
			}
			if want := tc.chance * procTrials; !withinTolerance(events, want) {
				t.Errorf("%d procs in %d %ss, want about %.0f", events, procTrials, tc.event, want)
			}
		})
	}
}

func buffLabelsActive(w wornSet, label string) bool {
	aura := w.character.GetAura(label)
	return aura != nil && aura.IsActive()
}

// fireUntil triggers until done reports true, and fails after the trials.
func fireUntil(t *testing.T, done func() bool, fire func()) {
	t.Helper()
	for i := 0; i < procTrials && !done(); i++ {
		fire()
	}
	if !done() {
		t.Fatalf("no proc in %d triggers", procTrials)
	}
}

type burstCase struct {
	name      string
	setID     int32
	pieces    int
	bonus     string
	buff      string
	buffSpell int32
}

func TestSpellPowerBurstsRaiseSpellPowerForTheClientDuration(t *testing.T) {
	cases := []burstCase{
		{"The Elements 5P", 187, 5, "Item - The Furious Storm Proc", "The Furious Storm", 27775},
		{"Lightforge 5P", 188, 5, "Item - Crusader's Wrath Proc - Lightforge Armor", "Crusader's Wrath", 27499},
		{"Soulforge 4P", 1673, 4, "Item - Crusader's Wrath Proc - Lightforge Armor", "Crusader's Wrath", 27499},
		{"Five Thunders 4P", 1679, 4, "Item - The Furious Storm Proc", "The Furious Storm", 27775},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := core.MustClientSpellRow(tc.buffSpell)
			castLabel, meleeLabel := tc.bonus+" (spellcast)", tc.bonus+" (melee autoattack)"
			if wear(t, mageHost, tc.setID, tc.pieces-1).character.GetAura(castLabel) != nil {
				t.Fatalf("%d pieces already carry the burst", tc.pieces-1)
			}
			for _, label := range []string{castLabel, meleeLabel} {
				w := wear(t, mageHost, tc.setID, tc.pieces)
				aura := w.character.GetAura(label)
				if aura == nil {
					t.Fatalf("%d pieces do not carry %q", tc.pieces, label)
				}
				before := w.character.GetStat(stats.SpellPower)
				fireUntil(t, func() bool { return buffLabelsActive(w, tc.buff) }, func() {
					if label == castLabel {
						w.cast(aura)
					} else {
						w.swing(aura, core.ProcMaskMeleeMHAuto)
					}
				})
				if got := w.character.GetStat(stats.SpellPower) - before; got != row.Effects[0].Points {
					t.Errorf("%q: spell power +%v, want the client's +%v", label, got, row.Effects[0].Points)
				}
				if got := w.character.GetAura(tc.buff).Duration; got != time.Duration(row.DurationMS)*time.Millisecond {
					t.Errorf("%q: lasts %v, want %dms", label, got, row.DurationMS)
				}
			}
		})
	}
}

const (
	roarOfTheCrowdSet   int32 = 1
	roarOfTheCrowdPiece       = 4
	roarOfTheCrowdAura        = "Roar of the Crowd"
	roarOfTheCrowdProc        = "Roar of the Crowd (hit)"
	roarOfTheCrowdHaste       = 1.05
)

func TestRoarOfTheCrowdGrantsFivePercentAttackSpeed(t *testing.T) {
	if wear(t, warriorHost, roarOfTheCrowdSet, roarOfTheCrowdPiece-1).character.GetAura(roarOfTheCrowdProc) != nil {
		t.Fatal("three pieces already carry Roar of the Crowd")
	}
	w := wear(t, warriorHost, roarOfTheCrowdSet, roarOfTheCrowdPiece)
	proc := w.character.GetAura(roarOfTheCrowdProc)
	if proc == nil {
		t.Fatal("four pieces do not carry Roar of the Crowd")
	}
	before := w.character.PseudoStats.MeleeSpeedMultiplier
	fireUntil(t, func() bool { return buffLabelsActive(w, roarOfTheCrowdAura) }, func() { w.swing(proc, core.ProcMaskMeleeMHAuto) })
	if got := w.character.PseudoStats.MeleeSpeedMultiplier / before; math.Abs(got-roarOfTheCrowdHaste) > 1e-9 {
		t.Errorf("attack speed x%v, want the client's x%v", got, roarOfTheCrowdHaste)
	}
	w.character.GetAura(roarOfTheCrowdAura).Deactivate(w.sim)
	if got := w.character.PseudoStats.MeleeSpeedMultiplier / before; math.Abs(got-1) > 1e-9 {
		t.Errorf("attack speed x%v after the buff fades, want x1", got)
	}
}

const (
	deviousStrikeSet    int32 = 161
	deviousStrikeBleed  int32 = 1292029
	deviousStrikeProc         = "Devious Strike (melee hit)"
	deviousStrikeChance       = 0.05
	daggerSkillPieces         = 5
	daggerSkillBonus          = 1.0
)

func TestDeviousStrikeBleedsForTheClientDamage(t *testing.T) {
	if wear(t, warriorHost, deviousStrikeSet, 3).character.GetAura(deviousStrikeProc) != nil {
		t.Fatal("three pieces already carry Devious Strike")
	}
	w := wear(t, warriorHost, deviousStrikeSet, 4)
	proc := w.character.GetAura(deviousStrikeProc)
	if proc == nil {
		t.Fatal("four pieces do not carry Devious Strike")
	}
	bleed := w.character.GetSpell(core.ActionID{SpellID: deviousStrikeBleed})
	row := core.MustClientSpellRow(deviousStrikeBleed)
	fireUntil(t, func() bool { return bleed.Dot(w.target()).IsActive() }, func() { w.swing(proc, core.ProcMaskMeleeMHAuto) })

	dot := bleed.Dot(w.target())
	wantTicks := row.DurationMS / row.Effects[0].PeriodMS
	if dot.NumberOfTicks != wantTicks {
		t.Errorf("bleed has %d ticks, want %d", dot.NumberOfTicks, wantTicks)
	}
	if dot.TickLength != time.Duration(row.Effects[0].PeriodMS)*time.Millisecond {
		t.Errorf("bleed ticks every %v, want %dms", dot.TickLength, row.Effects[0].PeriodMS)
	}
}

func TestDefiasLeatherFivePieceRaisesDaggerSkill(t *testing.T) {
	four := wear(t, warriorHost, deviousStrikeSet, daggerSkillPieces-1).character.PseudoStats.DaggersSkill
	five := wear(t, warriorHost, deviousStrikeSet, daggerSkillPieces).character.PseudoStats.DaggersSkill
	if five-four != daggerSkillBonus {
		t.Errorf("five pieces add %v dagger skill, want the client's %v", five-four, daggerSkillBonus)
	}
}

const (
	ironweaveSet               int32 = 520
	ironweavePenetration             = 5.0
	ironweavePenetrationPieces       = 3
)

func TestIronweaveThreePieceAddsSpellPenetration(t *testing.T) {
	two := wear(t, mageHost, ironweaveSet, ironweavePenetrationPieces-1).character.GetStat(stats.SpellPenetration)
	three := wear(t, mageHost, ironweaveSet, ironweavePenetrationPieces).character.GetStat(stats.SpellPenetration)
	if three-two != ironweavePenetration {
		t.Errorf("three pieces add %v spell penetration, want the client's %v", three-two, ironweavePenetration)
	}
}
