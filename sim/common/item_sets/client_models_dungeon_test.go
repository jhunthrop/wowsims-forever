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
	necropileSet    int32 = 122
	necropilePieces       = 3
	necropileBonus  int32 = 1299737
	necropileLeech  int32 = 1299735
	necropileProc         = "Necropile Drain (harmful spellcast)"
	leechTrials           = 400

	// A landed leech averages within this fraction of its expected damage.
	leechTolerance = 0.1
)

// castPastCooldown delivers a harmful cast after the bonus's internal
// cooldown has passed, so every cast may proc.
func castPastCooldown(w wornSet, proc *core.Aura, cooldown time.Duration) {
	w.sim.CurrentTime += cooldown
	w.cast(proc)
}

func TestNecropileDrainLeechesTheClientAmountOnEveryHarmfulCast(t *testing.T) {
	if wear(t, mageHost, necropileSet, necropilePieces-1).character.GetAura(necropileProc) != nil {
		t.Fatal("two pieces already carry Necropile Drain")
	}
	w := wear(t, mageHost, necropileSet, necropilePieces)
	proc := w.character.GetAura(necropileProc)
	if proc == nil {
		t.Fatal("three pieces do not carry Necropile Drain")
	}
	leech := w.character.GetSpell(core.ActionID{SpellID: necropileLeech})
	cooldown := time.Duration(core.MustClientSpellRow(necropileBonus).InternalCooldownMS) * time.Millisecond
	amount := core.MustClientSpellRow(necropileLeech).Effects[0].Points

	w.cast(proc)
	w.cast(proc)
	metrics := func() *core.SpellMetrics { return &leech.SpellMetrics[w.target().UnitIndex] }
	if metrics().Casts != 1 {
		t.Errorf("two casts inside the %v internal cooldown leeched %d times, want 1", cooldown, metrics().Casts)
	}
	for i := 1; i < leechTrials; i++ {
		castPastCooldown(w, proc, cooldown)
	}
	if metrics().Casts != leechTrials {
		t.Fatalf("%d spaced casts leeched %d times, want every one", leechTrials, metrics().Casts)
	}
	landed := float64(metrics().Hits + metrics().ResistedHits)
	average := metrics().TotalDamage / landed
	expected := leech.CalcDamage(w.sim, w.target(), amount, leech.OutcomeExpectedMagicAlwaysHit).Damage
	if math.Abs(average-expected) > expected*leechTolerance {
		t.Errorf("a landed leech averages %.1f, want about %.1f (the client's %v after the raid's multipliers)", average, expected, amount)
	}
}

const (
	bloodmailSet    int32 = 123
	bloodmailPieces       = 3
	bloodmailBleed  int32 = 1299739
	bloodmailProc         = "Bloodmail (melee hit)"
)

func TestBloodmailBleedsForTheClientTicks(t *testing.T) {
	if wear(t, warriorHost, bloodmailSet, bloodmailPieces-1).character.GetAura(bloodmailProc) != nil {
		t.Fatal("two pieces already carry Bloodmail")
	}
	w := wear(t, warriorHost, bloodmailSet, bloodmailPieces)
	proc := w.character.GetAura(bloodmailProc)
	if proc == nil {
		t.Fatal("three pieces do not carry Bloodmail")
	}
	bleed := w.character.GetSpell(core.ActionID{SpellID: bloodmailBleed})
	fireUntil(t, func() bool { return bleed.Dot(w.target()).IsActive() }, func() { w.swing(proc, core.ProcMaskMeleeMHAuto) })

	row := core.MustClientSpellRow(bloodmailBleed)
	dot := bleed.Dot(w.target())
	if want := row.DurationMS / row.Effects[0].PeriodMS; dot.NumberOfTicks != want {
		t.Errorf("bleed has %d ticks, want %d", dot.NumberOfTicks, want)
	}
	if want := time.Duration(row.Effects[0].PeriodMS) * time.Millisecond; dot.TickLength != want {
		t.Errorf("bleed ticks every %v, want %v", dot.TickLength, want)
	}
}

const (
	deathboneSet    int32 = 124
	deathbonePieces       = 3
	deathboneBonus  int32 = 1299743
	deathboneBuff   int32 = 1299746
	deathboneProc         = "Deathbone Surge (damage taken)"
	deathboneAura         = "Deathbone Surge"
	surgeLockout          = 500
)

func TestDeathboneSurgeRaisesSpellPowerWhenStruckThenWaitsItsCooldown(t *testing.T) {
	if wear(t, warriorHost, deathboneSet, deathbonePieces-1).character.GetAura(deathboneProc) != nil {
		t.Fatal("two pieces already carry Deathbone Surge")
	}
	w := wear(t, warriorHost, deathboneSet, deathbonePieces)
	proc := w.character.GetAura(deathboneProc)
	if proc == nil {
		t.Fatal("three pieces do not carry Deathbone Surge")
	}
	buffRow := core.MustClientSpellRow(deathboneBuff)
	spellPower := w.character.GetStat(stats.SpellPower)
	fireUntil(t, func() bool { return buffLabelsActive(w, deathboneAura) }, func() { w.struck(proc) })

	buff := w.character.GetAura(deathboneAura)
	if got := buff.Duration; got != time.Duration(buffRow.DurationMS)*time.Millisecond {
		t.Errorf("surge lasts %v, want %dms", got, buffRow.DurationMS)
	}
	for _, effect := range buffRow.Effects {
		if effect.Points != buffRow.Effects[0].Points {
			t.Fatalf("surge effects differ: %v", buffRow.Effects)
		}
	}
	if got := w.character.GetStat(stats.SpellPower) - spellPower; got != buffRow.Effects[0].Points {
		t.Errorf("spell power +%v, want the client's +%v", got, buffRow.Effects[0].Points)
	}

	buff.Deactivate(w.sim)
	for i := 0; i < surgeLockout; i++ {
		w.struck(proc)
	}
	if buffLabelsActive(w, deathboneAura) {
		t.Fatal("surge procs again inside its internal cooldown")
	}
	w.sim.CurrentTime += time.Duration(core.MustClientSpellRow(deathboneBonus).InternalCooldownMS) * time.Millisecond
	fireUntil(t, func() bool { return buffLabelsActive(w, deathboneAura) }, func() { w.struck(proc) })
}

