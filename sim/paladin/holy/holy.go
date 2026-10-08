// Package holy is the healing Paladin: Holy Light, Flash of Light, Holy
// Shock and Light's Vigil on a fake raid (sim/healsim), with the shared
// Paladin's seals, auras and talents underneath.
package holy

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/paladin"
)

func RegisterHolyPaladin() {
	core.RegisterAgentFactory(
		proto.Player_HolyPaladin{},
		proto.Spec_SpecHolyPaladin,
		func(character *core.Character, options *proto.Player) core.Agent {
			return NewHolyPaladin(character, options)
		},
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_HolyPaladin)
			if !ok {
				panic("Invalid spec value for Holy Paladin!")
			}
			player.Spec = playerSpec
		},
	)
}

func NewHolyPaladin(character *core.Character, options *proto.Player) *HolyPaladin {
	holyOptions := options.GetHolyPaladin().Options

	return &HolyPaladin{
		Paladin: paladin.NewPaladin(character, options, holyOptions),
	}
}

type HolyPaladin struct {
	*paladin.Paladin
}

func (holy *HolyPaladin) GetPaladin() *paladin.Paladin {
	return holy.Paladin
}

// GetMainTarget is the tank: the last fake raid member (sim/healsim lays
// the raid out), which a heal with no explicit target lands on. Without a
// fake raid the healer is its own target.
func (holy *HolyPaladin) GetMainTarget() *core.Unit {
	if tank := lastTargetDummy(holy.Env.Raid); tank != nil {
		return &tank.Unit
	}
	return &holy.Unit
}

func lastTargetDummy(raid *core.Raid) *core.TargetDummy {
	var tank *core.TargetDummy
	for _, party := range raid.Parties {
		for _, player := range party.Players {
			if dummy, ok := player.(*core.TargetDummy); ok {
				tank = dummy
			}
		}
	}
	return tank
}

func (holy *HolyPaladin) Initialize() {
	holy.CurrentTarget = holy.GetMainTarget()
	holy.Paladin.Initialize()
}

func (holy *HolyPaladin) Reset(sim *core.Simulation) {
	holy.Paladin.Reset(sim)
}
