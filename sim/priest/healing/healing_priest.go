package healing

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/priest"
)

// innerFirePrepullAt is when the option's Inner Fire is cast: just before
// the pull, as a player buffs.
const innerFirePrepullAt = -time.Second

func RegisterHealingPriest() {
	core.RegisterAgentFactory(
		proto.Player_HealingPriest{},
		proto.Spec_SpecHealingPriest,
		func(character *core.Character, options *proto.Player) core.Agent {
			return NewHealingPriest(character, options)
		},
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_HealingPriest)
			if !ok {
				panic("Invalid spec value for Healing Priest!")
			}
			player.Spec = playerSpec
		},
	)
}

type HealingPriest struct {
	*priest.Priest

	Options *proto.HealingPriest_Options
}

func NewHealingPriest(character *core.Character, options *proto.Player) *HealingPriest {
	healingOptions := options.GetHealingPriest()

	basePriest := priest.New(character, options.TalentsString)
	hpriest := &HealingPriest{
		Priest:  basePriest,
		Options: healingOptions.Options,
	}

	return hpriest
}

func (hpriest *HealingPriest) GetPriest() *priest.Priest {
	return hpriest.Priest
}

// GetMainTarget is the unit a cast with no named target lands on: the
// fake raid's tank when the sim has one, else the encounter's first
// enemy, the way every caster's default target is its enemy (so Smite and
// Shadow Word: Pain have a target), else the priest itself.
func (hpriest *HealingPriest) GetMainTarget() *core.Unit {
	if tank := tankDummy(hpriest.Env.Raid); tank != nil {
		return tank
	}
	if len(hpriest.Env.Encounter.Targets) > 0 {
		return hpriest.Env.GetTargetUnit(0)
	}
	return &hpriest.Unit
}

// tankDummy is the fake raid's tank: the damage model makes the last
// target dummy the tank and the ones before it the members the raid-wide
// pulses land on.
func tankDummy(raid *core.Raid) *core.Unit {
	var tank *core.Unit
	for _, party := range raid.Parties {
		for _, player := range party.Players {
			if dummy, ok := player.(*core.TargetDummy); ok {
				tank = &dummy.Unit
			}
		}
	}
	return tank
}

func (hpriest *HealingPriest) Initialize() {
	hpriest.CurrentTarget = hpriest.GetMainTarget()
	hpriest.Priest.Initialize()
	hpriest.RegisterHealingSpells()
	hpriest.RegisterPowerInfusion(hpriest.GetUnit(hpriest.Options.GetPowerInfusionTarget()))
	if hpriest.Options.GetUseInnerFire() {
		hpriest.castInnerFireBeforeThePull()
	}
}

// castInnerFireBeforeThePull buffs with the highest rank the priest knows.
func (hpriest *HealingPriest) castInnerFireBeforeThePull() {
	for rank := len(hpriest.InnerFire) - 1; rank > 0; rank-- {
		if spell := hpriest.InnerFire[rank]; spell != nil {
			hpriest.RegisterPrepullAction(innerFirePrepullAt, func(sim *core.Simulation) {
				spell.Cast(sim, &hpriest.Unit)
			})
			return
		}
	}
}

func (hpriest *HealingPriest) Reset(sim *core.Simulation) {
}
