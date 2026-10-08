package tankwarrior

import (
	"fmt"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// The tank fight harness: the boss the site measures a tank against
// (data/curated/tank-encounter.json, applied by sim/request's
// applyTankFight) built by hand so this package can run it without the
// site. One level-63 boss with no creature type swings a single melee
// attack at the tank, the tank stands in front of it, and a healing
// model heals him on a steady cadence.

// bossProfile is the boss's melee and the healers' output.
type bossProfile struct {
	swingSpeed        float64
	minBaseDamage     float64
	damageSpread      float64
	healsPerSecond    float64
	healCadence       float64
	healCadenceSpread float64
	burstWindow       int32
}

// bossArmor is the level-63 boss preset of the site's api.TargetArmorByLevel.
const bossArmor = 3731

// siteBoss is data/curated/tank-encounter.json at the reference level.
var siteBoss = bossProfile{
	swingSpeed: 2.0, minBaseDamage: 1900, damageSpread: 0.33,
	healsPerSecond: 440, healCadence: 2.0, healCadenceSpread: 0.5, burstWindow: 6,
}

// leadBoss is the lead's brief: a harder hit and more healing.
var leadBoss = bossProfile{
	swingSpeed: 2.0, minBaseDamage: 2400, damageSpread: 0.33,
	healsPerSecond: 600, healCadence: 2.0, healCadenceSpread: 0.2, burstWindow: 6,
}

const (
	fightSeconds   = 180
	harnessRounds  = 400
	harnessSeed    = 20261007
	harnessGearSet = "forever_l60"
	gearSetsDir    = "../../../ui/tank_warrior/gear_sets"
)

// fightConfig is one harness run.
type fightConfig struct {
	talents string
	apl     *proto.APLRotation
	boss    bossProfile
	gear    *proto.EquipmentSpec
	debuffs *proto.Debuffs
	level   int32
	options *proto.Player_TankWarrior
	noBuffs bool
	// bossAttackPower is the boss's attack power: 0 unless a test asks,
	// because the curated tank boss carries none.
	bossAttackPower float64
}

// fightResult is what the harness prints and the tests assert on.
type fightResult struct {
	dtps, tmi, chanceOfDeath, threatPerSecond, dps float64
}

func (r fightResult) String() string {
	return fmt.Sprintf("DTPS %7.1f  TMI %7.1f  death %5.3f  TPS %7.1f  DPS %6.1f", r.dtps, r.tmi, r.chanceOfDeath, r.threatPerSecond, r.dps)
}

func harnessGear() *proto.EquipmentSpec {
	return core.GetGearSet(gearSetsDir, harnessGearSet).GearSet
}

func harnessAPL(path string) *proto.APLRotation {
	return core.GetAplRotation("../../../ui/tank_warrior/apls", path).Rotation
}

func (cfg fightConfig) request() *proto.RaidSimRequest {
	level := cfg.level
	if level == 0 {
		level = 60
	}
	options := cfg.options
	if options == nil {
		options = PlayerOptionsBasic
	}
	buffs := core.FullBuffs
	raidBuffs, partyBuffs, playerBuffs := buffs.Raid, buffs.Party, buffs.Player
	if cfg.noBuffs {
		raidBuffs, partyBuffs, playerBuffs = &proto.RaidBuffs{}, &proto.PartyBuffs{}, &proto.IndividualBuffs{}
	}
	player := core.WithSpec(&proto.Player{
		Name:            "Tank",
		Class:           proto.Class_ClassWarrior,
		Race:            proto.Race_RaceOrc,
		Level:           level,
		Equipment:       cfg.gear,
		Consumes:        P1Consumes.Consumes,
		Buffs:           playerBuffs,
		TalentsString:   cfg.talents,
		Rotation:        cfg.apl,
		InFrontOfTarget: true,
		ReactionTimeMs:  150,
		HealingModel: &proto.HealingModel{
			Hps:              cfg.boss.healsPerSecond,
			CadenceSeconds:   cfg.boss.healCadence,
			CadenceVariation: cfg.boss.healCadenceSpread,
			BurstWindow:      cfg.boss.burstWindow,
		},
	}, options)
	debuffs := cfg.debuffs
	if debuffs == nil {
		debuffs = &proto.Debuffs{}
	}
	raid := core.SinglePlayerRaidProto(player, partyBuffs, raidBuffs, debuffs)
	raid.Tanks = []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 0}}
	return &proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: fightSeconds,
			Targets: []*proto.Target{{
				Level:         63,
				Stats:         stats.Stats{stats.Armor: bossArmor, stats.AttackPower: cfg.bossAttackPower}.ToFloatArray(),
				TankIndex:     0,
				SwingSpeed:    cfg.boss.swingSpeed,
				MinBaseDamage: cfg.boss.minBaseDamage,
				DamageSpread:  cfg.boss.damageSpread,
				ParryHaste:    true,
			}},
		},
		SimOptions: &proto.SimOptions{Iterations: harnessRounds, RandomSeed: harnessSeed, IsTest: true},
	}
}

func runFight(t testing.TB, cfg fightConfig) fightResult {
	t.Helper()
	result := core.RunRaidSim(cfg.request())
	if result.Error != nil {
		t.Fatalf("fight failed: %s", result.Error.Message)
	}
	m := result.RaidMetrics.Parties[0].Players[0]
	return fightResult{
		dtps:            m.Dtps.Avg,
		tmi:             m.Tmi.Avg,
		chanceOfDeath:   m.ChanceOfDeath,
		threatPerSecond: m.Threat.Avg,
		dps:             m.Dps.Avg,
	}
}
