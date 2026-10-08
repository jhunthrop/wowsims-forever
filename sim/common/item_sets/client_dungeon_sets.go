package item_sets

// The Forever dungeon sets and the Tier 0 sets Forever re-itemised with
// 2 to 6 piece bonuses (client ItemSets, build 1.60.1.70009).
func init() {
	registerClientSets(
		// The Gladiator.
		clientSet{
			id:     1,
			models: map[int32]bonusModel{1301123: roarOfTheCrowd},
			noSim:  map[int32]string{1301126: noSimMovement},
		},
		// Spider's Kiss.
		clientSet{id: 65, models: map[int32]bonusModel{17332: armorShredOnMeleeHit}},
		// The Postmaster.
		clientSet{
			id: 81,
			noSim: map[int32]string{
				1302376: noSimMovement,
				1302380: noSimSpellReflect,
			},
		},
		// Cadaverous Garb.
		clientSet{id: 121, noSim: map[int32]string{1292546: noSimMovement}},
		// Necropile Raiment.
		clientSet{id: 122, models: map[int32]bonusModel{
			1299734: ratingBonus,
			1299737: necropileDrain,
		}},
		// Bloodmail Regalia.
		clientSet{id: 123, models: map[int32]bonusModel{
			1299740: bloodmail,
			1299738: ratingBonus,
		}},
		// Deathbone Guardian.
		clientSet{id: 124, models: map[int32]bonusModel{1299743: burstWhenStruck}},
		// Defias Leather.
		clientSet{
			id:     161,
			models: map[int32]bonusModel{1292028: deviousStrike, 7534: daggerSkill},
		},
		// Embrace of the Viper.
		clientSet{
			id: 162,
			noSim: map[int32]string{
				1291796: noSimStruckUtility,
				1291807: noSimStun,
			},
		},
		// Chain of the Scarlet Crusade.
		clientSet{
			id:     163,
			models: map[int32]bonusModel{1293674: enragingLight},
			noSim:  map[int32]string{1293670: noSimStruckUtility},
		},
		// Spirit of Eskhandar (three pieces obtainable).
		clientSet{id: 261, noSim: map[int32]string{22648: noSimUnreachable}},
		// Magister's Regalia.
		clientSet{
			id:     181,
			models: map[int32]bonusModel{450527: manaOnSpellcast},
			noSim:  map[int32]string{27867: noSimStruckUtility},
		},
		// Vestments of the Devout.
		clientSet{
			id:     182,
			models: map[int32]bonusModel{450576: manaOnSpellcast},
			noSim:  map[int32]string{27778: noSimStruckUtility},
		},
		// Dreadmist Raiment.
		clientSet{
			id: 183,
			noSim: map[int32]string{
				27780:  noSimStruckUtility,
				450585: noSimSelfHeal,
			},
		},
		// Shadowcraft Armor.
		clientSet{
			id:     184,
			models: map[int32]bonusModel{27787: resourceOnMeleeHit},
			noSim: map[int32]string{
				1302698: noSimStruckUtility,
				21347:   noSimMovement,
			},
		},
		// Wildheart Raiment.
		clientSet{
			id:     185,
			models: map[int32]bonusModel{450608: wildheartReturns},
			noSim:  map[int32]string{1302708: noSimStruckUtility},
		},
		// Beaststalker Armor.
		clientSet{
			id:     186,
			models: map[int32]bonusModel{450577: manaOnAutoattack},
			noSim:  map[int32]string{1302712: noSimStruckUtility},
		},
		// The Elements.
		clientSet{
			id:     187,
			models: map[int32]bonusModel{450626: spellPowerBurst},
			noSim:  map[int32]string{1302718: noSimStruckUtility},
		},
		// Lightforge Armor.
		clientSet{
			id:     188,
			models: map[int32]bonusModel{450625: spellPowerBurst},
			noSim:  map[int32]string{1302716: noSimStruckUtility},
		},
		// Battlegear of Valor.
		clientSet{
			id:     189,
			models: map[int32]bonusModel{450587: resourceOnMeleeHit},
			noSim: map[int32]string{
				1302721: noSimStruckUtility,
				21347:   noSimMovement,
			},
		},
	)
}
