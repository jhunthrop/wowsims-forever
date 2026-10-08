package tank

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/druid"
)

func RegisterFeralTankDruid() {
	core.RegisterAgentFactory(
		proto.Player_FeralTankDruid{},
		proto.Spec_SpecFeralTankDruid,
		func(character *core.Character, options *proto.Player) core.Agent {
			return NewFeralTankDruid(character, options)
		},
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_FeralTankDruid)
			if !ok {
				panic("Invalid spec value for Feral Tank Druid!")
			}
			player.Spec = playerSpec
		},
	)
}

func NewFeralTankDruid(character *core.Character, options *proto.Player) *FeralTankDruid {
	tankOptions := options.GetFeralTankDruid()
	if tankOptions == nil || tankOptions.Options == nil {
		panic("Feral Tank Druid needs its options")
	}

	bear := &FeralTankDruid{
		Druid:   druid.New(character, druid.Bear, druid.SelfBuffs{}, options.TalentsString),
		Options: tankOptions.Options,
	}

	bear.SelfBuffs.InnervateTarget = &proto.UnitReference{}
	if tankOptions.Options.InnervateTarget != nil {
		bear.SelfBuffs.InnervateTarget = tankOptions.Options.InnervateTarget
	}

	bear.EnableRageBar(core.RageBarOptions{
		StartingRage:          bear.Options.StartingRage,
		DamageDealtMultiplier: 1,
		DamageTakenMultiplier: 1,
	})

	bear.EnableAutoAttacks(bear, core.AutoAttackOptions{
		// Base paw weapon.
		MainHand:       bear.GetBearWeapon(),
		AutoSwingMelee: true,
		ReplaceMHSwing: func(sim *core.Simulation, mhSwingSpell *core.Spell) *core.Spell {
			return bear.ReplaceBearMHFunc(sim, mhSwingSpell)
		},
	})
	bear.ReplaceBearMHFunc = func(_ *core.Simulation, mhSwingSpell *core.Spell) *core.Spell {
		return mhSwingSpell
	}

	bear.PseudoStats.FeralCombatEnabled = true

	if healingModel := options.HealingModel; healingModel != nil && healingModel.InspirationUptime > 0 {
		core.ApplyInspiration(bear.GetCharacter(), healingModel.InspirationUptime)
	}

	return bear
}

type FeralTankDruid struct {
	*druid.Druid

	Options *proto.FeralTankDruid_Options
}

func (bear *FeralTankDruid) GetDruid() *druid.Druid {
	return bear.Druid
}

func (bear *FeralTankDruid) Initialize() {
	bear.Druid.Initialize()
	bear.RegisterFeralTankSpells()
}

func (bear *FeralTankDruid) Reset(sim *core.Simulation) {
	bear.Druid.Reset(sim)
	bear.CancelShapeshift(sim)
	bear.BearFormAura.Activate(sim)
	bear.PseudoStats.Stunned = false
}
