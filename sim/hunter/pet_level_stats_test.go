package hunter

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

func newHunterPetAtLevel(t *testing.T, level int32) *HunterPet {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class: proto.Class_ClassHunter, Race: proto.Race_RaceOrc, Level: level,
		Equipment: &proto.EquipmentSpec{}, DistanceFromTarget: 25,
	}, P1PlayerOptions)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)
	built, ok := env.Raid.Parties[0].Players[0].(*Hunter)
	if !ok || built.pet == nil {
		t.Fatal("player 0 did not build as a *Hunter with a pet")
	}
	return built.pet
}

// Round 9 (physical lane): a hunter's pet is the hunter's level, and its
// base stats and weapon damage follow that level. The engine gave every pet
// its level-60 sheet. The per-level stats are Era's (no Forever table is in
// the client: Hunter Pet Scaling 415429's channels are server-computed):
// pet_levelstats entry 1 in mangoszero's and tortoise-wow's world databases
// (level 30: Strength 61, Agility 52, Stamina 115, Intellect 32, Spirit 44;
// level 60: 136, 100, 256, 50, 80). Weapon damage is proportional to the
// pet's level in vmangos' Pet::InitStatsForLevel (petlevel x 1.2075 to
// 1.5225 a 2 s swing) and linear in it in cmangos', so the level-60 roll
// scales by level / 60.
func TestAHunterPetsStatsAndDamageFollowItsLevel(t *testing.T) {
	p60, p30 := newHunterPetAtLevel(t, 60), newHunterPetAtLevel(t, 30)
	s30 := p30.GetStats()
	for stat, want := range map[stats.Stat]float64{stats.Strength: 61, stats.Agility: 52, stats.Intellect: 32, stats.Spirit: 44} {
		if got := s30[stat]; got != want {
			t.Errorf("level-30 pet %v = %v, want %v", stat, got, want)
		}
	}
	if got, want := s30[stats.Stamina], 274.0*115/256; math.Abs(got-want) > 1e-6 {
		t.Errorf("level-30 pet Stamina = %v, want %v", got, want)
	}
	if got := p60.GetStats()[stats.Strength]; got != 136 {
		t.Errorf("level-60 pet Strength = %v, want 136", got)
	}
	mh60, mh30 := p60.AutoAttacks.MH(), p30.AutoAttacks.MH()
	if math.Abs(mh30.BaseDamageMin-mh60.BaseDamageMin*0.5) > 1e-9 || math.Abs(mh30.BaseDamageMax-mh60.BaseDamageMax*0.5) > 1e-9 {
		t.Errorf("level-30 pet weapon %v to %v, want half of level 60's %v to %v", mh30.BaseDamageMin, mh30.BaseDamageMax, mh60.BaseDamageMin, mh60.BaseDamageMax)
	}
}
