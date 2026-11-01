package shadow

import (
	"math"
	"strings"
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/priest"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// priestTalentStringWith returns an all-zero talent string (segments
// sized per priest.TalentTreeSizes) with the named PriestTalents fields
// set to 1. FillTalentsProto parses positionally against the proto field
// number and does not validate prerequisites or max rank (the same fact
// mage/talents_test.go documents for MageTalents), so this reaches a
// single node like Early Demise without needing a full, legal build.
func priestTalentStringWith(t *testing.T, fieldNames ...string) string {
	t.Helper()

	segments := make([]string, len(priest.TalentTreeSizes))
	for i, n := range priest.TalentTreeSizes {
		segments[i] = strings.Repeat("0", n)
	}

	for _, name := range fieldNames {
		fd := (&proto.PriestTalents{}).ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(name))
		if fd == nil {
			t.Fatalf("PriestTalents has no field named %q", name)
		}
		pos := int(fd.Number()) - 1
		treeIdx := 0
		for treeIdx < len(priest.TalentTreeSizes) && pos >= priest.TalentTreeSizes[treeIdx] {
			pos -= priest.TalentTreeSizes[treeIdx]
			treeIdx++
		}
		chars := []rune(segments[treeIdx])
		chars[pos] = '1'
		segments[treeIdx] = string(chars)
	}

	return strings.Join(segments, "-")
}

// priestTalentStringWithRank is priestTalentStringWith for a multi-rank
// (int32) talent field like Early Demise, whose node is one string digit
// 0-9 rather than one digit per rank the way a single-rank bool talent's
// is -- calling priestTalentStringWith twice for the same field only
// sets that one digit to '1' twice, not '2'.
func priestTalentStringWithRank(t *testing.T, fieldName string, rank int) string {
	t.Helper()

	str := priestTalentStringWith(t)
	fd := (&proto.PriestTalents{}).ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(fieldName))
	if fd == nil {
		t.Fatalf("PriestTalents has no field named %q", fieldName)
	}
	pos := int(fd.Number()) - 1
	treeIdx := 0
	for treeIdx < len(priest.TalentTreeSizes) && pos >= priest.TalentTreeSizes[treeIdx] {
		pos -= priest.TalentTreeSizes[treeIdx]
		treeIdx++
	}

	segments := strings.Split(str, "-")
	chars := []rune(segments[treeIdx])
	chars[pos] = rune('0' + rank)
	segments[treeIdx] = string(chars)
	return strings.Join(segments, "-")
}

// buildShadowPriestForTest stands up a level-60 shadow priest with the
// given talent string, through the shipping agent factory, with an
// encounter whose execute-phase proportions are all 1 so a single
// sim.Step() (processing PrePull's time-0 startPull action) cascades
// sim's execute phase straight to 20 -- see sim/core/sim.go's
// nextExecutePhase/advance -- putting sim.IsExecutePhase20() in the same
// state Early Demise's "target at or below 20% health" reads.
func buildShadowPriestForTest(t *testing.T, talentsStr string) (*core.Simulation, *priest.Priest) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassPriest,
			Race:          proto.Race_RaceUndead,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			Buffs:         core.FullBuffs.Player,
			TalentsString: talentsStr,
		},
		PlayerOptionsBasic,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration:             60,
			Targets:              []*proto.Target{core.DefaultTargetProtoLvl60},
			ExecuteProportion_35: 1,
			ExecuteProportion_25: 1,
			ExecuteProportion_20: 1,
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	// Step() only calls advance() (where the execute-phase cascade
	// lives) once CurrentTime actually moves past 0, and PrePull's own
	// startPull action sits at NextActionAt 0, so the first Step() does
	// not trigger it. A bounded loop up to the next few pending actions
	// (weapon swing timers, GCD ticks) is enough to reach a later time
	// and cascade to phase 20.
	for i := 0; i < 20 && !sim.IsExecutePhase20(); i++ {
		if done := sim.Step(); done {
			break
		}
	}

	agent, ok := sim.Raid.Parties[0].Players[0].(*ShadowPriest)
	if !ok {
		t.Fatal("the raid's first player is not a *ShadowPriest")
	}
	return sim, agent.Priest
}

