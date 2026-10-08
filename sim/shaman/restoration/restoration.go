package restoration

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/shaman"
)

func RegisterRestorationShaman() {
	core.RegisterAgentFactory(
		proto.Player_RestorationShaman{},
		proto.Spec_SpecRestorationShaman,
		func(character *core.Character, options *proto.Player) core.Agent {
			return NewRestorationShaman(character, options)
		},
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_RestorationShaman)
			if !ok {
				panic("Invalid spec value for Restoration Shaman!")
			}
			player.Spec = playerSpec
		},
	)
}

// RestorationShaman has no options of its own: the proto's earth_shield_p_p_m
// and totems are deprecated, Earth Shield is not learnable in Forever (the
// client has no learn row for it) and the rotation drops the totems.
type RestorationShaman struct {
	*shaman.Shaman
}

func NewRestorationShaman(character *core.Character, options *proto.Player) *RestorationShaman {
	_ = options.GetRestorationShaman()

	return &RestorationShaman{
		Shaman: shaman.NewShaman(character, options.TalentsString),
	}
}

func (resto *RestorationShaman) GetShaman() *shaman.Shaman {
	return resto.Shaman
}

func (resto *RestorationShaman) Reset(sim *core.Simulation) {
	resto.Shaman.Reset(sim)
}

// GetMainTarget is the tank: the last fake raid member, the one the raid
// damage model aims its melee hits at. Without fake members (a bare
// character build) the shaman is its own target.
func (resto *RestorationShaman) GetMainTarget() *core.Unit {
	var tank *core.TargetDummy
	for _, party := range resto.Env.Raid.Parties {
		for _, player := range party.Players {
			if dummy, ok := player.(*core.TargetDummy); ok {
				tank = dummy
			}
		}
	}
	if tank == nil {
		return &resto.Unit
	}
	return &tank.Unit
}

func (resto *RestorationShaman) Initialize() {
	resto.CurrentTarget = resto.GetMainTarget()
	resto.Shaman.Initialize()
	resto.RegisterHealingSpells()
}
