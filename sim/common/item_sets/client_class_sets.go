package item_sets

// The Forever class variants of the Tier 0 sets, and the Soulforge and
// Virtuous sets, which carry the same bonus spells at the same models.
func init() {
	registerClientSets(
		// Vestments of the Virtuous (two pieces obtainable).
		clientSet{id: 514, noSim: map[int32]string{27778: noSimUnreachable}},
		// Soulforge Armor (two pieces obtainable).
		clientSet{id: 516, noSim: map[int32]string{27498: noSimUnreachable}},
		// Feralheart Raiment.
		clientSet{
			id:     1667,
			models: map[int32]bonusModel{450608: wildheartReturns},
			noSim:  map[int32]string{1302708: noSimStruckUtility},
		},
		// Beastmaster Armor.
		clientSet{
			id:     1669,
			models: map[int32]bonusModel{450577: manaOnAutoattack},
			noSim:  map[int32]string{1302712: noSimStruckUtility},
		},
		// Sorcerer's Regalia.
		clientSet{
			id:     1671,
			models: map[int32]bonusModel{450527: manaOnSpellcast},
			noSim:  map[int32]string{27867: noSimStruckUtility},
		},
		// Soulforge Armor.
		clientSet{
			id:     1673,
			models: map[int32]bonusModel{450625: spellPowerBurst},
			noSim:  map[int32]string{1302716: noSimStruckUtility},
		},
		// Vestments of the Virtuous.
		clientSet{
			id:     1675,
			models: map[int32]bonusModel{450576: manaOnSpellcast},
			noSim:  map[int32]string{27778: noSimStruckUtility},
		},
		// Darkmantle Armor.
		clientSet{
			id:     1676,
			models: map[int32]bonusModel{27787: resourceOnMeleeHit},
			noSim: map[int32]string{
				21347:   noSimMovement,
				1302698: noSimStruckUtility,
			},
		},
		// The Five Thunders.
		clientSet{
			id:     1679,
			models: map[int32]bonusModel{450626: spellPowerBurst},
			noSim:  map[int32]string{1302718: noSimStruckUtility},
		},
		// Deathmist Raiment.
		clientSet{
			id: 1681,
			noSim: map[int32]string{
				450585: noSimSelfHeal,
				27780:  noSimStruckUtility,
			},
		},
		// Battlegear of Heroism.
		clientSet{
			id:     1778,
			models: map[int32]bonusModel{450587: resourceOnMeleeHit},
			noSim: map[int32]string{
				21347:   noSimMovement,
				1302721: noSimStruckUtility,
			},
		},
		// The vanilla ids of the Tier 0.5 class sets: the client keeps the
		// original rows (2/4/6/8 bonuses) beside the Forever variants above,
		// and Phase 1 items carry both ids, so each id needs its own set.
		// Battlegear of Heroism (two pieces obtainable).
		clientSet{id: 511, noSim: map[int32]string{27419: noSimSelfHeal}},
		// Darkmantle Armor (two pieces obtainable).
		clientSet{id: 512, models: map[int32]bonusModel{27787: resourceOnMeleeHit}},
		// Feralheart Raiment.
		clientSet{id: 513, noSim: map[int32]string{27781: noSimStruckUtility}},
		// Beastmaster Armor (two pieces obtainable).
		clientSet{id: 515, models: map[int32]bonusModel{27785: manaOnRangedAttack}},
		// Sorcerer's Regalia (two pieces obtainable).
		clientSet{id: 517, noSim: map[int32]string{27867: noSimStruckUtility}},
		// The Five Thunders (one piece obtainable).
		clientSet{id: 519, models: map[int32]bonusModel{27774: spellPowerBurstOnCast}},
	)
}
