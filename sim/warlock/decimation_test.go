package warlock

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// DecimationRank2TalentsString sets only Decimation rank 2 (field 12 of
// the 19-node Demonology tree), the max-rank numbers the talent text and
// wowhead both give: -90% Soul Fire cooldown passively, and a 10 s
// window (-40% cast time, -40% cost) after Shadow Bolt or Searing Pain
// lands.
const DecimationRank2TalentsString = "-000000000002"

// decimationWithBaneTalentsString adds Bane 5/5 (Destruction field 3), whose
// flat -2 s Soul Fire cast time Decimation's -40% then scales.
const decimationWithBaneTalentsString = DecimationRank2TalentsString + "-005"

func newDecimationTestWarlock(t *testing.T) (*core.Simulation, *Warlock, *core.Unit) {
	t.Helper()
	return newDecimationTestWarlockWith(t, DecimationRank2TalentsString, 1)
}

// newDecimationTestWarlockWith builds the warlock against a dummy with no
// health bar whose last execute35 of the fight is below 35% health, stepped
// once so the encounter's execute phase is current.
func newDecimationTestWarlockWith(t *testing.T, talents string, execute35 float64) (*core.Simulation, *Warlock, *core.Unit) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassWarlock,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			Buffs:         core.FullBuffs.Player,
			TalentsString: talents,
		},
		&proto.Player_Warlock{
			Warlock: &proto.Warlock{
				Options: &proto.WarlockOptions{
					Armor:       proto.WarlockOptions_NoArmor,
					Summon:      proto.WarlockOptions_NoSummon,
					WeaponImbue: proto.WarlockOptions_NoWeaponImbue,
				},
			},
		},
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration:             60,
			ExecuteProportion_35: execute35,
			Targets:              []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	for i := 0; i < 10 && sim.CurrentTime <= 0; i++ {
		sim.Step()
	}

	agent, ok := sim.Raid.Parties[0].Players[0].(WarlockAgent)
	if !ok {
		t.Fatal("the raid's first player is not a warlock agent")
	}
	built := agent.GetWarlock()
	target := sim.Encounter.TargetUnits[0]
	return sim, built, target
}

// TestDecimationPassiveReducesSoulFireCooldown checks the always-on half
// of the talent: rank 2 shortens Soul Fire's 1-minute cooldown by 90%
// regardless of any proc.
func TestDecimationPassiveReducesSoulFireCooldown(t *testing.T) {
	_, built, _ := newDecimationTestWarlock(t)

	if len(built.SoulFire) == 0 {
		t.Fatal("level-60 warlock has no Soul Fire registered")
	}
	const tolerance = time.Millisecond
	for _, spell := range built.SoulFire {
		want := time.Minute / 10 // 90% reduction of the base 1-minute CD.
		got := spell.CD.Duration
		diff := got - want
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			t.Errorf("Soul Fire (rank %d) CD.Duration = %v, want %v (90%% reduction)", spell.Rank, got, want)
		}
	}
}

// TestDecimationProcReducesSoulFireCastTimeAndCost casts Shadow Bolt in the
// encounter's below-35% window (the dummy has no health bar, so the window
// is the encounter's execute phase) and checks the 10 s Decimation buff
// comes up and takes 40% off Soul Fire's cast time, then reverts once it
// expires. The buff's second effect is aura 256 (no reagent use): it frees
// the Soul Shard and leaves the mana cost alone.
func TestDecimationProcReducesSoulFireCastTimeAndCost(t *testing.T) {
	sim, built, target := newDecimationTestWarlock(t)

	if len(built.SoulFire) == 0 || len(built.ShadowBolt) == 0 {
		t.Fatal("level-60 warlock is missing Soul Fire or Shadow Bolt")
	}
	soulFire := built.SoulFire[len(built.SoulFire)-1]
	shadowBolt := built.ShadowBolt[len(built.ShadowBolt)-1]

	baseCastTime := soulFire.DefaultCast.CastTime
	baseCostMultiplier := soulFire.Cost.Multiplier

	shadowBolt.ApplyEffects(sim, target, shadowBolt)

	// Shadow Bolt's damage (and so its OnSpellHitDealt callback, which is
	// what activates the Decimation buff) is deferred behind the
	// missile's travel time (spell.WaitTravelTime), same as Aimed Shot;
	// step the sim until that pending action fires.
	aura := built.GetAura("Decimation")
	for i := 0; i < 200 && !aura.IsActive(); i++ {
		if done := sim.Step(); done {
			break
		}
	}
	if !aura.IsActive() {
		t.Fatal("Shadow Bolt landed on an eligible target but the Decimation buff did not activate")
	}

	wantCastTime := time.Duration(float64(baseCastTime) * 0.60)
	if got := soulFire.CastTime(); got != wantCastTime {
		t.Errorf("Soul Fire cast time during Decimation = %v, want %v", got, wantCastTime)
	}
	if got := soulFire.Cost.Multiplier; got != baseCostMultiplier {
		t.Errorf("Soul Fire cost multiplier during Decimation = %d, want %d (the buff frees the shard, not mana)", got, baseCostMultiplier)
	}

	// Advance past the 10 s window and confirm both revert.
	for i := 0; i < 2000; i++ {
		if !aura.IsActive() {
			break
		}
		if done := sim.Step(); done {
			break
		}
	}
	if aura.IsActive() {
		t.Fatal("Decimation aura is still active after its 10 s duration should have elapsed")
	}
	if got := soulFire.CastTime(); got != baseCastTime {
		t.Errorf("Soul Fire cast time after Decimation expired = %v, want %v", got, baseCastTime)
	}
	if got := soulFire.Cost.Multiplier; got != baseCostMultiplier {
		t.Errorf("Soul Fire cost multiplier after Decimation expired = %d, want %d", got, baseCostMultiplier)
	}
}

