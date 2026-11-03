package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Round 8 (auras lane): the raid's debuffs are cast at the raid's level, so
// each takes the rank that level has learned (1.60.1.70707 SpellEffect with
// the hotfix push, SpellLevels). The engine applied every one at its top
// rank whatever the level.

// newDebuffTarget is a target unit whose raid is one player of level.
func newDebuffTarget(level int32) *Unit {
	return &Unit{
		Type:        EnemyUnit,
		Level:       63,
		auraTracker: newAuraTracker(),
		Env: &Environment{
			MeasuringStats: true,
			Raid:           &Raid{AllPlayerUnits: []*Unit{{Type: PlayerUnit, Level: level}}},
		},
	}
}

func TestRaidDebuffsAreCastAtTheRaidsLevel(t *testing.T) {
	if got := raidDebuffLevel(newDebuffTarget(34)); got != 34 {
		t.Errorf("raid debuff level = %d, want the raid's 34", got)
	}
	if got := raidDebuffLevel(&Unit{}); got != CharacterMaxLevel {
		t.Errorf("raid debuff level with no raid = %d, want %d", got, CharacterMaxLevel)
	}
}

// Sunder Armor 7386, 7405, 8380, 11596, 11597 (levels 10, 22, 34, 46, 58):
// 90, 180, 270, 360 and 450 armor a stack (aura 674, misc 1).
// Faerie Fire 770, 778, 9749, 9907 (18, 30, 42, 54): 175, 285, 395, 505.
// Curse of Recklessness 704, 7658, 7659, 11717 (14, 28, 42, 56): 175, 285,
// 395, 505 armor, and no attack power (its other effect is aura 92, the
// no-flee). Curse of the Elements 440892, 1311676, 1311677, 1311680 (20,
// 30, 40, 50): 4, 6, 8, 10% magic damage taken and 30, 45, 60, 75
// resistance. Hunter's Mark 1130, 14323, 14324, 14325 (6, 22, 40, 58): 26,
// 59, 98, 143 ranged attack power. Judgement of the Crusader 21183, 20188,
// 20300, 20301, 20302, 20303 (Seal levels 6, 12, 22, 32, 42, 52): 23, 35,
// 58, 92, 127, 161 Holy damage taken.
func TestRaidDebuffsFollowTheClientRanks(t *testing.T) {
	checkRanks(t, "Sunder Armor", SunderArmorRanks, []levelWant{{9, 0}, {10, 90}, {22, 180}, {34, 270}, {46, 360}, {57, 360}, {58, 450}, {60, 450}})
	checkRanks(t, "Faerie Fire", FaerieFireRanks, []levelWant{{17, 0}, {18, 175}, {30, 285}, {42, 395}, {53, 395}, {54, 505}, {60, 505}})
	checkRanks(t, "Curse of Recklessness", CurseOfRecklessnessRanks, []levelWant{{13, 0}, {14, 175}, {28, 285}, {42, 395}, {55, 395}, {56, 505}, {60, 505}})
	checkRanks(t, "Curse of the Elements damage", CurseOfTheElementsDamageRanks, []levelWant{{19, 0}, {20, 4}, {30, 6}, {40, 8}, {50, 10}, {60, 10}})
	checkRanks(t, "Curse of the Elements resistance", CurseOfTheElementsResistanceRanks, []levelWant{{19, 0}, {20, 30}, {30, 45}, {40, 60}, {50, 75}, {60, 75}})
	checkRanks(t, "Hunter's Mark", HuntersMarkRanks, []levelWant{{5, 0}, {6, 26}, {22, 59}, {40, 98}, {57, 98}, {58, 143}, {60, 143}})
	checkRanks(t, "Judgement of the Crusader", JudgementOfTheCrusaderRanks, []levelWant{{5, 0}, {6, 23}, {12, 35}, {22, 58}, {32, 92}, {42, 127}, {51, 127}, {52, 161}, {60, 161}})
	if top := HuntersMarkRanks[len(HuntersMarkRanks)-1]; top.SpellID != HuntersMarkTopRankSpellID || top.Amount != HuntersMarkTopRankRangedAttackPower {
		t.Errorf("Hunter's Mark top rank = %+v, want %d at %v", top, HuntersMarkTopRankSpellID, HuntersMarkTopRankRangedAttackPower)
	}
	if top := JudgementOfTheCrusaderRanks[len(JudgementOfTheCrusaderRanks)-1]; top.Amount != JudgementOfTheCrusaderRaidBonus {
		t.Errorf("Judgement of the Crusader top rank = %v, want %v", top.Amount, float64(JudgementOfTheCrusaderRaidBonus))
	}
}

