package protection

import (
	"fmt"
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/paladin"
)

// The tank harness: one level 60 Protection paladin in decent gear tanking
// the published tank boss for 180 seconds under a healing model, as the
// lead's tank profile states it: level 63, no creature type, a single
// melee swing every 2.0 s rolling 2400 to 3192 before armor (crushing
// blows come from the +3 level gap in the attack table), the paladin as
// Raid.Tanks[0] and in front of the boss, and a healer at about 600 HPS in
// 2 s casts. It prints what a tank is judged on, so a rotation or a
// talent choice is measured, not argued.

const (
	harnessLevel           = 60
	harnessFightSeconds    = 180
	harnessIterations      = 2000
	harnessBossMinDamage   = 2400
	harnessBossSpread      = 0.33
	harnessBossSwingSpeed  = 2.0
	harnessBossLevel       = 63
	harnessHealingHPS      = 600
	harnessHealingCadence  = 2.0
	harnessHealingVariance = 0.2
	harnessTMIBurstWindow  = 6
	harnessGearSet         = "forever_l60"
	harnessRotation        = "forever_protection"
)

// harnessConfig is what a measurement varies; the zero value is the
// reference configuration.
type harnessConfig struct {
	Talents       string
	Rotation      *proto.APLRotation
	Options       *proto.PaladinOptions
	GearSet       string
	Healing       *proto.HealingModel
	BossMinDamage float64
}

// harnessResult is the per-iteration average of the tank's metrics.
type harnessResult struct {
	DPS           float64
	ThreatPerSec  float64
	DTPS          float64
	TMI           float64
	ChanceOfDeath float64

	// Actions is one line per ability, per second of the fight, and
	// Resources the mana and health the tank gained and spent by source.
	Actions   []string
	Resources []string
	Auras     []string
}

func harnessHealingModel() *proto.HealingModel {
	return &proto.HealingModel{
		Hps:              harnessHealingHPS,
		CadenceSeconds:   harnessHealingCadence,
		CadenceVariation: harnessHealingVariance,
		BurstWindow:      harnessTMIBurstWindow,
	}
}

func harnessBoss(minDamage float64) *proto.Target {
	return &proto.Target{
		Level:         harnessBossLevel,
		MobType:       proto.MobType_MobTypeUnknown,
		MinBaseDamage: minDamage,
		DamageSpread:  harnessBossSpread,
		SwingSpeed:    harnessBossSwingSpeed,
		TankIndex:     0,
	}
}

func (cfg harnessConfig) withDefaults() harnessConfig {
	if cfg.Talents == "" {
		cfg.Talents = paladin.ForeverProtectionTalents
	}
	if cfg.Rotation == nil {
		cfg.Rotation = core.GetAplRotation("../../../ui/protection_paladin/apls", harnessRotation).Rotation
	}
	if cfg.Options == nil {
		cfg.Options = &proto.PaladinOptions{RighteousFury: true}
	}
	if cfg.GearSet == "" {
		cfg.GearSet = harnessGearSet
	}
	if cfg.Healing == nil {
		cfg.Healing = harnessHealingModel()
	}
	if cfg.BossMinDamage == 0 {
		cfg.BossMinDamage = harnessBossMinDamage
	}
	return cfg
}

func runHarness(t testing.TB, cfg harnessConfig) harnessResult {
	t.Helper()
	cfg = cfg.withDefaults()

	player := core.WithSpec(&proto.Player{
		Class:           proto.Class_ClassPaladin,
		Race:            proto.Race_RaceHuman,
		Level:           harnessLevel,
		Equipment:       core.GetGearSet("../../../ui/protection_paladin/gear_sets", cfg.GearSet).GearSet,
		Buffs:           core.FullBuffs.Player,
		TalentsString:   cfg.Talents,
		Rotation:        cfg.Rotation,
		InFrontOfTarget: true,
		HealingModel:    cfg.Healing,
		ReactionTimeMs:  150,
	}, &proto.Player_ProtectionPaladin{ProtectionPaladin: &proto.ProtectionPaladin{Options: cfg.Options}})

	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, nil)
	raid.Tanks = append(raid.Tanks, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0})

	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: harnessFightSeconds,
			Targets:  []*proto.Target{harnessBoss(cfg.BossMinDamage)},
		},
		SimOptions: &proto.SimOptions{Iterations: harnessIterations, IsTest: true, RandomSeed: 101},
	})
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}

	metrics := result.RaidMetrics.Parties[0].Players[0]
	return harnessResult{
		DPS:           metrics.Dps.Avg,
		ThreatPerSec:  metrics.Threat.Avg,
		DTPS:          metrics.Dtps.Avg,
		TMI:           metrics.Tmi.Avg,
		ChanceOfDeath: metrics.ChanceOfDeath,
		Actions:       actionLines(metrics),
		Resources:     resourceLines(metrics),
		Auras:         auraLines(metrics),
	}
}

// auraLines is the uptime in seconds of the auras that decide a tank's
// mitigation, and how often each came up.
func auraLines(metrics *proto.UnitMetrics) []string {
	var lines []string
	for _, aura := range metrics.Auras {
		switch aura.Id.GetSpellId() {
		case spellHolyShield, spellSealOfFury, auraRedoubt, auraSealOfFuryShield, auraIronCreed:
			lines = append(lines, fmt.Sprintf("  aura %-8d uptime %6.1fs procs %6.1f", aura.Id.GetSpellId(), aura.UptimeSecondsAvg, aura.ProcsAvg))
		}
	}
	return lines
}

// perSecond turns a total over every iteration into a per second figure.
const perSecond = harnessFightSeconds * harnessIterations

func actionLines(metrics *proto.UnitMetrics) []string {
	var lines []string
	for _, action := range metrics.Actions {
		var casts, damage, threat, shielding float64
		for _, target := range action.Targets {
			casts += float64(target.Casts)
			damage += target.Damage
			threat += target.Threat
			shielding += target.Shielding
		}
		lines = append(lines, fmt.Sprintf("  spell %-8d tag %d  casts %6.1f  dmg/s %7.1f  threat/s %7.1f  shield/s %6.1f",
			action.Id.GetSpellId(), action.Id.Tag, casts/harnessIterations, damage/perSecond, threat/perSecond, shielding/perSecond))
	}
	return lines
}

func resourceLines(metrics *proto.UnitMetrics) []string {
	var lines []string
	for _, resource := range metrics.Resources {
		lines = append(lines, fmt.Sprintf("  %s other %d spell %-8d events %7.1f gain/s %7.2f actual/s %7.2f",
			resource.Type, resource.Id.GetOtherId(), resource.Id.GetSpellId(), float64(resource.Events)/harnessIterations, resource.Gain/perSecond, resource.ActualGain/perSecond))
	}
	return lines
}

func (result harnessResult) String() string {
	return fmt.Sprintf("DPS %.1f  threat/s %.1f  DTPS %.1f  TMI %.1f  death %.3f",
		result.DPS, result.ThreatPerSec, result.DTPS, result.TMI, result.ChanceOfDeath)
}
