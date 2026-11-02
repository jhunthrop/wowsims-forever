package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Round 9 (physical lane): Mortal Shots (19485, aura 108 misc 15, crit
// damage bonus) carries class mask 424449 / ..., and mask 0 bit 16384 is
// Serpent Sting (13555's SpellClassOptions). Forever's DoTs crit (owner,
// 2026-10-31), so the sting's tick crits take the talent's bonus like the
// shots do; the engine left it out.
func TestMortalShotsReachesSerpentSting(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			TalentsString:      "5320000501000000-0051550001503050-500000000000000000",
			DistanceFromTarget: 25,
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)
	built, ok := env.Raid.Parties[0].Players[0].(*Hunter)
	if !ok {
		t.Fatal("player 0 did not build as a *Hunter")
	}
	if built.mortalShots() <= 0 {
		t.Fatalf("the test build takes no Mortal Shots (%v)", built.mortalShots())
	}
	if built.SerpentSting == nil || built.ArcaneShot == nil {
		t.Fatal("no Serpent Sting or Arcane Shot registered")
	}
	want := built.ArcaneShot.CritDamageBonus
	if got := built.SerpentSting.CritDamageBonus; got != want {
		t.Errorf("Serpent Sting crit damage bonus = %v, want Arcane Shot's (Mortal Shots) %v", got, want)
	}
}
