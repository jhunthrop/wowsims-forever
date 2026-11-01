package core

import (
	"math"
	"testing"

	googleProto "google.golang.org/protobuf/proto"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Demonic Rune (item 12662, spell 16666): SpellEffect 694074 energizes 1200
// mana and 694075 deals 800 school damage to the user, both with variance
// 0.5 ("Restores 900 to 1500 mana at the cost of 600 to 1000 life"). The
// health half takes its mean, so using a rune draws no extra random number.
func TestDemonicRuneCostsItsUserHealth(t *testing.T) {
	target := googleProto.Clone(DefaultTargetProtoLvl60).(*proto.Target)
	env, _, _ := NewEnvironment(
		SinglePlayerRaidProto(&proto.Player{
			Name:      "Rune Test",
			Race:      proto.Race_RaceHuman,
			Class:     proto.Class_ClassShaman,
			Spec:      &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{}},
			Equipment: &proto.EquipmentSpec{},
			Consumes:  &proto.Consumes{DefaultConjured: proto.Conjured_ConjuredDemonicRune},
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		&proto.Encounter{Duration: 180, Targets: []*proto.Target{target}},
		false,
	)
	sim := newSimWithEnv(env, &proto.SimOptions{RandomSeed: 1}, simsignals.CreateSignals())
	sim.reset()
	character := env.Raid.Parties[0].Players[0].GetCharacter()
	rune := character.GetSpell(ActionID{ItemID: 12662})
	if rune == nil {
		t.Fatal("the Demonic Rune is not registered")
	}
	before := character.CurrentHealth()
	rune.ApplyEffects(sim, &character.Unit, rune)
	if got := before - character.CurrentHealth(); math.Abs(got-demonicRuneHealthCost) > 1e-9 || demonicRuneHealthCost != 800 {
		t.Errorf("a Demonic Rune cost %v health (constant %v), want the client's mean 800", got, demonicRuneHealthCost)
	}
}
