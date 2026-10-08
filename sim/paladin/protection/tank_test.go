package protection

import (
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	googleProto "google.golang.org/protobuf/proto"
)

// harshBossMinDamage is a boss hitting about 1.7 times as hard as the
// published tank boss, hard enough that the healing model cannot keep a
// tank alive without its defensives, so what they buy shows.
const harshBossMinDamage = 4000

// The reference tank keeps up with the published boss for the whole
// fight: no death in 2000 iterations.
func TestTheReferenceTankSurvivesThePublishedBoss(t *testing.T) {
	got := runHarness(t, harnessConfig{})
	t.Logf("reference: %v\n%s\n%s\n%s", got, strings.Join(got.Actions, "\n"), strings.Join(got.Resources, "\n"), strings.Join(got.Auras, "\n"))

	if got.ChanceOfDeath != 0 {
		t.Errorf("the reference tank dies %.1f%% of the time against the published boss", 100*got.ChanceOfDeath)
	}
	if got.DTPS <= 0 || got.ThreatPerSec <= 0 {
		t.Errorf("DTPS %v and threat %v: the fight did not run", got.DTPS, got.ThreatPerSec)
	}
}

// A tank rotation makes more threat than the same character playing a
// damage rotation: Seal of Righteousness, Judgement, Holy Strike and
// Hammer of the Righteous with Righteous Fury off and no Holy Shield.
func TestTheTankRotationOutThreatsADamageRotation(t *testing.T) {
	const damageRotation = `{
		"type": "TypeAPL",
		"prepullActions": [{"action": {"castSpell": {"spellId": {"spellId": 20293}}}, "doAtValue": {"const": {"val": "-1.5s"}}}],
		"priorityList": [
			{"action": {"condition": {"cmp": {"op": "OpLe", "lhs": {"currentSealRemainingTime": {}}, "rhs": {"const": {"val": "2s"}}}}, "castSpell": {"spellId": {"spellId": 20293}}}},
			{"action": {"castSpell": {"spellId": {"spellId": 407632}}}},
			{"action": {"castSpell": {"spellId": {"spellId": 10333}}}},
			{"action": {"castSpell": {"spellId": {"spellId": 20271}}}}
		]}`
	tank := runHarness(t, harnessConfig{})
	damage := runHarness(t, harnessConfig{
		Rotation: core.APLRotationFromJsonString(damageRotation),
		Options:  &proto.PaladinOptions{},
	})
	t.Logf("tank %v\ndamage %v", tank, damage)

	if tank.ThreatPerSec <= 1.3*damage.ThreatPerSec {
		t.Errorf("the tank rotation makes %.0f threat a second against the damage rotation's %.0f, want at least 30%% more", tank.ThreatPerSec, damage.ThreatPerSec)
	}
	if tank.DTPS >= damage.DTPS {
		t.Errorf("the tank takes %.0f damage a second against the damage rotation's %.0f, want less", tank.DTPS, damage.DTPS)
	}
}

// What the rotation's lines buy, as the change when each is removed from
// the curated rotation. The figures are logged for the rotation notes; the
// assertions are the signs that must hold.
func TestRotationAblation(t *testing.T) {
	reference := runHarness(t, harnessConfig{})
	rotation := core.GetAplRotation("../../../ui/protection_paladin/apls", harnessRotation).Rotation

	for _, tc := range []struct {
		name         string
		spellID      int32
		threatLowers bool
		damageRises  bool
	}{
		{"Seal of Fury", spellSealOfFury, true, true},
		{"Holy Shield", spellHolyShield, true, true},
		{"Hammer of the Righteous", spellHammerRighteous, true, false},
		{"Holy Strike", spellHolyStrike, true, false},
		{"Judgement", spellJudgement, true, false},
		{"Swift Judgement", spellSwiftJudgement, true, false},
		{"Consecration", spellConsecration, true, false},
	} {
		got := runHarness(t, harnessConfig{Rotation: withoutSpell(rotation, tc.spellID)})
		t.Logf("without %-24s threat %+6.1f%%  DTPS %+6.1f%%", tc.name,
			100*(got.ThreatPerSec/reference.ThreatPerSec-1), 100*(got.DTPS/reference.DTPS-1))
		if tc.threatLowers && got.ThreatPerSec >= reference.ThreatPerSec {
			t.Errorf("removing %s raises threat from %.1f to %.1f", tc.name, reference.ThreatPerSec, got.ThreatPerSec)
		}
		if tc.damageRises && got.DTPS <= reference.DTPS {
			t.Errorf("removing %s lowers damage taken from %.1f to %.1f", tc.name, reference.DTPS, got.DTPS)
		}
	}
}

// The three defensives lower the chance of death against a boss the
// healing model cannot out-heal, which is what they are for.
func TestDefensiveCooldownsLowerTheChanceOfDeath(t *testing.T) {
	rotation := core.GetAplRotation("../../../ui/protection_paladin/apls", harnessRotation).Rotation
	withCooldowns := runHarness(t, harnessConfig{BossMinDamage: harshBossMinDamage})
	without := runHarness(t, harnessConfig{
		BossMinDamage: harshBossMinDamage,
		Rotation:      withoutSpell(withoutSpell(withoutSpell(rotation, spellTemplarsBulwark), spellDivineShield), spellDivineProtection),
	})
	t.Logf("with defensives %v\nwithout %v", withCooldowns, without)

	if withCooldowns.ChanceOfDeath >= without.ChanceOfDeath {
		t.Errorf("chance of death %.3f with the defensives, %.3f without", withCooldowns.ChanceOfDeath, without.ChanceOfDeath)
	}
	if withCooldowns.DTPS >= without.DTPS {
		t.Errorf("damage taken %.1f with the defensives, %.1f without", withCooldowns.DTPS, without.DTPS)
	}
}

// withoutSpell is the rotation with every priority line that casts the
// spell removed.
func withoutSpell(rotation *proto.APLRotation, spellID int32) *proto.APLRotation {
	trimmed := googleProto.Clone(rotation).(*proto.APLRotation)
	kept := trimmed.PriorityList[:0]
	for _, item := range trimmed.PriorityList {
		if cast := item.GetAction().GetCastSpell(); cast != nil && cast.SpellId.GetSpellId() == spellID {
			continue
		}
		kept = append(kept, item)
	}
	trimmed.PriorityList = kept
	return trimmed
}