// Curse of Recklessness takes armor only: Forever's rows carry no attack
// power effect (Era's gave the target 90).
func TestCurseOfRecklessnessTakesTheRanksArmorOnly(t *testing.T) {
	for _, c := range []struct {
		level int32
		armor float64
	}{{60, 505}, {30, 285}} {
		target := newDebuffTarget(c.level)
		aura := CurseOfRecklessnessAura(target)
		aura.Activate(&Simulation{})
		if got := target.GetStat(stats.Armor); got != -c.armor {
			t.Errorf("level %d: armor after Curse of Recklessness = %v, want %v", c.level, got, -c.armor)
		}
		if got := target.GetStat(stats.AttackPower); got != 0 {
			t.Errorf("level %d: Curse of Recklessness gave the target %v attack power, want 0", c.level, got)
		}
	}
}

// Forever made four vanilla raid debuffs the caster's own: Shadow Weaving
// (15258), Improved Scorch's Fire Vulnerability (22959) and Improved Shadow
// Bolt's Shadow Vulnerability (17794) are aura 270, damage taken from the
// caster, and Stormstrike (17364) aura 271 on the shaman's own Lightning
// Bolt, Chain Lightning and Earth Shock (Wowhead Forever: "the Shadow damage
// the target takes from you", "all Fire damage you deal to your target",
// "the damage you deal to the target with your next Lightning Bolt"). A
// raid member's copy gives this player nothing, so the raid-debuff flags
// put no aura on the target.
func TestCasterOnlyDebuffsGiveTheRaidNothing(t *testing.T) {
	target := newDebuffTarget(60)
	applyDebuffEffects(target, 0, &proto.Debuffs{
		ShadowWeaving: true, ImprovedScorch: true, ImprovedShadowBolt: true, Stormstrike: true,
	}, &proto.Raid{})
	for _, label := range []string{"Shadow Weaving", "Improved Scorch", "Improved Shadow Bolt", "Stormstrike", "Isb External Proc Aura", "Stormstrike External Proc Aura"} {
		if target.GetAura(label) != nil {
			t.Errorf("the raid's %s put an aura on the target", label)
		}
	}
}

// Round 9 (casters lane): Forever removed Curse of Shadow. Its ranks 17862
// and 17937 have no SpellName or SpellEffect row in 1.60.1.70707 (with the
// hotfix push), and Wowhead Forever finds no such spell; Curse of the
// Elements (1311680) is the all-magic curse. The raid flag gave Era's +10%
// Shadow and Arcane damage taken and took 75 resistance off twice.
func TestTheRaidsCurseOfShadowIsGone(t *testing.T) {
	target := newDebuffTarget(60)
	applyDebuffEffects(target, 0, &proto.Debuffs{CurseOfShadow: true}, &proto.Raid{})
	if target.GetAura("Curse of Shadow") != nil {
		t.Error("the raid's Curse of Shadow put an aura on the target")
	}
}

// Winter's Chill (12579) is aura 271 in Forever: Frost damage taken from the
// applying mage, whose own next Cone of Cold reads and spends the stacks. The
// raid flag put five stacks on the boss under the mage's own aura label, so a
// Frost mage's Cone of Cold spent stacks its talent never applied (round 9).
func TestRaidWintersChillGivesNoOneStacks(t *testing.T) {
	target := newDebuffTarget(60)
	applyDebuffEffects(target, 0, &proto.Debuffs{WintersChill: true}, &proto.Raid{})
	if target.GetAura("Winter's Chill") != nil {
		t.Error("the raid's Winter's Chill put an aura on the target")
	}
}

// Thunder Clap's slow in Forever is the trained family's: 6343 to 11581
// (SkillLineAbility 6073 to 6078) all carry aura 319 at -20, "increasing
// the time between their attacks by 20%" (the 461810 reissue's -10 is not
// trained). The raid debuff had Era's 10% (15% with the 5-piece).
func TestThunderClapRaidDebuffSlowsTwentyPercent(t *testing.T) {
	for _, c := range []struct {
		effect proto.TristateEffect
		label  string
	}{{proto.TristateEffect_TristateEffectRegular, "ThunderClap-20"}, {proto.TristateEffect_TristateEffectImproved, "ThunderClap-30"}} {
		target := newDebuffTarget(60)
		applyDebuffEffects(target, 0, &proto.Debuffs{ThunderClap: c.effect}, &proto.Raid{})
		aura := target.GetAuraByID(ActionID{SpellID: 11581})
		if aura == nil {
			t.Fatalf("%v: no Thunder Clap 11581 on the target", c.effect)
		}
		if aura.Label != c.label {
			t.Errorf("%v: Thunder Clap debuff is %q, want %q", c.effect, aura.Label, c.label)
		}
	}
}
