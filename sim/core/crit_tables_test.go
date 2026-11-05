package core

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Forever unifies item crit, not the two crit tables. The 1.60.1.70707
// client keeps three crit auras apart: aura 52 is melee and ranged (weapon)
// crit, aura 57 is spell crit, aura 290 is both ("Increased Critical 1 -
// Melee and Ranged" 1318954, "Increased Critical 1 - Spell" 1318956,
// "Increased Critical 1 - All" 436239 are Forever's own rows), and a spell's
// SpellCategories.DefenseType decides which table it rolls: magic rolls
// spell crit, melee and ranged roll melee crit. Agility feeds melee crit and
// Intellect spell crit: on the beta logs (foreverlogs.gg, levels 13 to 30)
// rogue Instant Poison crit 9 of 409 hits (2.2%) while the same rogues'
// melee crit 19%, and a paladin's Consecration ticks crit 6.1% of 1762
// against its 12.4% melee crit.

// critTables is one character's crit chance on each table, in percent.
type critTables struct {
	melee, ranged, spell, magicRanged, healing float64
}

// critTablesFor builds one level 60 player with the given consumes and
// buffs and reads the crit chance of a melee strike, a ranged shot, a
// fire spell, a magic-school spell with a ranged defense type (Serpent
// Sting, 13552, DefenseType 3) and a heal.
func critTablesFor(t *testing.T, class proto.Class, consumes *proto.Consumes, raidBuffs *proto.RaidBuffs) critTables {
	t.Helper()
	return critTablesWith(t, class, consumes, raidBuffs, nil)
}

// critTablesWith is critTablesFor with setup run on the character before
// its stats are finalized (a set bonus's or a talent's effect).
func critTablesWith(t *testing.T, class proto.Class, consumes *proto.Consumes, raidBuffs *proto.RaidBuffs, setup func(*Character)) critTables {
	t.Helper()
	// The fake agent this package registers (dot_test.go); the class
	// decides the base stats and the per-level crit rates.
	player := &proto.Player{
		Name:      "Crit Tables",
		Race:      proto.Race_RaceOrc,
		Class:     class,
		Consumes:  consumes,
		Buffs:     &proto.IndividualBuffs{},
		Spec:      &proto.Player_ElementalShaman{},
		Equipment: &proto.EquipmentSpec{},
	}
	raidProto := SinglePlayerRaidProto(player, &proto.PartyBuffs{}, raidBuffs, &proto.Debuffs{})
	encounterProto := &proto.Encounter{
		Targets:  []*proto.Target{{Name: "target", Level: 60}},
		Duration: 180,
	}
	env := &Environment{State: Created}
	env.construct(raidProto, encounterProto)
	character := env.Raid.Parties[0].Players[0].GetCharacter()
	// The fake agent wires neither crit dependency nor a weapon; a class
	// constructor does both.
	AddCritStatDependencies(character, class)
	character.Equipment[proto.ItemSlot_ItemSlotMainHand] = Item{
		ID:              1,
		WeaponType:      proto.WeaponType_WeaponTypeSword,
		HandType:        proto.HandType_HandTypeMainHand,
		WeaponDamageMin: 50,
		WeaponDamageMax: 100,
		SwingSpeed:      2,
	}
	raidStats := env.initialize(raidProto, encounterProto)
	caster := &character.Unit
	register := func(id int32, school SpellSchool, defense DefenseType, cast proto.CastType, mask ProcMask) *Spell {
		return caster.RegisterSpell(SpellConfig{
			ActionID:         ActionID{SpellID: id},
			SpellSchool:      school,
			DefenseType:      defense,
			CastType:         cast,
			ProcMask:         mask,
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
		})
	}
	melee := register(1, SpellSchoolPhysical, DefenseTypeMelee, proto.CastType_CastTypeMainHand, ProcMaskMeleeMHSpecial)
	ranged := register(2, SpellSchoolPhysical, DefenseTypeRanged, proto.CastType_CastTypeRanged, ProcMaskRangedSpecial)
	fire := register(3, SpellSchoolFire, DefenseTypeMagic, proto.CastType_CastTypeMainHand, ProcMaskSpellDamage)
	sting := register(4, SpellSchoolNature, DefenseTypeRanged, proto.CastType_CastTypeRanged, ProcMaskRangedSpecial)
	heal := register(5, SpellSchoolHoly, DefenseTypeMagic, proto.CastType_CastTypeMainHand, ProcMaskSpellHealing)
	if setup != nil {
		setup(character)
	}
	env.finalize(raidProto, encounterProto, raidStats, false)

	target := env.Encounter.TargetUnits[0]
	table := func(spell *Spell) *AttackTable {
		return caster.AttackTables[target.UnitIndex][spell.CastType]
	}
	return critTables{
		melee:       100 * melee.PhysicalCritChance(table(melee)),
		ranged:      100 * ranged.PhysicalCritChance(table(ranged)),
		spell:       100 * fire.SpellCritChance(target),
		magicRanged: 100 * sting.SpellCritChance(target),
		healing:     100 * heal.HealingCritChance(),
	}
}

