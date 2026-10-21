package item_sets_test

import (
	"testing"

	engine "github.com/wowsims/classic/sim"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// The set models are class-agnostic, so each test wears a set on whichever
// class has the resource or the swing the model needs.

const (
	testLevel = 60

	// A synthetic one-hand main hand, so a proc per minute has a swing
	// speed to scale by. No real item carries this id.
	testWeaponID     int32 = 9_900_001
	testWeaponSpeed        = 2.0
	testWeaponDamage       = 10.0
)

type hostClass struct {
	class proto.Class
	race  proto.Race
	spec  any
}

var (
	mageHost = hostClass{proto.Class_ClassMage, proto.Race_RaceTroll,
		&proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{Armor: proto.Mage_Options_MoltenArmor}}}}
	warriorHost = hostClass{proto.Class_ClassWarrior, proto.Race_RaceOrc,
		&proto.Player_Warrior{Warrior: &proto.Warrior{Options: &proto.Warrior_Options{}}}}
	rogueHost = hostClass{proto.Class_ClassRogue, proto.Race_RaceHuman,
		&proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.RogueOptions{}}}}
)

// wornSet is a character wearing pieces of a client set, with the weapon.
type wornSet struct {
	sim       *core.Simulation
	character *core.Character
}

func (w wornSet) target() *core.Unit { return w.character.CurrentTarget }

func testWeapon() *proto.SimItem {
	return &proto.SimItem{
		Id:              testWeaponID,
		Name:            "Test sword",
		Type:            proto.ItemType_ItemTypeWeapon,
		WeaponType:      proto.WeaponType_WeaponTypeSword,
		HandType:        proto.HandType_HandTypeMainHand,
		WeaponDamageMin: testWeaponDamage,
		WeaponDamageMax: testWeaponDamage,
		WeaponSpeed:     testWeaponSpeed,
		Stats:           stats.Stats{}.ToFloatArray(),
	}
}

// wear builds a character of host in pieces of the client set, against
// the given targets (the default humanoid boss when there are none).
func wear(t *testing.T, host hostClass, setID int32, pieces int, targets ...*proto.Target) wornSet {
	t.Helper()
	engine.RegisterAll()
	gear, db := core.ClientSetTestGear(setID, pieces)
	gear.Items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: testWeaponID}
	db.Items = append(db.Items, testWeapon())
	if len(targets) == 0 {
		targets = []*proto.Target{core.DefaultTargetProtoLvl60}
	}

	player := core.WithSpec(&proto.Player{
		Name:      "Set model test",
		Race:      host.race,
		Class:     host.class,
		Level:     testLevel,
		Equipment: gear,
		Database:  db,
		Buffs:     core.FullBuffs.Player,
		Rotation:  &proto.APLRotation{},
	}, host.spec)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: targets},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	return wornSet{sim: sim, character: sim.Raid.Parties[0].Players[0].GetCharacter()}
}

// swing delivers a landed hit with the given proc mask to an aura's
// OnSpellHitDealt, as the engine would after a swing.
func (w wornSet) swing(aura *core.Aura, mask core.ProcMask) {
	w.swingAt(aura, mask, w.target())
}

func (w wornSet) swingAt(aura *core.Aura, mask core.ProcMask, target *core.Unit) {
	aura.OnSpellHitDealt(aura, w.sim, &core.Spell{ProcMask: mask},
		&core.SpellResult{Outcome: core.OutcomeHit, Target: target, Damage: 1})
}

// cast delivers a completed damage spell to an aura's OnCastComplete.
func (w wornSet) cast(aura *core.Aura) {
	aura.OnCastComplete(aura, w.sim, &core.Spell{ProcMask: core.ProcMaskSpellDamage})
}

// struck delivers damage taken from the target to an aura.
func (w wornSet) struck(aura *core.Aura) {
	aura.OnSpellHitTaken(aura, w.sim, &core.Spell{ProcMask: core.ProcMaskMeleeMHAuto},
		&core.SpellResult{Outcome: core.OutcomeHit, Target: &w.character.Unit, Damage: 1})
}