// TestDecimationDamageBonusBelowHealthThreshold checks the talent's other
// half: Shadow Bolt (and Searing Pain, same mechanism) deals 6% more
// damage (rank 2) against a target below 35% health, and no bonus at
// all above it.
func TestDecimationDamageBonusBelowHealthThreshold(t *testing.T) {
	sim, built, target := newDecimationTestWarlock(t)
	if got, want := built.decimationDamageMultiplier(sim, target), 1.06; got != want {
		t.Errorf("decimationDamageMultiplier(no health bar, execute phase) = %v, want %v", got, want)
	}

	// The default level-60 target proto tracks no Health stat (a pure
	// tank-a-dummy encounter has no need to), so decimationDamageMultiplier
	// would otherwise see a 0/0 (NaN) health percent; give it one here so
	// RemoveHealth below actually moves CurrentHealthPercent.
	target.AddStatDynamic(sim, stats.Health, 1000)
	target.EnableHealthBar()
	metrics := target.NewHealthMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionDamageTaken})
	maxHealth := target.MaxHealth()
	target.GainHealth(sim, maxHealth, metrics)

	target.RemoveHealth(sim, maxHealth*0.50)
	if got, want := built.decimationDamageMultiplier(sim, target), 1.0; got != want {
		t.Errorf("decimationDamageMultiplier(50%% health) = %v, want %v (no bonus above 35%%)", got, want)
	}

	target.GainHealth(sim, maxHealth, metrics)
	target.RemoveHealth(sim, maxHealth*0.80)
	if got, want := built.decimationDamageMultiplier(sim, target), 1.06; got != want {
		t.Errorf("decimationDamageMultiplier(20%% health) = %v, want %v", got, want)
	}
}

// TestDecimationNeedsTheTargetBelow35Percent pins the talent text's gate:
// "When you cast Shadow Bolt or Searing Pain on an enemy below 35% health".
// A dummy with no health bar is below 35% only in the encounter's execute
// window, so outside it neither the buff nor the damage bonus applies.
func TestDecimationNeedsTheTargetBelow35Percent(t *testing.T) {
	sim, built, target := newDecimationTestWarlockWith(t, DecimationRank2TalentsString, 0.2)
	if sim.IsExecutePhase35() {
		t.Fatal("the fight opens in the execute phase")
	}
	if got := built.decimationDamageMultiplier(sim, target); got != 1 {
		t.Errorf("decimationDamageMultiplier above 35%% = %v, want 1", got)
	}
	shadowBolt := built.ShadowBolt[len(built.ShadowBolt)-1]
	shadowBolt.ApplyEffects(sim, target, shadowBolt)
	for i := 0; i < 200 && shadowBolt.SpellMetrics[target.UnitIndex].TotalDamage == 0; i++ {
		sim.Step()
	}
	if built.GetAura("Decimation").IsActive() {
		t.Error("Shadow Bolt on a target above 35% health granted Decimation")
	}
}

// TestDecimationScalesTheBaneShortenedCastTime pins the order of the two
// modifiers: Bane takes a flat 2 s off Soul Fire's 6 s, and Decimation's
// aura 108 (misc 10, a percent modifier) takes 40% of what is left, 2.4 s.
func TestDecimationScalesTheBaneShortenedCastTime(t *testing.T) {
	sim, built, target := newDecimationTestWarlockWith(t, decimationWithBaneTalentsString, 1)
	soulFire := built.SoulFire[len(built.SoulFire)-1]
	if got, want := soulFire.DefaultCast.CastTime, 4*time.Second; got != want {
		t.Fatalf("Soul Fire with Bane 5/5 casts in %v, want %v", got, want)
	}
	built.GetAura("Decimation").Activate(sim)
	_ = target
	if got, want := soulFire.CastTime(), 2400*time.Millisecond; got != want {
		t.Errorf("Soul Fire with Bane 5/5 during Decimation casts in %v, want %v", got, want)
	}
}
