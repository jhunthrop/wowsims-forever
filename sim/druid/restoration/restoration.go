package restoration

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/druid"
)

func RegisterRestorationDruid() {
	core.RegisterAgentFactory(
		proto.Player_RestorationDruid{},
		proto.Spec_SpecRestorationDruid,
		func(character *core.Character, options *proto.Player) core.Agent {
			return NewRestorationDruid(character, options)
		},
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_RestorationDruid)
			if !ok {
				panic("Invalid spec value for Restoration Druid!")
			}
			player.Spec = playerSpec
		},
	)
}

func NewRestorationDruid(character *core.Character, options *proto.Player) *RestorationDruid {
	restoOptions := options.GetRestorationDruid()
	resto := &RestorationDruid{
		Druid:   druid.New(character, druid.Humanoid, druid.SelfBuffs{}, options.TalentsString),
		Options: restoOptions.Options,
	}

	// A restoration druid casts Nature's Swiftness ahead of the heal it makes
	// instant, not the moment it is ready.
	resto.CastsNaturesSwiftnessByHand = true

	resto.SelfBuffs.InnervateTarget = &proto.UnitReference{Type: proto.UnitReference_Self}
	if target := restoOptions.Options.GetInnervateTarget(); target.GetType() != proto.UnitReference_Unknown {
		// A restoration druid casts Nature's Swiftness ahead of the heal it makes
		// instant, not the moment it is ready.
		resto.CastsNaturesSwiftnessByHand = true

		resto.SelfBuffs.InnervateTarget = target
	}

	return resto
}

type RestorationDruid struct {
	*druid.Druid

	Options *proto.RestorationDruid_Options
}

func (resto *RestorationDruid) GetDruid() *druid.Druid {
	return resto.Druid
}

// GetMainTarget is who a heal with no explicit target lands on: the tank, the
// last fake raid member (see sim/healsim), or the druid when the raid has none.
func (resto *RestorationDruid) GetMainTarget() *core.Unit {
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

func (resto *RestorationDruid) Initialize() {
	resto.CurrentTarget = resto.GetMainTarget()
	resto.Druid.Initialize()
	resto.RegisterHealingSpells()
}

func (resto *RestorationDruid) Reset(sim *core.Simulation) {
	resto.Druid.Reset(sim)
}
