package item_sets

import "github.com/wowsims/classic/sim/core"

// The crafted, quest and event sets whose bonuses the client carries at
// thresholds the vanilla sets did not have.
func init() {
	registerClientSets(
		// Volcanic Armor.
		clientSet{id: 141, models: map[int32]bonusModel{9233: damageOnMeleeHit(core.SpellSchoolFire)}},
		// Stormshroud Armor.
		clientSet{id: 142, models: map[int32]bonusModel{
			18979: damageOnMeleeHit(core.SpellSchoolNature),
			23863: resourceOnMeleeHit,
		}},
		// Imperial Plate.
		clientSet{id: 321, noSim: map[int32]string{1302372: noSimMovement}},
		// The Darksoul.
		clientSet{id: 444},
		// Augur's Regalia.
		clientSet{id: 476, noSim: map[int32]string{24461: noSimUnreachable, 24462: noSimUnreachable}},
		// Haruspex's Garb.
		clientSet{id: 479, noSim: map[int32]string{24479: noSimUnreachable, 24480: noSimUnreachable}},
		// Black Dragon Mail.
		clientSet{id: 489},
		// Twilight Trappings.
		clientSet{id: 492, noSim: map[int32]string{24746: noSimCosmetic}},
		// Ironweave Battlesuit.
		clientSet{
			id:     520,
			models: map[int32]bonusModel{26283: spellPenetration},
			noSim: map[int32]string{
				27733:   noSimMovement,
				1302374: noSimMovement,
			},
		},
		// Battlegear of Undead Slaying, Undead Slayer's Armor, Garb of the
		// Undead Slayer and Regalia of Undead Cleansing.
		clientSet{id: 533, models: map[int32]bonusModel{29068: undeadSlaying}},
		clientSet{id: 534, models: map[int32]bonusModel{29068: undeadSlaying}},
		clientSet{id: 535, models: map[int32]bonusModel{29068: undeadSlaying}},
		clientSet{id: 536, models: map[int32]bonusModel{29068: undeadSlaying}},
		// Blessed Plate.
		clientSet{id: 1968, noSim: map[int32]string{1302375: noSimMovement}},
		// Stormcloth Regalia.
		clientSet{id: 1972},
		// Teachings of the Furbolgs.
		clientSet{id: 2071, noSim: map[int32]string{1294087: noSimUnreachable}},
		// Blessing of Kalimdor.
		clientSet{id: 2132, noSim: map[int32]string{1318244: noSimMovement}},
	)
}