func (c critTables) minus(o critTables) critTables {
	return critTables{c.melee - o.melee, c.ranged - o.ranged, c.spell - o.spell, c.magicRanged - o.magicRanged, c.healing - o.healing}
}

func assertCritDelta(t *testing.T, name string, got, want critTables) {
	t.Helper()
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	if !near(got.melee, want.melee) || !near(got.ranged, want.ranged) || !near(got.spell, want.spell) ||
		!near(got.magicRanged, want.magicRanged) || !near(got.healing, want.healing) {
		t.Errorf("%s moves crit by %+v, want %+v", name, got, want)
	}
}

func noConsumes() *proto.Consumes { return &proto.Consumes{} }

// Elemental Sharpening Stone (enchant 2506, spell 22755: aura 52, 2) is
// weapon crit: melee only (the engine already takes it back off ranged),
// never a spell or a heal.
func TestSharpeningStoneIsMeleeCritOnly(t *testing.T) {
	base := critTablesFor(t, proto.Class_ClassShaman, noConsumes(), &proto.RaidBuffs{})
	stoned := critTablesFor(t, proto.Class_ClassShaman,
		&proto.Consumes{MainHandImbue: proto.WeaponImbue_ElementalSharpeningStone}, &proto.RaidBuffs{})
	assertCritDelta(t, "Elemental Sharpening Stone", stoned.minus(base), critTables{melee: 2})
}

// Brilliant Wizard Oil (enchant 2628, spell 25113: aura 57, 1) is spell
// crit only.
func TestBrilliantWizardOilIsSpellCritOnly(t *testing.T) {
	base := critTablesFor(t, proto.Class_ClassShaman, noConsumes(), &proto.RaidBuffs{})
	oiled := critTablesFor(t, proto.Class_ClassShaman,
		&proto.Consumes{MainHandImbue: proto.WeaponImbue_BrilliantWizardOil}, &proto.RaidBuffs{})
	assertCritDelta(t, "Brilliant Wizard Oil", oiled.minus(base), critTables{spell: 1, healing: 1})
}

// Elixir of the Mongoose (17538) is 25 Agility and aura 290, 2: the 2%
// reaches every table, the Agility only melee and ranged.
func TestMongooseCritReachesEveryTableItsAgilityOnlyMelee(t *testing.T) {
	base := critTablesFor(t, proto.Class_ClassShaman, noConsumes(), &proto.RaidBuffs{})
	mongoose := critTablesFor(t, proto.Class_ClassShaman,
		&proto.Consumes{AgilityElixir: proto.AgilityElixir_ElixirOfTheMongoose}, &proto.RaidBuffs{})
	agi := 25 * CritPerAgiAtLevel(proto.Class_ClassShaman, 60)
	assertCritDelta(t, "Elixir of the Mongoose", mongoose.minus(base),
		critTables{melee: 2 + agi, ranged: 2 + agi, spell: 2, magicRanged: 2 + agi, healing: 2})
}

// Intellect feeds spell crit only (Arcane Brilliance-style Intellect: the
// Runn Tum Tuber Surprise's 15).
func TestIntellectIsSpellCritOnly(t *testing.T) {
	base := critTablesFor(t, proto.Class_ClassShaman, noConsumes(), &proto.RaidBuffs{})
	fed := critTablesFor(t, proto.Class_ClassShaman,
		&proto.Consumes{Food: proto.Food_FoodRunnTumTuberSurprise}, &proto.RaidBuffs{})
	intel := 15 * SpellCritPerIntAtLevel(proto.Class_ClassShaman, 60)
	if intel == 0 {
		t.Fatal("the shaman has no Intellect crit rate at 60")
	}
	assertCritDelta(t, "15 Intellect", fed.minus(base), critTables{spell: intel, healing: intel})
}

