package conformance

import (
	"fmt"

	_ "github.com/wowsims/classic/sim/common" // item effects, same as every level_smoke_test.go
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/druid/balance"
	"github.com/wowsims/classic/sim/druid/feral"
	"github.com/wowsims/classic/sim/hunter"
	"github.com/wowsims/classic/sim/mage"
	"github.com/wowsims/classic/sim/paladin/protection"
	"github.com/wowsims/classic/sim/paladin/retribution"
	"github.com/wowsims/classic/sim/priest/shadow"
	dpsrogue "github.com/wowsims/classic/sim/rogue/dps_rogue"
	"github.com/wowsims/classic/sim/shaman/elemental"
	"github.com/wowsims/classic/sim/shaman/enhancement"
	"github.com/wowsims/classic/sim/shaman/warden"
	dpswarlock "github.com/wowsims/classic/sim/warlock/dps"
	dpswarrior "github.com/wowsims/classic/sim/warrior/dps_warrior"
	tankwarrior "github.com/wowsims/classic/sim/warrior/tank_warrior"
)

// registerAll registers every Preset's agent factory (core.RegisterAgentFactory,
// via sim/core/agent.go), exactly the set sim/register_all.go registers for
// the site. It is not that function directly because RegisterAll's package
// (sim) imports every class package as its own import block already built
// for the production registration order; duplicating the calls here keeps
// this package's only reason to import each class package visible in one
// place, next to Presets.
func registerAll() {
	hunter.RegisterHunter()
	mage.RegisterMage()
	dpswarlock.RegisterDpsWarlock()
	protection.RegisterProtectionPaladin()
	retribution.RegisterRetributionPaladin()
	dpswarrior.RegisterDpsWarrior()
	tankwarrior.RegisterTankWarrior()
	balance.RegisterBalanceDruid()
	feral.RegisterFeralDruid()
	shadow.RegisterShadowPriest()
	elemental.RegisterElementalShaman()
	enhancement.RegisterEnhancementShaman()
	warden.RegisterWardenShaman()
	dpsrogue.RegisterDpsRogue()
}

// buildEncounter is a single default target; no gear, buffs or duration
// matter here since this package never runs a sim, only builds a
// character and reads its Spellbook (core/level_smoke.go's own
// smokeEncounter does the same, for the same reason).
func buildEncounter() *proto.Encounter {
	return &proto.Encounter{
		Duration: 10,
		Targets:  []*proto.Target{core.NewDefaultTarget()},
	}
}

// buildCharacter builds preset at level exactly as core.RunLevelSmoke does
// (no gear; RunLevelSmoke's own doc comment explains why), but returns the
// built character instead of running a sim, since this package only reads
// registered spells. A panic during registration (a level-awareness defect
// like the ones RunLevelSmoke itself was written to catch, or - for
// CombatSwordsRogue/CombatDaggersRogue - the WeaponExpertise talent-rewrite
// gap sim/rogue/dps_rogue's own TestLevelSmoke already skips around) is
// recovered and returned as an error, so one spec's defect does not stop
// every other spec's report row.
func buildCharacter(preset Preset, level int32) (built *core.Character, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic building %s at level %d: %v", preset.Label, level, r)
		}
	}()

	distance := preset.DistanceFromTarget
	if distance == 0 {
		distance = 5
	}

	player := core.WithSpec(&proto.Player{
		Class:              preset.Class,
		Race:               preset.Race,
		Level:              level,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      preset.Talents,
		DistanceFromTarget: distance,
	}, preset.SpecOptions)

	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	env, _, _ := core.NewEnvironment(raid, buildEncounter(), true)
	built = env.Raid.Parties[0].Players[0].GetCharacter()
	if built.Level != level {
		return nil, fmt.Errorf("%s at level %d: built character Level = %d", preset.Label, level, built.Level)
	}
	return built, nil
}