// TestShadowWordDeathLevel60HasMaxRankAndDealsDamage rebuilds the case
// this lane exists to close: a level-60 priest must have Shadow Word:
// Death registered at its top rank (1309636, per spellconst), and
// casting it must deal shadow damage, not just exist as an entry in the
// rank slice.
func TestShadowWordDeathLevel60HasMaxRankAndDealsDamage(t *testing.T) {
	sim, built := buildShadowPriestForTest(t, priestTalentStringWith(t))

	spell := built.ShadowWordDeath[priest.ShadowWordDeathRanks]
	if spell == nil {
		t.Fatalf("level-60 priest has no Shadow Word: Death rank %d registered", priest.ShadowWordDeathRanks)
	}
	if got, want := spell.ActionID.SpellID, int32(1309636); got != want {
		t.Errorf("Shadow Word: Death spell ID = %d, want %d", got, want)
	}

	target := sim.Encounter.TargetUnits[0]
	damageBefore := spell.SpellMetrics[target.UnitIndex].TotalDamage
	spell.ApplyEffects(sim, target, spell)

	metrics := spell.SpellMetrics[target.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Shadow Word: Death landed neither a hit nor a crit (misses=%d)", metrics.Misses)
	}
	if metrics.TotalDamage <= damageBefore {
		t.Errorf("Shadow Word: Death dealt %v total damage, want more than the pre-cast %v", metrics.TotalDamage, damageBefore)
	}
}

// TestShadowWordDeathBacklashDamagesTheCaster pins Forever's backlash
// (Wowhead Forever tooltip, spell 1309636): "If your target is not killed
// by Shadow Word: Death, you take backlash damage equal to 30% of your
// maximum health", not Era's damage-dealt backlash.
func TestShadowWordDeathBacklashDamagesTheCaster(t *testing.T) {
	sim, built := buildShadowPriestForTest(t, priestTalentStringWith(t))
	spell := built.ShadowWordDeath[priest.ShadowWordDeathRanks]
	target := sim.Encounter.TargetUnits[0]

	healthBefore := built.CurrentHealth()
	spell.ApplyEffects(sim, target, spell)
	lost := healthBefore - built.CurrentHealth()
	if want := 0.30 * built.MaxHealth(); math.Abs(lost-want) > 1e-6 {
		t.Errorf("caster lost %v health, want %v (30%% of maximum health)", lost, want)
	}
}

// TestEarlyDemiseCritBonus pins the talent tooltip's two stated rates
// (15%/30%) against a target at or below 20% health, and that the bonus
// is zero both without the talent and outside that health window.
func TestEarlyDemiseCritBonus(t *testing.T) {
	sim, withTalent := buildShadowPriestForTest(t, priestTalentStringWithRank(t, "early_demise", 2))
	if withTalent.Talents.EarlyDemise != 2 {
		t.Fatalf("EarlyDemise = %d, want 2", withTalent.Talents.EarlyDemise)
	}
	if !sim.IsExecutePhase20() {
		t.Fatal("test encounter's execute proportions did not cascade to phase 20 after PrePull's first step")
	}
	if got, want := withTalent.EarlyDemiseCritBonus(sim), 30.0*core.CritRatingPerCritChance; got != want {
		t.Errorf("earlyDemiseCritBonus at rank 2 in execute phase = %v, want %v (30%% crit)", got, want)
	}

	_, withoutTalent := buildShadowPriestForTest(t, priestTalentStringWith(t))
	if got := withoutTalent.EarlyDemiseCritBonus(sim); got != 0 {
		t.Errorf("earlyDemiseCritBonus without the talent = %v, want 0", got)
	}
}