// Leader of the Pack (24932) and Moonkin Aura (24907) are aura 290 in
// Forever: every table. Blizzard's notes of 3 November 2026 make them
// "again exclusive" with each other, so the two together give 3%, once.
func TestLeaderOfThePackAndMoonkinAuraReachEveryTable(t *testing.T) {
	base := critTablesFor(t, proto.Class_ClassShaman, noConsumes(), &proto.RaidBuffs{})
	every3 := critTables{melee: 3, ranged: 3, spell: 3, magicRanged: 3, healing: 3}
	for name, buffs := range map[string]*proto.RaidBuffs{
		"Leader of the Pack":                  {LeaderOfThePack: true},
		"Moonkin Aura":                        {MoonkinAura: true},
		"Leader of the Pack and Moonkin Aura": {LeaderOfThePack: true, MoonkinAura: true},
	} {
		buffed := critTablesFor(t, proto.Class_ClassShaman, noConsumes(), buffs)
		assertCritDelta(t, name, buffed.minus(base), every3)
	}
}

// A rogue has no spell crit of its own (Era's base 0 and no Intellect
// rate), so its poisons crit only from all-table sources.
func TestRogueAgilityDoesNotReachPoisons(t *testing.T) {
	base := critTablesFor(t, proto.Class_ClassRogue, noConsumes(), &proto.RaidBuffs{})
	if base.spell != 0 {
		t.Errorf("a bare rogue's spell crit is %v%%, want 0", base.spell)
	}
	if base.melee <= 0 {
		t.Errorf("a bare rogue's melee crit is %v%%, want its Agility crit", base.melee)
	}
}

// The client's flat-bonus decoder (set bonuses, client effects) keeps the
// three auras apart.
func TestClientFlatBonusKeepsCritTablesApart(t *testing.T) {
	for _, c := range []struct {
		name    string
		effects []ClientEffect
		want    stats.Stats
	}{
		{"aura 52 (Increased Critical 2, 7598)", []ClientEffect{{Effect: 6, Aura: 52, Points: 2}}, stats.Stats{stats.MeleeCrit: 2}},
		{"aura 57 (Increased Critical Spell, 18382)", []ClientEffect{{Effect: 6, Aura: 57, Points: 2}}, stats.Stats{stats.SpellCrit: 2}},
		{"aura 290 (Increased Critical 1 - All, 436239)", []ClientEffect{{Effect: 6, Aura: 290, Points: 1}}, stats.Stats{stats.Crit: 1}},
		{"aura 52 and 57 (Increased Critical 1%, 1220596)", []ClientEffect{{Effect: 6, Aura: 52, Points: 1}, {Effect: 6, Aura: 57, Misc0: 126, Points: 1}}, stats.Stats{stats.Crit: 1}},
	} {
		got, ok := DecodeClientFlatBonus(ClientSpell{Name: c.name, Effects: c.effects})
		if !ok {
			t.Errorf("%s does not decode", c.name)
			continue
		}
		if got.Stats != c.want {
			t.Errorf("%s decodes to %v, want %v", c.name, got.Stats, c.want)
		}
	}
}

// A school's spell crit (PseudoStats.SchoolBonusCritChance: Vestments of
// Prophecy 5-piece "Holy spells", The Ten Storms 5-piece "Nature spells",
// both aura 57 in the 70707 client) reaches that school's heals, which roll
// the spell table, as well as its damage spells.
func TestSchoolSpellCritReachesThatSchoolsHeals(t *testing.T) {
	base := critTablesFor(t, proto.Class_ClassShaman, noConsumes(), &proto.RaidBuffs{})
	holy := critTablesWith(t, proto.Class_ClassShaman, noConsumes(), &proto.RaidBuffs{}, func(c *Character) {
		c.PseudoStats.SchoolBonusCritChance[stats.SchoolIndexHoly] += 2 * CritRatingPerCritChance
	})
	assertCritDelta(t, "2% Holy spell crit", holy.minus(base), critTables{healing: 2})
}
