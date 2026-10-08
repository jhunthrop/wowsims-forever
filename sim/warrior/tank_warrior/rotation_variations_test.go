package tankwarrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/warrior"
)

// The tank rotation's priorities were chosen by running variations of
// forever_protection (data/curated/apl/warrior-protection.json in the
// site, synced into ui/tank_warrior/apls) through the fight harness, and
// this file keeps those measurements as assertions: each says which way a
// variation moved damage taken, threat or the chance of death, never by how
// much, because the figures move with every number the nightly changes.
// `go test -v -run TestRotationVariations` prints the table.

// Spell ids of the rotation's lines, from the client's tables.
const (
	spellShieldBlock  = 2565
	spellShieldWall   = 871
	spellLastStand    = 12975
	spellThunderClap  = 11581
	spellSunderArmor  = 11597
	spellHeroicStrike = 11567
	spellRevenge      = 11601
	spellShieldSlam   = 23925
	spellBloodrage    = 2687
	spellWhirlwind    = 1680
)

// hardBoss hits hard enough that the healers cannot hold the untalented
// rotation: the fight on which the survival cooldowns earn their keep.
var hardBoss = bossProfile{
	swingSpeed: 2.0, minBaseDamage: 4200, damageSpread: 0.33,
	healsPerSecond: 440, healCadence: 2.0, healCadenceSpread: 0.5, burstWindow: 6,
}

// without is the rotation with every line that casts one of the spells
// removed, the way a variation is made.
func without(rotation *proto.APLRotation, spellIDs ...int32) *proto.APLRotation {
	skip := map[int32]bool{}
	for _, id := range spellIDs {
		skip[id] = true
	}
	kept := make([]*proto.APLListItem, 0, len(rotation.PriorityList))
	for _, item := range rotation.PriorityList {
		if !skip[item.GetAction().GetCastSpell().GetSpellId().GetSpellId()] {
			kept = append(kept, item)
		}
	}
	return &proto.APLRotation{Type: rotation.Type, PrepullActions: rotation.PrepullActions, PriorityList: kept}
}

func tankRotation() *proto.APLRotation { return harnessAPL("forever_protection") }

func fight(t *testing.T, boss bossProfile, rotation *proto.APLRotation) fightResult {
	t.Helper()
	return runFight(t, fightConfig{talents: P1Talents, apl: rotation, boss: boss, gear: harnessGear()})
}

func TestARaidReadyTankSurvivesTheCuratedBosses(t *testing.T) {
	for name, boss := range map[string]bossProfile{"site": siteBoss, "lead": leadBoss} {
		got := fight(t, boss, tankRotation())
		t.Logf("%-5s %s", name, got)
		if got.chanceOfDeath != 0 {
			t.Errorf("%s boss: chance of death %v, want 0", name, got.chanceOfDeath)
		}
		if got.dtps <= 0 || got.threatPerSecond <= 0 {
			t.Errorf("%s boss: DTPS %v and threat per second %v must both be positive", name, got.dtps, got.threatPerSecond)
		}
	}
}

func TestTheTankRotationOutThreatensADpsRotationOnTheSameGear(t *testing.T) {
	berserker := &proto.Player_TankWarrior{TankWarrior: &proto.TankWarrior{Options: &proto.TankWarrior_Options{
		Stance: proto.WarriorStance_WarriorStanceBerserker,
	}}}
	dpsRotation := core.APLRotationFromJsonString(`{"type":"TypeAPL","priorityList":[
		{"action":{"castSpell":{"spellId":{"spellId":2687}}}},
		{"action":{"castSpell":{"spellId":{"spellId":1680}}}},
		{"action":{"condition":{"cmp":{"op":"OpGe","lhs":{"currentRage":{}},"rhs":{"const":{"val":"40"}}}},
			"castSpell":{"spellId":{"spellId":11567,"rank":8,"tag":1}}}}]}`)

	tank := fight(t, siteBoss, tankRotation())
	dps := runFight(t, fightConfig{talents: P1Talents, apl: dpsRotation, boss: siteBoss, gear: harnessGear(), options: berserker})
	t.Logf("tank %s", tank)
	t.Logf("dps  %s", dps)
	if tank.threatPerSecond <= dps.threatPerSecond {
		t.Errorf("tank threat per second %v does not beat the DPS rotation's %v", tank.threatPerSecond, dps.threatPerSecond)
	}
	if tank.dtps >= dps.dtps {
		t.Errorf("tank DTPS %v is not below the Berserker Stance DPS rotation's %v", tank.dtps, dps.dtps)
	}
}

