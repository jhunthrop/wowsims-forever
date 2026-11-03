package core

// The raid debuffs by level. A raid debuff has no caster in the sim, so it
// is cast at the raid's level (raidDebuffLevel), and each takes the highest
// rank that level has learned, as the raid buffs do (buff_ranks.go). The
// class casts (a warrior's Sunder Armor, a druid's Faerie Fire, a warlock's
// Curse of Recklessness) share these auras, so they follow the same ranks.
// Amounts are the client's (1.60.1.70707 SpellEffect with the hotfix push;
// SpellLevels for the level); the site's sim/leveling/buff_ranks_client_test.go
// pins the rows against the CSVs.
var (
	// Sunder Armor: armor removed per stack (aura 674, misc 1).
	SunderArmorRanks = BuffRanks{
		{SpellID: 7386, Level: 10, Amount: 90},
		{SpellID: 7405, Level: 22, Amount: 180},
		{SpellID: 8380, Level: 34, Amount: 270},
		{SpellID: 11596, Level: 46, Amount: 360},
		{SpellID: 11597, Level: 58, Amount: 450},
	}

	// Expose Armor: armor removed per combo point. Forever's rows carry
	// aura 674 at 0 (server-computed); the amounts are the Wowhead Forever
	// tooltips, Sunder Armor's per rank (Era's rank 5 was 340 a point).
	ExposeArmorRanks = BuffRanks{
		{SpellID: 8647, Level: 14, Amount: 90},
		{SpellID: 8649, Level: 26, Amount: 180},
		{SpellID: 8650, Level: 36, Amount: 270},
		{SpellID: 11197, Level: 46, Amount: 360},
		{SpellID: 11198, Level: 56, Amount: 450},
	}

	// Faerie Fire: armor removed (aura 674, misc 1).
	FaerieFireRanks = BuffRanks{
		{SpellID: 770, Level: 18, Amount: 175},
		{SpellID: 778, Level: 30, Amount: 285},
		{SpellID: 9749, Level: 42, Amount: 395},
		{SpellID: 9907, Level: 54, Amount: 505},
	}

	// Curse of Recklessness: armor removed (aura 674, misc 1). Forever's
	// rows have no attack power effect: the other is aura 92, no fleeing.
	CurseOfRecklessnessRanks = BuffRanks{
		{SpellID: 704, Level: 14, Amount: 175},
		{SpellID: 7658, Level: 28, Amount: 285},
		{SpellID: 7659, Level: 42, Amount: 395},
		{SpellID: 11717, Level: 56, Amount: 505},
	}

	// Curse of the Elements (Forever's all-magic curse): magic damage taken
	// in percent (aura 87) and magic resistance removed (aura 22), both
	// over misc value 126.
	CurseOfTheElementsDamageRanks = BuffRanks{
		{SpellID: 440892, Level: 20, Amount: 4},
		{SpellID: 1311676, Level: 30, Amount: 6},
		{SpellID: 1311677, Level: 40, Amount: 8},
		{SpellID: 1311680, Level: 50, Amount: 10},
	}
	CurseOfTheElementsResistanceRanks = BuffRanks{
		{SpellID: 440892, Level: 20, Amount: 30},
		{SpellID: 1311676, Level: 30, Amount: 45},
		{SpellID: 1311677, Level: 40, Amount: 60},
		{SpellID: 1311680, Level: 50, Amount: 75},
	}

	// Demoralizing Shout and Demoralizing Roar: melee attack power removed
	// (aura 99, the sign flipped), the rank's base points plus its points
	// per level up to its max level. Forever has no Improved Demoralizing
	// Shout or Roar talent.
	DemoralizingShoutRaidRanks = BuffRanks{
		{SpellID: 1160, Level: 14, Amount: 49, PerLevel: 1.4, MaxLevel: 24},
		{SpellID: 6190, Level: 24, Amount: 77, PerLevel: 1.4, MaxLevel: 34},
		{SpellID: 11554, Level: 34, Amount: 98, PerLevel: 1.4, MaxLevel: 44},
		{SpellID: 11555, Level: 44, Amount: 147, PerLevel: 1.4, MaxLevel: 54},
		{SpellID: 11556, Level: 54, Amount: 196, PerLevel: 1.4, MaxLevel: 64},
	}
	DemoralizingRoarRaidRanks = BuffRanks{
		{SpellID: 99, Level: 10, Amount: 46, PerLevel: 1.0, MaxLevel: 20},
		{SpellID: 1735, Level: 20, Amount: 74, PerLevel: 1.4, MaxLevel: 30},
		{SpellID: 9490, Level: 32, Amount: 95, PerLevel: 1.0, MaxLevel: 42},
		{SpellID: 9747, Level: 42, Amount: 144, PerLevel: 1.4, MaxLevel: 52},
		{SpellID: 9898, Level: 52, Amount: 193, PerLevel: 1.4, MaxLevel: 62},
	}

	// Hunter's Mark: ranged attack power against the target (effect 1,
	// aura 127). Rank 4 is 14325, the rank SkillLineAbility teaches.
	HuntersMarkRanks = BuffRanks{
		{SpellID: 1130, Level: 6, Amount: 26},
		{SpellID: 14323, Level: 22, Amount: 59},
		{SpellID: 14324, Level: 40, Amount: 98},
		{SpellID: HuntersMarkTopRankSpellID, Level: 58, Amount: HuntersMarkTopRankRangedAttackPower},
	}

	// Judgement of the Crusader: Holy damage taken (aura 14), by the level
	// of the Seal of the Crusader rank it judges (21082, 20162, 20305,
	// 20306, 20307, 20308). SpellID is the judgement's.
	JudgementOfTheCrusaderRanks = BuffRanks{
		{SpellID: 21183, Level: 6, Amount: 23},
		{SpellID: 20188, Level: 12, Amount: 35},
		{SpellID: 20300, Level: 22, Amount: 58},
		{SpellID: 20301, Level: 32, Amount: 92},
		{SpellID: 20302, Level: 42, Amount: 127},
		{SpellID: 20303, Level: 52, Amount: JudgementOfTheCrusaderRaidBonus},
	}
)

// raidDebuffLevel is the level a debuff on target is cast at: the raid's
// first player (a Forever sim's raid is one level), or the level cap when
// the target has no raid.
func raidDebuffLevel(target *Unit) int {
	if target.Env != nil && target.Env.Raid != nil && len(target.Env.Raid.AllPlayerUnits) > 0 {
		return int(target.Env.Raid.AllPlayerUnits[0].Level)
	}
	return CharacterMaxLevel
}

// raidDebuffRank is the rank of ranks the raid's level casts on target, and
// whether it has learned one. Below the first it is rank 1's id at nothing.
func raidDebuffRank(ranks BuffRanks, target *Unit) (BuffRank, bool) {
	if rank, ok := ranks.Learned(raidDebuffLevel(target)); ok {
		return rank, true
	}
	return BuffRank{SpellID: ranks[0].SpellID}, false
}

// exposeArmorRaidComboPoints is the raid's Expose Armor: a five-point cast.
const exposeArmorRaidComboPoints = 5
