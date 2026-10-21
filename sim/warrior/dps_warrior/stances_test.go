package dpswarrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/warrior"
)

// Berserker Stance's OnExpire multiplied DamageTakenMultiplier by 1.1
// where OnGain had already multiplied by 1.1, so every stance cycle
// compounded +10% damage taken instead of undoing the stance. It is
// pre-existing at master and invisible to the DPS suites, which never
// leave Berserker Stance; the tank suite that would show it is skipped
// awaiting the Forever talent rewrite. A stance's OnExpire must return
// every pseudo-stat it touched to where it found it.
func TestLeavingBerserkerStanceUndoesItsDamageTaken(t *testing.T) {
	sim, war := newWarriorSim(t, warrior.ForeverFuryTalents)

	// The Fury build opens in Berserker Stance, so the baseline is
	// taken with Battle Stance active.
	war.BattleStanceAura.Activate(sim)
	base := war.PseudoStats.DamageTakenMultiplier

	for cycle := 0; cycle < 3; cycle++ {
		war.BerserkerStanceAura.Activate(sim)
		if got := war.PseudoStats.DamageTakenMultiplier; got != base*1.1 {
			t.Fatalf("cycle %d: in Berserker Stance DamageTakenMultiplier is %v, want %v", cycle, got, base*1.1)
		}
		// Battle Stance shares the exclusive stance category, so
		// activating it expires Berserker Stance the way a real stance
		// dance does.
		war.BattleStanceAura.Activate(sim)
		if got := war.PseudoStats.DamageTakenMultiplier; got != base {
			t.Fatalf("cycle %d: after leaving Berserker Stance DamageTakenMultiplier is %v, want the original %v", cycle, got, base)
		}
	}
}

// newWarriorSim returns a reset sim and the warrior in it, for tests
// that need to activate auras rather than only read registered spells.
func newWarriorSim(t *testing.T, talents string) (*core.Simulation, *warrior.Warrior) {
	t.Helper()

	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{
			Parties: []*proto.Party{{
				Players: []*proto.Player{{
					Name:          "Warrior",
					Class:         proto.Class_ClassWarrior,
					Race:          proto.Race_RaceOrc,
					TalentsString: talents,
					Consumes:      &proto.Consumes{},
					Buffs:         &proto.IndividualBuffs{},
					Spec:          PlayerOptionsFury,
					Equipment:     &proto.EquipmentSpec{},
				}},
				Buffs: &proto.PartyBuffs{},
			}},
		},
		Encounter: &proto.Encounter{
			Targets:  []*proto.Target{{Name: "target", Level: 63, MobType: proto.MobType_MobTypeDemon}},
			Duration: 60,
		},
	}, simsignals.CreateSignals())
	sim.Reset()

	agent, ok := sim.Raid.Parties[0].Players[0].(warrior.WarriorAgent)
	if !ok {
		t.Fatalf("the raid's first player is not a warrior agent")
	}
	return sim, agent.GetWarrior()
}
