package elemental

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/shaman"
)

func newManaSpringShaman(t *testing.T, raidBuffs *proto.RaidBuffs) (*core.Simulation, *shaman.Shaman) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:     proto.Class_ClassShaman,
			Race:      proto.Race_RaceTroll,
			Level:     60,
			Equipment: &proto.EquipmentSpec{},
		},
		&proto.Player_ElementalShaman{
			ElementalShaman: &proto.ElementalShaman{Options: &proto.ElementalShaman_Options{}},
		},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, raidBuffs, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	agent, ok := sim.Raid.Parties[0].Players[0].(shaman.ShamanAgent)
	if !ok {
		t.Fatal("the raid's first player is not a shaman agent")
	}
	return sim, agent.GetShaman()
}

// Round 11 (casters X1): two Mana Spring auras do not stack, so under the
// raid buffs' Mana Spring Totem the shaman's own totem adds nothing
// (water_totems.go keeps only its excess). The rotation's Mana Spring line
// then spent the fight's first global and its mana for nothing. A totem
// that would add nothing over the raid's is not cast; with no raid Mana
// Spring it is.
func TestOwnManaSpringIsNotCastUnderTheRaidsManaSpring(t *testing.T) {
	top := func(s *shaman.Shaman) *core.Spell {
		spell := s.ManaSpringTotem[len(s.ManaSpringTotem)-1]
		if spell == nil {
			t.Fatal("level-60 shaman has no top Mana Spring Totem rank")
		}
		return spell
	}

	sim, s := newManaSpringShaman(t, &proto.RaidBuffs{ManaSpringTotem: proto.TristateEffect_TristateEffectRegular})
	if top(s).CanCast(sim, s.CurrentTarget) {
		t.Error("own Mana Spring Totem castable under the raid's Mana Spring, which it cannot add to")
	}

	sim, s = newManaSpringShaman(t, &proto.RaidBuffs{})
	if !top(s).CanCast(sim, s.CurrentTarget) {
		t.Error("own Mana Spring Totem not castable with no raid Mana Spring")
	}
}
