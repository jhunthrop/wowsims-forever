package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

func newWarlockAtLevel(t *testing.T, level int32) *Warlock {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:     proto.Class_ClassWarlock,
			Race:      proto.Race_RaceOrc,
			Level:     level,
			Equipment: &proto.EquipmentSpec{},
		},
		&proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.WarlockOptions{
			Armor:  proto.WarlockOptions_NoArmor,
			Summon: proto.WarlockOptions_Succubus,
		}}},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)
	agent, ok := env.Raid.Parties[0].Players[0].(WarlockAgent)
	if !ok {
		t.Fatal("player 0 did not build as a warlock")
	}
	return agent.GetWarlock()
}

// The site publishes warlock bands at levels 20 and 30, but the demons' base
// sheets exist only at 25, 40, 50 and 60, so a level-20 or level-30 demon had
// no Strength, Stamina, mana or weapon: it never swung and could not cast.
// Between and below the anchors the sheet is interpolated by level.
func TestDemonsHaveASheetAtEveryBandLevel(t *testing.T) {
	for _, level := range []int32{20, 30} {
		warlock := newWarlockAtLevel(t, level)
		for _, pet := range warlock.BasePets {
			got := pet.GetStats()
			if got[stats.Stamina] <= 0 || got[stats.Mana] <= 0 {
				t.Errorf("level %d %s: Stamina %.1f, Mana %.1f, want a sheet", level, pet.Name, got[stats.Stamina], got[stats.Mana])
			}
			if pet != warlock.Imp && pet.AutoAttacks.MH().BaseDamageMax <= 0 {
				t.Errorf("level %d %s has no melee weapon", level, pet.Name)
			}
		}
	}
	// Level 30 lies a third of the way from the level-25 Succubus (Strength
	// 50, weapon 23-38) to the level-40 one (74, 41-61).
	succubus := newWarlockAtLevel(t, 30).Succubus
	if got := succubus.GetStats()[stats.Strength]; got != 58 {
		t.Errorf("level 30 Succubus Strength = %.2f, want 58", got)
	}
	if mh := succubus.AutoAttacks.MH(); mh.BaseDamageMin != 29 || mh.BaseDamageMax != 45.666666666666664 {
		t.Errorf("level 30 Succubus weapon = %.2f-%.2f, want 29-45.67", mh.BaseDamageMin, mh.BaseDamageMax)
	}
}

// A demon casts the highest rank its level has learned: Firebolt ranks 3
// (level 18) and 4 (28), Lash of Pain ranks 1 (20) and 2 (28). The old
// level-to-rank map knew only 25, 40, 50 and 60 and fell back to rank 1.
func TestDemonsCastTheirTopRankAtEveryBandLevel(t *testing.T) {
	for _, tc := range []struct {
		level          int32
		firebolt, lash int
	}{{20, 3, 1}, {30, 4, 2}, {40, 5, 3}, {50, 6, 4}, {60, 7, 6}} {
		warlock := newWarlockAtLevel(t, tc.level)
		if got := warlock.Imp.primaryAbility.Rank; got != tc.firebolt {
			t.Errorf("level %d Firebolt rank = %d, want %d", tc.level, got, tc.firebolt)
		}
		if got := warlock.Succubus.primaryAbility.Rank; got != tc.lash {
			t.Errorf("level %d Lash of Pain rank = %d, want %d", tc.level, got, tc.lash)
		}
	}
}

// A carried Succubus sheet swings a weapon of its own level. The level-50
// sheet used to repeat the level-40 weapon (41-61, "Not updated" upstream),
// so a level-50 Succubus hit no harder than a level-40 one and then jumped
// to 95-131 at 60 (sim-accuracy round 20).
func TestSuccubusWeaponGrowsBetweenCarriedSheets(t *testing.T) {
	for i := 1; i < len(petSheetLevels); i++ {
		lo, hi := succubusSheet(petSheetLevels[i-1]), succubusSheet(petSheetLevels[i])
		if hi.AutoAttacks.MainHand.BaseDamageMin <= lo.AutoAttacks.MainHand.BaseDamageMin ||
			hi.AutoAttacks.MainHand.BaseDamageMax <= lo.AutoAttacks.MainHand.BaseDamageMax {
			t.Errorf("Succubus weapon at %d (%.0f-%.0f) is no better than at %d (%.0f-%.0f)",
				petSheetLevels[i], hi.AutoAttacks.MainHand.BaseDamageMin, hi.AutoAttacks.MainHand.BaseDamageMax,
				petSheetLevels[i-1], lo.AutoAttacks.MainHand.BaseDamageMin, lo.AutoAttacks.MainHand.BaseDamageMax)
		}
	}
}
