package hunter

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// publishedSurvivalTalents is the band-60 Survival build the site publishes
// (data/builds/1.60.1.70707/bis/hunter-survival.json), Resourcefulness 2/2.
const publishedSurvivalTalents = "0000000000000000-3250050010000000-500230231051222151"

func buildPublishedSurvivalHunter(t *testing.T) (*core.Simulation, *Hunter) {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassHunter,
		Race:               proto.Race_RaceOrc,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{},
		TalentsString:      publishedSurvivalTalents,
		DistanceFromTarget: 5,
	}, P1PlayerOptions)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	agent, ok := sim.Raid.Parties[0].Players[0].(HunterAgent)
	if !ok {
		t.Fatal("the raid's first player is not a hunter agent")
	}
	return sim, agent.GetHunter()
}

// castingTickGain is the mana one regen tick restores inside the five-second
// rule, from a pool well under its cap.
func castingTickGain(sim *core.Simulation, hunter *Hunter, metrics *core.ResourceMetrics) float64 {
	hunter.SpendMana(sim, hunter.CurrentMana()/2, metrics)
	hunter.PseudoStats.FiveSecondRuleRefreshTime = sim.CurrentTime + 5*time.Second
	before := hunter.CurrentMana()
	hunter.ManaTick(sim)
	return hunter.CurrentMana() - before
}

// TestResourcefulnessRegenContinuesWhileCasting: Resourcefulness 440529
// "grants your melee and ranged ability Critical Strikes a 100% chance to
// allow 50% of your Mana Regeneration to continue while casting for 30 sec".
// The aura raised PseudoStats.SpiritRegenRateCasting, but the unit's regen
// per tick is cached (core UpdateManaRegenRates) and nothing refreshed it, so
// the casting tick never grew.
func TestResourcefulnessRegenContinuesWhileCasting(t *testing.T) {
	sim, hunter := buildPublishedSurvivalHunter(t)
	aura := hunter.GetAura("Resourcefulness")
	if aura == nil {
		t.Fatal("the published Survival build registers no Resourcefulness aura")
	}
	metrics := hunter.NewManaMetrics(core.ActionID{SpellID: 440529, Tag: 99})

	without := castingTickGain(sim, hunter, metrics)
	aura.Activate(sim)
	with := castingTickGain(sim, hunter, metrics)

	spirit := hunter.SpiritManaRegenPerSecondDefault()
	if hunter.SpiritManaRegenPerSecond != nil {
		spirit = hunter.SpiritManaRegenPerSecond()
	}
	want := 0.5 * spirit * hunter.PseudoStats.SpiritRegenMultiplier * 2 // half the spirit regen, one 2 s tick
	if got := with - without; want <= 0 || math.Abs(got-want) > 0.01 {
		t.Errorf("Resourcefulness adds %.2f mana a casting tick, want %.2f (50%% of %.2f spirit regen a second)", got, want, spirit)
	}
}
