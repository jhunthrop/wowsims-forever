package balance

import (
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// treeFields are each Druid talent tree's absolute proto field range
// (1-based, inclusive), matching talents_auto_gen.go's TalentTreeSizes
// [16, 19, 16] in client tree order Balance, Feral, Restoration (see
// proto/druid.proto's "node" comments for the field each belongs to).
var treeFields = [3][2]int{
	{1, 16},  // Balance
	{17, 35}, // Feral
	{36, 51}, // Restoration
}

// druidTalentsString builds a 3-segment talent string with every point
// zeroed except the absolute proto field numbers named, so a single
// talent can be isolated from a real build regardless of which tree it
// sits in. FillTalentsProto (sim/core/character.go) leaves any field
// past the end of its own segment at the proto zero value.
func druidTalentsString(t *testing.T, points map[int]int) string {
	t.Helper()

	segments := make([][]byte, len(treeFields))
	for i, r := range treeFields {
		segments[i] = make([]byte, r[1]-r[0]+1)
		for j := range segments[i] {
			segments[i][j] = '0'
		}
	}

	for field, value := range points {
		placed := false
		for i, r := range treeFields {
			if field >= r[0] && field <= r[1] {
				if value < 0 || value > 9 {
					t.Fatalf("field %d value %d does not fit one talent-string digit", field, value)
				}
				segments[i][field-r[0]] = byte('0' + value)
				placed = true
				break
			}
		}
		if !placed {
			t.Fatalf("field %d is outside every known tree range %v", field, treeFields)
		}
	}

	return string(segments[0]) + "-" + string(segments[1]) + "-" + string(segments[2])
}

// newBalanceDruidSimWithTalents builds a level-60, ungeared Balance druid
// with an arbitrary talents string, so a single talent can be isolated
// from a real build. Mirrors sim/druid/feral's
// newFeralDruidSimWithTalents.
func newBalanceDruidSimWithTalents(t *testing.T, talents string) (*BalanceDruid, *core.Simulation, *core.Unit) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassDruid,
			Race:               proto.Race_RaceTauren,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              core.FullBuffs.Player,
			TalentsString:      talents,
			DistanceFromTarget: 30,
		},
		PlayerOptionsAdaptive,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	target := &proto.Target{
		Level: 60,
		Stats: stats.Stats{
			stats.Armor: 100,
		}.ToFloatArray(),
	}

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{target},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	built, ok := sim.Raid.Parties[0].Players[0].(*BalanceDruid)
	if !ok {
		t.Fatal("the raid's first player is not a *BalanceDruid")
	}

	return built, sim, sim.Encounter.TargetUnits[0]
}

// TestGenesisIncreasesPeriodicDamageMultiplier covers: "Increases the
// periodic damage ... done by your spells and abilities by
// 1/2/3/4/5%." (node 104924, proto field Genesis). Only the damage half
// is modeled (no healing spells registered); scoped to Moonfire and
// Insect Swarm, this package's Balance DoTs.
func TestGenesisIncreasesPeriodicDamageMultiplier(t *testing.T) {
	const genesisField = 2
	const insectSwarmField = 9 // bool: Insect Swarm must be learned to register.

	base, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{insectSwarmField: 1}))
	mfRank := len(base.Moonfire) - 1
	baseMfMult := base.Moonfire[mfRank].PeriodicDamageMultiplierAdditive
	isRank := len(base.InsectSwarm) - 1
	baseIsMult := base.InsectSwarm[isRank].PeriodicDamageMultiplierAdditive

	talented, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{genesisField: 5, insectSwarmField: 1}))
	if got, want := talented.Moonfire[mfRank].PeriodicDamageMultiplierAdditive, baseMfMult+0.05; got != want {
		t.Errorf("5/5 Genesis Moonfire.PeriodicDamageMultiplierAdditive = %v, want %v", got, want)
	}
	if got, want := talented.InsectSwarm[isRank].PeriodicDamageMultiplierAdditive, baseIsMult+0.05; got != want {
		t.Errorf("5/5 Genesis InsectSwarm.PeriodicDamageMultiplierAdditive = %v, want %v", got, want)
	}
}

