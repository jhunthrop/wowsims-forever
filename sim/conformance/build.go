package conformance

import (
	"fmt"

	_ "github.com/wowsims/classic/sim/common" // item effects, same as every level_smoke_test.go
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/druid/balance"
	"github.com/wowsims/classic/sim/druid/feral"
	"github.com/wowsims/classic/sim/druid/restoration"
	bear "github.com/wowsims/classic/sim/druid/tank"
	"github.com/wowsims/classic/sim/hunter"
	"github.com/wowsims/classic/sim/mage"
	holypaladin "github.com/wowsims/classic/sim/paladin/holy"
	"github.com/wowsims/classic/sim/paladin/protection"
	"github.com/wowsims/classic/sim/paladin/retribution"
	"github.com/wowsims/classic/sim/priest/healing"
	"github.com/wowsims/classic/sim/priest/shadow"
	dpsrogue "github.com/wowsims/classic/sim/rogue/dps_rogue"
	"github.com/wowsims/classic/sim/shaman/elemental"
	"github.com/wowsims/classic/sim/shaman/enhancement"
	shamanrestoration "github.com/wowsims/classic/sim/shaman/restoration"
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
	holypaladin.RegisterHolyPaladin()
	protection.RegisterProtectionPaladin()
	retribution.RegisterRetributionPaladin()
	dpswarrior.RegisterDpsWarrior()
	tankwarrior.RegisterTankWarrior()
	balance.RegisterBalanceDruid()
	feral.RegisterFeralDruid()
	bear.RegisterFeralTankDruid()
	restoration.RegisterRestorationDruid()
	healing.RegisterHealingPriest()
	shadow.RegisterShadowPriest()
	elemental.RegisterElementalShaman()
	enhancement.RegisterEnhancementShaman()
	warden.RegisterWardenShaman()
	shamanrestoration.RegisterRestorationShaman()
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
// registered spells. talentsString is taken as an explicit parameter
// rather than read off preset.Talents: every caller in this package needs
// a DIFFERENT talent string from the same preset's class/race/gear/spec
// options - the empty string for the talent-free comparison table
// (buildForComparison), preset.Talents itself only as that function's
// last-resort fallback, and a single talent at rank 1 for one entry of
// TalentGatedSpells - so the preset's own Talents field is never the
// right default to reach for silently.
//
// A panic during registration (a level-awareness defect like the ones
// RunLevelSmoke itself was written to catch, or - for
// CombatSwordsRogue/CombatDaggersRogue built with their real talent
// strings - the WeaponExpertise talent-rewrite gap sim/rogue/dps_rogue's
// own TestLevelSmoke already skips around) is recovered and returned as
// an error, so one spec's defect does not stop every other spec's
// report row.
func buildCharacter(preset Preset, level int32, talentsString string) (built *core.Character, err error) {
	return buildCharacterWearing(preset, level, talentsString, &proto.EquipmentSpec{}, nil)
}

// buildCharacterWearing is buildCharacter with gear: equipment, and the
// database that defines any item in it the engine does not already know.
func buildCharacterWearing(preset Preset, level int32, talentsString string, equipment *proto.EquipmentSpec, database *proto.SimDatabase) (built *core.Character, err error) {
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
		Equipment:          equipment,
		Database:           database,
		Buffs:              core.FullBuffs.Player,
		TalentsString:      talentsString,
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

// buildForComparison builds preset at level with an EMPTY talent string:
// every row the main golden table reports is measured against this bare
// build, not against the preset's real talent spend, so a talent
// modifier (Improved Frostbolt shaving Frostbolt's cost, Reverberation
// shaving Earth Shock's, Improved Starfire shaving Starfire's, and every
// other percent- or flat-reduction talent across the thirteen presets)
// can never again read as a client/engine mismatch - only a genuine
// base-number or registration defect can.
//
// If the empty-talent build itself cannot register (none of today's
// thirteen presets hits this - every one was verified to build clean
// with "" - but a future spec could gate a spell its own Initialize
// requires further down the registration chain), it falls back to the
// preset's real talent string so one spec's gap does not drop every
// other row from the report, and the fallback is returned as a non-empty
// note for the golden's header to record. That fallback build is NOT
// talent-free, so any mismatch it reports should be read with that in
// mind - the note exists so a reader knows to.
func buildForComparison(preset Preset, level int32) (built *core.Character, fallbackNote string, err error) {
	built, err = buildCharacter(preset, level, "")
	if err == nil {
		return built, "", nil
	}
	emptyErr := err

	built, err = buildCharacter(preset, level, preset.Talents)
	if err != nil {
		return nil, "", fmt.Errorf("empty-talent build failed (%v), and the preset's own full talent build also failed: %w", emptyErr, err)
	}
	return built, fmt.Sprintf("%s/L%d: empty-talent build failed (%v); fell back to the preset's full talent build, so this spec/level's rows are not talent-free", preset.Label, level, emptyErr), nil
}
