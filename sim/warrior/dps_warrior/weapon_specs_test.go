package dpswarrior

import (
	"sort"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/warrior"
)

// Neither weapon-specialization talent is in ForeverFuryTalents, so no
// golden covers either, and both were inert in ways a reader of the code
// would not see:
//
//   - Two-Handed Weapon Specialization was a SpellMod over
//     WarriorSpellMaskSpecials|WarriorSpellMaskOnNextSwing. Auto-attacks
//     carry no ClassSpellMask, so a 2H warrior's largest damage source
//     took nothing from a talent whose tooltip reads "+3% damage with
//     two-handed melee weapons".
//   - Dual Wield Specialization guarded its off-hand damage multiplier
//     with `if spell.BonusCoefficient > 0`, an opaque proxy for "is this
//     a physical swing". It reached the off-hand auto after all — the
//     coefficient is 1 for a physical auto and 0 for a non-physical one
//     (core/attack.go:494) — so the talent was not inert, but nothing
//     pinned that and the line reads as though it excluded its own
//     subject. The guard is now the school check it stood for, and this
//     test is what holds the +25% in place.
//
// Both are asserted against a built character here rather than by
// reading the talent body back.

func TestTwoHandedWeaponSpecializationCoversWhiteSwings(t *testing.T) {
	twoHander := weaponWithHandType(t, proto.HandType_HandTypeTwoHand)

	// Three points is the talent's max rank: "+3% damage with
	// two-handed melee weapons".
	const points = 3
	const want = 1.03

	with := buildWarriorWithWeapons(t, talentStringWithRank(t, emptyWarriorTalents, "two_handed_weapon_specialization", points), twoHander, 0)
	without := buildWarriorWithWeapons(t, emptyWarriorTalents, twoHander, 0)

	got := with.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical]
	base := without.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical]
	if got != base*want {
		t.Errorf("with a two-hander and %d points the Physical damage multiplier is %v, want %v", points, got, base*want)
	}

	// The white swing is the point of the fix: a mask-based mod reached
	// it not at all, and a school multiplier must.
	mh := with.AutoAttacks.MHAuto()
	if mh == nil {
		t.Fatal("the warrior has no main-hand auto-attack spell")
	}
	if mh.ClassSpellMask != 0 {
		t.Fatalf("the main-hand auto-attack now carries ClassSpellMask %d; if autos are masked, a spell mod would reach them and this talent can go back to being one", mh.ClassSpellMask)
	}

	// A one-hander takes nothing: the hand-type guard is the only thing
	// narrowing a school multiplier to the tooltip's weapons.
	oneHander := weaponWithHandType(t, proto.HandType_HandTypeOneHand)
	oneHanded := buildWarriorWithWeapons(t, talentStringWithRank(t, emptyWarriorTalents, "two_handed_weapon_specialization", points), oneHander, 0)
	oneHandedBase := buildWarriorWithWeapons(t, emptyWarriorTalents, oneHander, 0)
	if got, base := oneHanded.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical], oneHandedBase.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical]; got != base {
		t.Errorf("with a one-hander the talent changed the Physical multiplier from %v to %v; it must not apply", base, got)
	}
}

func TestDualWieldSpecializationMultipliesTheOffHandAuto(t *testing.T) {
	oneHander := weaponWithHandType(t, proto.HandType_HandTypeOneHand)

	// Five points is the talent's max rank: "+25% off-hand weapon
	// damage", so 5% a point.
	const points = 5
	const want = 1.25

	with := buildWarriorWithWeapons(t, talentStringWithRank(t, emptyWarriorTalents, "dual_wield_specialization", points), oneHander, oneHander)
	without := buildWarriorWithWeapons(t, emptyWarriorTalents, oneHander, oneHander)

	oh, ohBase := with.AutoAttacks.OHAuto(), without.AutoAttacks.OHAuto()
	if oh == nil || ohBase == nil {
		t.Fatal("the dual-wielding warrior has no off-hand auto-attack spell")
	}
	if !oh.SpellSchool.Matches(core.SpellSchoolPhysical) {
		t.Fatalf("the off-hand auto-attack's school is %v, not Physical; the talent's school guard would skip it", oh.SpellSchool)
	}
	if got, base := oh.DamageMultiplier, ohBase.DamageMultiplier; got != base*want {
		t.Errorf("the off-hand auto-attack's DamageMultiplier is %v with %d points, want %v", got, points, base*want)
	}

	// The main hand is untouched: the talent names the off hand only.
	if got, base := with.AutoAttacks.MHAuto().DamageMultiplier, without.AutoAttacks.MHAuto().DamageMultiplier; got != base {
		t.Errorf("the main-hand auto-attack's DamageMultiplier moved from %v to %v; Dual Wield Specialization is an off-hand talent", base, got)
	}
}

// weaponWithHandType returns the lowest-numbered weapon in the loaded
// item database with the given hand type, so the test picks the same
// item on every run without pinning an id that a database refresh could
// retire.
func weaponWithHandType(t *testing.T, handType proto.HandType) int32 {
	t.Helper()

	var ids []int32
	for id, item := range core.ItemsByID {
		if item.Type == proto.ItemType_ItemTypeWeapon && item.HandType == handType && item.WeaponDamageMax > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		t.Fatalf("no weapon with hand type %v in the item database; this test needs --tags=with_db", handType)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids[0]
}

// buildWarriorWithWeapons stands up one warrior through the shipping
// agent factory with the given weapons equipped. An off-hand id of 0
// leaves the slot empty.
func buildWarriorWithWeapons(t *testing.T, talents string, mainHand, offHand int32) *warrior.Warrior {
	t.Helper()

	items := []*proto.ItemSpec{}
	for i := 0; i < int(proto.ItemSlot_ItemSlotRanged)+1; i++ {
		items = append(items, &proto.ItemSpec{})
	}
	items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: mainHand}
	if offHand != 0 {
		items[proto.ItemSlot_ItemSlotOffHand] = &proto.ItemSpec{Id: offHand}
	}

	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{
			Parties: []*proto.Party{{
				Players: []*proto.Player{{
					Name:          "Warrior",
					Class:         proto.Class_ClassWarrior,
					Race:          proto.Race_RaceOrc,
					TalentsString: talents,
					Consumes:      &proto.Consumes{},
					Buffs:         &proto.IndividualBuffs{},
					Spec:          PlayerOptionsFury,
					Equipment:     &proto.EquipmentSpec{Items: items},
				}},
				Buffs: &proto.PartyBuffs{},
			}},
		},
		Encounter: &proto.Encounter{
			Targets:  []*proto.Target{{Name: "target", Level: 63, MobType: proto.MobType_MobTypeDemon}},
			Duration: 60,
		},
	}, simsignals.CreateSignals())
	sim.Reset()

	agent, ok := sim.Raid.Parties[0].Players[0].(warrior.WarriorAgent)
	if !ok {
		t.Fatalf("the raid's first player is not a warrior agent")
	}
	return agent.GetWarrior()
}