// The tooltips of the two combat-rating rows: "Increased Hit Chance 00.5"
// and "Increased Critical 1.5 - All".
func TestRatingBonusesAddTheStatedPercent(t *testing.T) {
	cases := []struct {
		name    string
		set     int32
		pieces  int
		stat    stats.Stat
		percent float64
	}{
		{"Necropile 2P hit", necropileSet, 2, stats.Hit, 0.5},
		{"Bloodmail 5P crit", bloodmailSet, 5, stats.Crit, 1.5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fewer := wear(t, warriorHost, tc.set, tc.pieces-1).character.GetStat(tc.stat)
			worn := wear(t, warriorHost, tc.set, tc.pieces).character.GetStat(tc.stat)
			if got := worn - fewer; math.Abs(got-tc.percent) > 1e-9 {
				t.Errorf("%d pieces add %v, want %v", tc.pieces, got, tc.percent)
			}
		})
	}
}

type meleeDamageCase struct {
	name   string
	setID  int32
	pieces int
	bonus  int32
	bolt   int32
	label  string
}

const (
	meleeDamageSwings    = 20_000
	meleeDamageTolerance = 0.1
)

func meleeDamageCases() []meleeDamageCase {
	return []meleeDamageCase{
		{"Volcanic Armor 3P", 141, 3, 9233, 9057, "Firebolt (melee hit)"},
		{"Stormshroud Armor 2P", 142, 2, 18979, 18980, "Lightning (melee hit)"},
	}
}

func TestMeleeDamageProcsDealTheClientDamageAtTheClientChance(t *testing.T) {
	for _, tc := range meleeDamageCases() {
		t.Run(tc.name, func(t *testing.T) {
			if wear(t, warriorHost, tc.setID, tc.pieces-1).character.GetAura(tc.label) != nil {
				t.Fatalf("%d pieces already carry %q", tc.pieces-1, tc.label)
			}
			w := wear(t, warriorHost, tc.setID, tc.pieces)
			proc := w.character.GetAura(tc.label)
			if proc == nil {
				t.Fatalf("%d pieces do not carry %q", tc.pieces, tc.label)
			}
			for i := 0; i < meleeDamageSwings; i++ {
				w.swing(proc, core.ProcMaskMeleeMHAuto)
			}
			bolt := w.character.GetSpell(core.ActionID{SpellID: tc.bolt})
			metrics := bolt.SpellMetrics[w.target().UnitIndex]
			chance := float64(core.MustClientSpellRow(tc.bonus).ProcChance) / percentDivisor
			if want := chance * meleeDamageSwings; !withinTolerance(int(metrics.Casts), want) {
				t.Errorf("%d procs in %d hits, want about %.0f", metrics.Casts, meleeDamageSwings, want)
			}
			amount := core.MustClientSpellRow(tc.bolt).Effects[0].Points
			expected := bolt.CalcDamage(w.sim, w.target(), amount, bolt.OutcomeExpectedMagicHitAndCrit).Damage
			average := metrics.TotalDamage / float64(metrics.Casts)
			if math.Abs(average-expected) > expected*meleeDamageTolerance {
				t.Errorf("a proc averages %.1f damage, want about %.1f (the client's %v)", average, expected, amount)
			}
		})
	}
}

const (
	undeadSlayingSet    int32 = 533
	undeadSlayingPieces       = 3
	undeadSlayingBonus  int32 = 29068
)

func TestUndeadSlayingRaisesDamageOnlyAgainstUndead(t *testing.T) {
	multiplierOn := func(pieces int, target int) float64 {
		w := wear(t, warriorHost, undeadSlayingSet, pieces, core.DefaultTargetProtoLvl60, undeadTarget())
		unit := w.sim.Encounter.AllTargetUnits[target]
		return w.character.AttackTables[unit.UnitIndex][proto.CastType_CastTypeMainHand].DamageDealtMultiplier
	}
	const humanoid, undead = 0, 1
	want := 1 + core.MustClientSpellRow(undeadSlayingBonus).Effects[0].Points/percentDivisor
	if got := multiplierOn(undeadSlayingPieces, undead) / multiplierOn(undeadSlayingPieces-1, undead); math.Abs(got-want) > 1e-9 {
		t.Errorf("damage against undead x%v, want the client's x%v", got, want)
	}
	if got := multiplierOn(undeadSlayingPieces, humanoid) / multiplierOn(undeadSlayingPieces-1, humanoid); math.Abs(got-1) > 1e-9 {
		t.Errorf("damage against a humanoid x%v, want x1", got)
	}
}
