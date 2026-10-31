package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// survivalMeleeStrikes are the strikes a Forever Survival hunter fights
// with. Client masks: Raptor Strike and Mongoose Bite bit 1 of word 0,
// Wing Clip bit 6, Counterattack bit 19 of word 1, Strider Kick bit 21 of
// word 2.
func survivalMeleeStrikes(hunter *Hunter) map[string]*core.Spell {
	return map[string]*core.Spell{
		"Raptor Strike": hunter.RaptorStrike,
		"Mongoose Bite": hunter.MongooseBite,
		"Wing Clip":     hunter.WingClip,
		"Counterattack": hunter.Counterattack,
		"Strider Kick":  hunter.StriderKick,
	}
}

// Mortal Shots (19485) is "critical strike damage bonus on all ranged
// abilities"; its class mask (424449, -2147483648, 268435968, 8388608)
// covers the shots and none of the melee strikes.
func TestMortalShotsLeavesMeleeStrikesAlone(t *testing.T) {
	with := buildHunterForTalentTest(t, 60, "mortal_shots", 5, proto.Hunter_Options_PetNone)
	without := buildHunterForTalentTest(t, 60, "mortal_shots", 0, proto.Hunter_Options_PetNone)
	for name, spell := range survivalMeleeStrikes(with) {
		if spell == nil {
			continue
		}
		base := survivalMeleeStrikes(without)[name]
		if spell.CritDamageBonus != base.CritDamageBonus {
			t.Errorf("%s crit damage bonus %.2f with Mortal Shots 5, want %.2f as without it", name, spell.CritDamageBonus, base.CritDamageBonus)
		}
	}
	if with.MultiShot.CritDamageBonus <= without.MultiShot.CritDamageBonus {
		t.Error("Mortal Shots no longer raises Multi-Shot's crit damage")
	}
}

// Efficiency (19416) reduces the cost of shots, stings and the melee
// strikes in its class mask: Raptor Strike, Mongoose Bite, Wing Clip and
// Counterattack (not Strider Kick). The talent text names "melee
// abilities".
func TestEfficiencyCoversMeleeStrikes(t *testing.T) {
	with := buildHunterForTalentTest(t, 60, "efficiency", 5, proto.Hunter_Options_PetNone)
	without := buildHunterForTalentTest(t, 60, "efficiency", 0, proto.Hunter_Options_PetNone)
	for _, name := range []string{"Raptor Strike", "Mongoose Bite", "Wing Clip", "Counterattack"} {
		spell, base := survivalMeleeStrikes(with)[name], survivalMeleeStrikes(without)[name]
		if spell == nil || spell.Cost == nil {
			continue
		}
		if spell.Cost.Multiplier >= base.Cost.Multiplier {
			t.Errorf("%s cost multiplier %d with Efficiency 5, want below %d", name, spell.Cost.Multiplier, base.Cost.Multiplier)
		}
	}
}

// Rapid Fire (3045) is aura 140 (ranged haste 40) plus aura 319 (melee
// haste 40): "Increases ranged and melee attack speed by 40%".
func TestRapidFireHastesMeleeToo(t *testing.T) {
	sim, hunter, _ := newRunningHunterForTalentTest(t, 60, "mortal_shots", 0, proto.Hunter_Options_PetNone, 5, nil)
	before := hunter.PseudoStats.MeleeSpeedMultiplier
	hunter.RapidFireAura.Activate(sim)
	if got := hunter.PseudoStats.MeleeSpeedMultiplier / before; !floatsClose(got, 1.4) {
		t.Errorf("Rapid Fire multiplies melee speed by %.3f, want 1.4", got)
	}
}