// TestNaturesMajestyAddsUnifiedCrit covers: "Increases your critical
// strike chance with spells and melee attacks by 2/4%." (node 104927,
// proto field NaturesMajesty). Forever's Crit stat already covers
// spell, melee and ranged crit together (sim/core/stats/stats.go), so
// this is a flat Stat addition.
func TestNaturesMajestyAddsUnifiedCrit(t *testing.T) {
	const naturesMajestyField = 5

	base, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, nil))
	baseCrit := base.GetStat(stats.Crit)

	talented, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{naturesMajestyField: 2}))
	if got, want := talented.GetStat(stats.Crit), baseCrit+4.0*core.CritRatingPerCritChance; got != want {
		t.Errorf("2/2 Nature's Majesty Crit stat = %v, want %v", got, want)
	}
}

// TestNaturesReachAddsHitToBalanceSpells covers: "... improves your
// chance to hit by 2/4%." on Wrath, Starfire and Moonfire (node 104929,
// proto field NaturesReach). The range half is not modeled.
func TestNaturesReachAddsHitToBalanceSpells(t *testing.T) {
	const naturesReachField = 6

	base, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, nil))
	wrathRank := len(base.Wrath) - 1
	starfireRank := len(base.Starfire) - 1
	moonfireRank := len(base.Moonfire) - 1
	baseWrathHit := base.Wrath[wrathRank].BonusHitRating
	baseStarfireHit := base.Starfire[starfireRank].BonusHitRating
	baseMoonfireHit := base.Moonfire[moonfireRank].BonusHitRating

	talented, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{naturesReachField: 2}))
	bonus := 4.0 * core.HitRatingPerHitChance
	if got, want := talented.Wrath[wrathRank].BonusHitRating, baseWrathHit+bonus; got != want {
		t.Errorf("2/2 Nature's Reach Wrath.BonusHitRating = %v, want %v", got, want)
	}
	if got, want := talented.Starfire[starfireRank].BonusHitRating, baseStarfireHit+bonus; got != want {
		t.Errorf("2/2 Nature's Reach Starfire.BonusHitRating = %v, want %v", got, want)
	}
	if got, want := talented.Moonfire[moonfireRank].BonusHitRating, baseMoonfireHit+bonus; got != want {
		t.Errorf("2/2 Nature's Reach Moonfire.BonusHitRating = %v, want %v", got, want)
	}
}

// TestNaturesSplendorAddsOneDotTick covers: "Increases the duration of
// your Moonfire ... spell by 3 sec ... and your Insect Swarm spell by 2
// sec." (node 104928, bool, proto field NaturesSplendor). Moonfire ticks
// every 3 sec and Insect Swarm every 2 sec, so +3/+2 sec is exactly +1
// tick on each. The Rejuvenation/Regrowth half is not modeled.
func TestNaturesSplendorAddsOneDotTick(t *testing.T) {
	const naturesSplendorField = 8
	const insectSwarmField = 9 // bool: Insect Swarm must be learned to register.

	base, _, baseTarget := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{insectSwarmField: 1}))
	mfRank := len(base.Moonfire) - 1
	isRank := len(base.InsectSwarm) - 1
	baseMfTicks := base.Moonfire[mfRank].Dot(baseTarget).NumberOfTicks
	baseIsTicks := base.InsectSwarm[isRank].Dot(baseTarget).NumberOfTicks

	talented, _, target := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{naturesSplendorField: 1, insectSwarmField: 1}))
	if got, want := talented.Moonfire[mfRank].Dot(target).NumberOfTicks, baseMfTicks+1; got != want {
		t.Errorf("Nature's Splendor Moonfire ticks = %d, want %d", got, want)
	}
	if got, want := talented.InsectSwarm[isRank].Dot(target).NumberOfTicks, baseIsTicks+1; got != want {
		t.Errorf("Nature's Splendor Insect Swarm ticks = %d, want %d", got, want)
	}
}

// TestEclipseDiscountsStarfireCastTimeAfterWrath covers: "Your Wrath
// spell reduces the cast time of your next 2 Starfire spells by 0.5 sec
// [at 3/3]. Stores up to 4 charges. Lasts 15 sec." (node 104935, proto
// field Eclipse).
func TestEclipseDiscountsStarfireCastTimeAfterWrath(t *testing.T) {
	const eclipseField = 14

	built, sim, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{eclipseField: 3}))
	wrathRank := len(built.Wrath) - 1
	starfireRank := len(built.Starfire) - 1
	baseCastTime := built.Starfire[starfireRank].DefaultCast.CastTime

	// OnCastComplete (Eclipse's trigger) is invoked from the Spell.Cast
	// completion routine (sim/core/cast.go), not from ApplyEffects, so
	// it is driven directly here the same way
	// sim/core/racials_test.go's TestOnCastComplete does.
	built.OnCastComplete(sim, built.Wrath[wrathRank].Spell)
	if got, want := built.Starfire[starfireRank].DefaultCast.CastTime, baseCastTime-500*time.Millisecond; got != want {
		t.Errorf("Starfire cast time after one Wrath cast with 3/3 Eclipse = %v, want %v", got, want)
	}

	// The one Wrath cast above banked 2 charges; spending both with two
	// Starfire casts and no further Wrath in between must find the
	// discount gone.
	built.OnCastComplete(sim, built.Starfire[starfireRank].Spell)
	built.OnCastComplete(sim, built.Starfire[starfireRank].Spell)
	if got, want := built.Starfire[starfireRank].DefaultCast.CastTime, baseCastTime; got != want {
		t.Errorf("Starfire cast time after spending both Eclipse charges = %v, want the untalented %v", got, want)
	}
}