func TestShieldBlockAndThunderClapCutDamageTaken(t *testing.T) {
	full := fight(t, siteBoss, tankRotation())
	noBlock := fight(t, siteBoss, without(tankRotation(), spellShieldBlock))
	noClap := fight(t, siteBoss, without(tankRotation(), spellThunderClap))
	t.Logf("full      %s", full)
	t.Logf("no block  %s", noBlock)
	t.Logf("no clap   %s", noClap)
	if noBlock.dtps <= full.dtps {
		t.Errorf("without Shield Block DTPS %v is not above %v", noBlock.dtps, full.dtps)
	}
	if noClap.dtps <= full.dtps {
		t.Errorf("without Thunder Clap DTPS %v is not above %v", noClap.dtps, full.dtps)
	}
}

func TestSunderArmorAndHeroicStrikeAreThreat(t *testing.T) {
	full := fight(t, siteBoss, tankRotation())
	noSunder := fight(t, siteBoss, without(tankRotation(), spellSunderArmor))
	noStrike := fight(t, siteBoss, without(tankRotation(), spellHeroicStrike))
	t.Logf("no sunder %s", noSunder)
	t.Logf("no strike %s", noStrike)
	if noSunder.threatPerSecond >= full.threatPerSecond {
		t.Errorf("without Sunder Armor threat %v is not below %v", noSunder.threatPerSecond, full.threatPerSecond)
	}
	if noStrike.threatPerSecond >= full.threatPerSecond {
		t.Errorf("without Heroic Strike threat %v is not below %v", noStrike.threatPerSecond, full.threatPerSecond)
	}
}

func TestShieldWallAndLastStandCutTheChanceOfDeath(t *testing.T) {
	full := fight(t, hardBoss, tankRotation())
	bare := fight(t, hardBoss, without(tankRotation(), spellShieldWall, spellLastStand))
	t.Logf("with cooldowns    %s", full)
	t.Logf("without cooldowns %s", bare)
	if bare.chanceOfDeath <= full.chanceOfDeath {
		t.Errorf("the survival cooldowns left the chance of death at %v (was %v without them)", full.chanceOfDeath, bare.chanceOfDeath)
	}
}

// Demoralizing Shout takes attack power off the boss, so it pays only when
// the boss has some. The curated tank boss has none, which is why the
// rotation leaves it out; this keeps the engine honest about the other
// case (a boss with 320 attack power, the engine's own default target).
func TestDemoralizingShoutPaysOnlyAgainstABossWithAttackPower(t *testing.T) {
	shout := core.APLRotationFromJsonString(`{"type":"TypeAPL","priorityList":[
		{"action":{"condition":{"not":{"val":{"auraIsActive":{"sourceUnit":{"type":"CurrentTarget"},"auraId":{"spellId":11556,"rank":5}}}}},
			"castSpell":{"spellId":{"spellId":11556,"rank":5}}}}]}`)
	withShout := &proto.APLRotation{Type: tankRotation().Type, PriorityList: append(append([]*proto.APLListItem{}, shout.PriorityList...), tankRotation().PriorityList...)}

	for _, attackPower := range []float64{0, 320} {
		run := func(rotation *proto.APLRotation) fightResult {
			return runFight(t, fightConfig{talents: P1Talents, apl: rotation, boss: siteBoss, gear: harnessGear(), bossAttackPower: attackPower})
		}
		base, shouted := run(tankRotation()), run(withShout)
		t.Logf("boss attack power %3.0f: without %s", attackPower, base)
		t.Logf("boss attack power %3.0f: with    %s", attackPower, shouted)
		if attackPower == 0 && shouted.dtps < base.dtps*0.99 {
			t.Errorf("Demoralizing Shout cut DTPS %v -> %v against a boss with no attack power", base.dtps, shouted.dtps)
		}
		if attackPower > 0 && shouted.dtps >= base.dtps {
			t.Errorf("Demoralizing Shout did not cut DTPS (%v -> %v) against a boss with %v attack power", base.dtps, shouted.dtps, attackPower)
		}
	}
}

// Every line of the curated rotation names a spell this engine registers
// for the tank at 60 - a line that does not resolve is a warning, and a
// rotation that quietly loses a line is not the one that was measured.
func TestEveryLineOfTheTankRotationResolves(t *testing.T) {
	war, _ := buildTank(t, P1Talents, true)
	registered := map[int32]bool{}
	for _, spell := range war.Spellbook {
		registered[spell.ActionID.SpellID] = true
	}
	for _, id := range []int32{
		spellShieldBlock, spellShieldWall, spellLastStand, spellThunderClap, spellSunderArmor,
		spellHeroicStrike, spellRevenge, spellShieldSlam, spellBloodrage, warrior.DefensiveStanceSpellId[0],
	} {
		if !registered[id] {
			t.Errorf("spell %d is not in the tank's spellbook", id)
		}
	}
	for _, item := range tankRotation().PriorityList {
		id := item.GetAction().GetCastSpell().GetSpellId().GetSpellId()
		if id != 0 && !registered[id] {
			t.Errorf("the rotation casts %d, which the tank does not have", id)
		}
	}
}