// TestNaturalistIncreasesAllDamageDealt covers: "... increases all
// damage you deal by 1/2/3/4/5%." (node 104922, proto field Naturalist -
// in the Restoration tree, field 38, even though its damage half reads
// on every spell). The Healing Touch cast-time half is not modeled.
func TestNaturalistIncreasesAllDamageDealt(t *testing.T) {
	const naturalistField = 38

	base, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, nil))
	baseMult := base.PseudoStats.DamageDealtMultiplier

	talented, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{naturalistField: 5}))
	if got, want := talented.PseudoStats.DamageDealtMultiplier, baseMult*1.05; got != want {
		t.Errorf("5/5 Naturalist DamageDealtMultiplier = %v, want %v", got, want)
	}
}

// TestLivingSpiritIncreasesSpirit covers: "Increases your Spirit by
// 5/10/15%." (node 104911, proto field LivingSpirit, Restoration tree
// field 48).
func TestLivingSpiritIncreasesSpirit(t *testing.T) {
	const livingSpiritField = 48

	base, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, nil))
	baseSpirit := base.GetStat(stats.Spirit)

	talented, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{livingSpiritField: 3}))
	if got, want := talented.GetStat(stats.Spirit), baseSpirit*1.15; got != want {
		t.Errorf("3/3 Living Spirit Spirit stat = %v, want %v", got, want)
	}
}

// TestNaturesSwiftnessMakesWrathInstant covers: "When activated, your
// next Nature spell becomes an instant cast spell." (node 104921, bool,
// proto field NaturesSwiftness, Restoration tree field 47). Wrath is
// this package's only Nature-school damage spell.
func TestNaturesSwiftnessMakesWrathInstant(t *testing.T) {
	const naturesSwiftnessField = 47

	built, sim, target := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{naturesSwiftnessField: 1}))
	if built.NaturesSwiftness == nil {
		t.Fatal("1/1 Nature's Swiftness druid has no Nature's Swiftness spell registered")
	}
	wrathRank := len(built.Wrath) - 1
	baseCastTime := built.Wrath[wrathRank].DefaultCast.CastTime
	if baseCastTime <= 0 {
		t.Fatalf("precondition: untalented Wrath cast time = %v, want > 0", baseCastTime)
	}
	// CastTimeMultiplier is a multiplicative factor (final cast time =
	// base * CastTimeMultiplier), not a delta, and RegisterSpell
	// defaults it to 1 - see applyNaturesGrace's identical -1/+1 above.
	baseMultiplier := built.Wrath[wrathRank].CastTimeMultiplier

	// Drive ApplyEffects directly (the real code a Cast would run), the
	// same way sim/druid/feral/claw_level_test.go does, so the
	// assertion isn't gated on GCD/resource bookkeeping that's out of
	// scope here.
	built.NaturesSwiftness.ApplyEffects(sim, target, built.NaturesSwiftness.Spell)
	if got, want := built.Wrath[wrathRank].CastTimeMultiplier, baseMultiplier-1; got != want {
		t.Errorf("Wrath.CastTimeMultiplier after Nature's Swiftness = %v, want %v (0 = instant)", got, want)
	}

	// OnCastComplete (which consumes the buff) is invoked from the
	// Spell.Cast completion routine (sim/core/cast.go), not from
	// ApplyEffects, so it is driven directly here the same way
	// sim/core/racials_test.go's TestOnCastComplete does.
	built.OnCastComplete(sim, built.Wrath[wrathRank].Spell)
	if got, want := built.Wrath[wrathRank].CastTimeMultiplier, baseMultiplier; got != want {
		t.Errorf("Wrath.CastTimeMultiplier after the Nature's Swiftness Wrath cast completed = %v, want %v (consumed, back to baseline)", got, want)
	}
}
