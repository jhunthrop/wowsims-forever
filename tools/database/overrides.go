package database

import (
	"regexp"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

var OtherItemIdsToFetch = []string{}

var ItemOverrides = []*proto.UIItem{
	// Valentine's day event rewards
	// {Id: 51804, Phase: 2},

	// Some items might slip past the phase filters defined in main.go
	// Dragonspur Wraps
	{Id: 20615, Phase: 4},

	// Argent Dawn Armaments of Battle
	{Id: 22657, Phase: 6},
	{Id: 22667, Phase: 6},
	{Id: 22668, Phase: 6},
	{Id: 22659, Phase: 6},
	{Id: 22678, Phase: 6},
	{Id: 22656, Phase: 6},

	{Id: 22681, Phase: 6},
	{Id: 22680, Phase: 6},
	{Id: 22688, Phase: 6},
	{Id: 22679, Phase: 6},
	{Id: 22690, Phase: 6},
	{Id: 22689, Phase: 6},

	// Nef Head
	{Id: 19383, Phase: 3},
	{Id: 19366, Phase: 3},
	{Id: 19384, Phase: 3},

	// Dire Maul Crafts
	{Id: 18504, Phase: 2},
	{Id: 18506, Phase: 2},
	{Id: 18508, Phase: 2},
	{Id: 18509, Phase: 2},
	{Id: 18510, Phase: 2},
	{Id: 18511, Phase: 2},

	{Id: 18405, Phase: 2},
	{Id: 18407, Phase: 2},
	{Id: 18408, Phase: 2},
	{Id: 18409, Phase: 2},
	{Id: 18413, Phase: 2},

	// Sunken Temple Class Quests
	{Id: 22274, Phase: 3},
	{Id: 22272, Phase: 3},
	{Id: 22458, Phase: 3},

	{Id: 20083, Phase: 3},
	{Id: 19991, Phase: 3},
	{Id: 19992, Phase: 3},

	{Id: 20035, Phase: 3},
	{Id: 20037, Phase: 3},
	{Id: 20036, Phase: 3},

	{Id: 20504, Phase: 3},
	{Id: 20512, Phase: 3},
	{Id: 20505, Phase: 3},

	{Id: 19990, Phase: 3},
	{Id: 20082, Phase: 3},
	{Id: 20006, Phase: 3},

	{Id: 19984, Phase: 3},
	{Id: 20255, Phase: 3},
	{Id: 19982, Phase: 3},

	{Id: 20369, Phase: 3},
	{Id: 20556, Phase: 3},
	{Id: 20503, Phase: 3},

	{Id: 20536, Phase: 3},
	{Id: 20534, Phase: 3},
	{Id: 20530, Phase: 3},

	{Id: 20130, Phase: 3},
	{Id: 20521, Phase: 3},
	{Id: 20517, Phase: 3},

	// AQ Patch Items with incorrect Phases
	// Shaman Totems
	{Id: 23199, Phase: 5},
	{Id: 23200, Phase: 5},
	{Id: 22345, Phase: 5},
	{Id: 22395, Phase: 5},

	// Crafted
	{Id: 22191, Phase: 5},
	{Id: 22195, Phase: 5},

	// Forever's own enchanting-crafted "Heart of the Mountain" family
	// (level-50 trinkets; data/builds/1.60.1.70009/items/*.json in the
	// forever data repo, ids 249469/249470). No Wowhead/Wago page exists
	// for a Forever-original item, so MergeItem's "not already in db.Items"
	// branch inserts these wholesale instead of patching a scraped entry.
	// Name/icon/ilvl/stats copied verbatim from the client data; quality is
	// proto.ItemQuality_ItemQualityRare (3), matching that data's own
	// "quality": 3 (Rare/blue), not Epic.
	{
		Id:      249469,
		Name:    "Frozen Heart of the Mountain",
		Icon:    "inv_misc_gem_sapphire_01",
		Type:    proto.ItemType_ItemTypeTrinket,
		Ilvl:    55,
		Quality: proto.ItemQuality_ItemQualityRare,
		Stats: stats.Stats{
			stats.Hit:              9,
			stats.FireResistance:   7,
			stats.FrostResistance:  7,
			stats.NatureResistance: 7,
			stats.ShadowResistance: 7,
		}.ToFloatArray(),
	},
	{
		Id:      249470,
		Name:    "Molten Heart of the Mountain",
		Icon:    "inv_misc_gem_ruby_01",
		Type:    proto.ItemType_ItemTypeTrinket,
		Ilvl:    55,
		Quality: proto.ItemQuality_ItemQualityRare,
		Stats: stats.Stats{
			stats.FireResistance:   7,
			stats.FrostResistance:  7,
			stats.NatureResistance: 7,
			stats.ShadowResistance: 7,
		}.ToFloatArray(),
	},

	// Forever's own relics (data/builds/1.60.1.70009/items/*.json, class 4
	// subclass 7/8/9). None exists on Wowhead or Wago, and each has an equip
	// effect registered in sim/<class>/items.go, which needs the item in the
	// database. Name, icon, item level and quality are the client's.
	foreverRelic(220606, "Idol of the Dream", "inv_relics_idolofferocity", proto.RangedWeaponType_RangedWeaponTypeIdol, 60, proto.ItemQuality_ItemQualityRare),
	foreverRelic(228176, "Totem of Thunder", "inv_relics_totemofrebirth", proto.RangedWeaponType_RangedWeaponTypeTotem, 65, proto.ItemQuality_ItemQualityRare),
	foreverRelic(249441, "Talons of Wrath", "spell_nature_natureswrath", proto.RangedWeaponType_RangedWeaponTypeIdol, 45, proto.ItemQuality_ItemQualityRare),
	foreverRelic(249442, "Libram of Invocation", "inv_relics_libramofhope", proto.RangedWeaponType_RangedWeaponTypeLibram, 45, proto.ItemQuality_ItemQualityRare),
	foreverRelic(272427, "Howling Idol", "trade_archaeology_catstatueemeraldeyes", proto.RangedWeaponType_RangedWeaponTypeIdol, 65, proto.ItemQuality_ItemQualityRare),
	foreverRelic(272432, "Totem of the Storm", "spell_shaman_stormtotem", proto.RangedWeaponType_RangedWeaponTypeTotem, 65, proto.ItemQuality_ItemQualityRare),
	foreverRelic(272433, "Burning Totem", "spell_fire_totemofwrath", proto.RangedWeaponType_RangedWeaponTypeTotem, 65, proto.ItemQuality_ItemQualityRare),
	foreverRelic(272435, "Libram of Law", "inv_misc_book_04", proto.RangedWeaponType_RangedWeaponTypeLibram, 65, proto.ItemQuality_ItemQualityRare),
	foreverRelic(279248, "Libram of Infusion", "inv_misc_book_13", proto.RangedWeaponType_RangedWeaponTypeLibram, 65, proto.ItemQuality_ItemQualityEpic),

	// Forever's Tier 1 mage set, Manaflare Regalia (client ItemSet 2098;
	// data/builds/1.60.1.70009/items/mage.json). No Wowhead or Wago page
	// exists, and sim/mage/item_sets_pve.go registers its bonuses by name,
	// which needs at least one item of the set in the database. The
	// client's item rows carry no stats for these pieces in this build, so
	// none are listed.
	foreverMageTierPiece(280455, "Manaflare Crown", "inv_helm_cloth_raidmagehyjalc60_d_01", proto.ItemType_ItemTypeHead),
	foreverMageTierPiece(280454, "Manaflare Mantle", "inv_shoulder_cloth_raidmagehyjalc60_d_01", proto.ItemType_ItemTypeShoulder),
	foreverMageTierPiece(280450, "Manaflare Robes", "inv_chest_cloth_raidmagehyjalc60_d_01", proto.ItemType_ItemTypeChest),
	foreverMageTierPiece(280451, "Manaflare Gloves", "inv_glove_cloth_raidmagehyjalc60_d_01", proto.ItemType_ItemTypeHands),
	foreverMageTierPiece(280453, "Manaflare Pants", "inv_pant_cloth_raidmagehyjalc60_d_01", proto.ItemType_ItemTypeLegs),
	foreverMageTierPiece(280452, "Manaflare Boots", "inv_boot_cloth_raidmagehyjalc60_d_01", proto.ItemType_ItemTypeFeet),
}

// Keep these sorted by item ID.
var ItemAllowList = map[int32]struct{}{}

// Items to remove from the UI
var ItemDenyList = map[int32]struct{}{
	9653:  {}, // Speedy Racer Goggles
	12104: {}, // Brindlethorn Tunic
	12805: {}, // Orb of Fire
	17782: {}, // talisman of the binding shard
	17783: {}, // talisman of the binding fragment
	17802: {}, // Deprecated version of Thunderfury
	20522: {}, // Feral Staff
	22736: {}, // Andonisus, Reaper of Souls

	// Unimplemented PvP Belts/Bracers (Marshal's/General's)
	16482: {},
	16447: {},
	16458: {},
	16470: {},
	17585: {},
	17609: {},
	16439: {},
	16464: {},

	16481: {},
	17606: {},
	16438: {},
	16445: {},
	16460: {},
	16469: {},
	17582: {},
	16461: {},

	16572: {},
	16557: {},
	16547: {},
	16556: {},
	16575: {},
	17589: {},
	17621: {},
	16537: {},

	16553: {},
	16546: {},
	16576: {},
	16538: {},
	16559: {},
	16570: {},
	17587: {},
	17619: {},

	// Bladebane Armguards
	14550: {},
}

// Item icons to include in the DB, so they don't need to be separately loaded in the UI.
var ExtraItemIcons = []int32{
	// Demonic Rune
	12662,

	// Explosives
	13180,
	11566,
	8956,
	10646,
	18641,
	15993,
	16040,

	// Food IDs
	13928,
	20452,
	13931,
	18254,
	21023,
	13813,
	13810,

	// Flask IDs
	13510,
	13511,
	13512,
	13513,

	// Zanza
	20079,

	// Blasted Lands
	8412,
	8423,
	8424,
	8411,

	// Agility Elixer IDs
	13452,
	9187,

	// Single Elixirs
	20007, // Mana Regen Elixir
	20004, // Major Troll's Blood Potion
	9088,  // Gift of Arthas

	// Armor Elixirs
	3389,  // Defense
	8951,  // Greater
	13445, // Superior Defense

	// Health Elixirs
	2458, // Minor Fortitude
	3825, // Fortitude

	// Strength
	12451,
	9206,

	// AP
	12460,
	12820,

	// Random
	5206, // Bogling Root

	// SP
	13454,
	9264,
	21546,
	17708,

	// Crystal
	11564, // Armor

	// Alcohol Buff
	18284,
	18269,
	20709,
	21114,
	21151,

	// Potions / In Battle Consumes
	13444,

	// Thistle Tea
	7676,

	// Weapon Oils
	20748,
	20749,
	12404,
	18262,
}

var SpellIconoverrides = []*proto.IconData{}

// Raid buffs / debuffs
var SharedSpellsIcons = []int32{
	// World Buffs
	22888, // Ony / Nef
	24425, // Spirit
	16609, // Warchief
	23768, // DMF Damage
	23736, // DMF Agi
	23766, // DMF Int
	23738, // DMF Spirit
	23737, // DMF Stam

	22818, // DM Stam
	22820, // DM Spell Crit
	22817, // DM AP

	15366, // Songflower

	29534, // Silithus

	18264, // Headmasters

	// Registered CD's
	10060, // Power Infusion
	29166, // Innervate

	// Mark
	1126,
	5232,
	6756,
	5234,
	8907,
	9884,
	9885,
	17055,

	20217, // Kings (Talent)
	25898, // Greater Kings
	25899, // Sanctuary

	10293, // Devo Aura
	20142, // Imp. Devo

	// Stoneskin Totem
	10408,
	16293,

	// Fort
	1243,
	1244,
	1245,
	2791,
	10937,
	10938,
	14767,

	// Spirit
	14752,
	14818,
	14819,
	27841,

	// Might
	19740,
	19834,
	19835,
	19836,
	19837,
	19838,
	25291,
	20048,

	// Commanding Shout
	6673,
	5242,
	6192,
	11549,
	11550,
	11551,
	25289,
	12861,

	// AP
	30811, // Unleashed Rage
	19506, // Trueshot

	// Battle Shout
	6673,
	5242,
	6192,
	11549,
	11550,
	11551,
	25289,
	12861, // Imp

	// Wisdom
	19742,
	19850,
	19852,
	19853,
	19854,
	25290,
	20245,

	// Mana Spring
	5675,
	10495,
	10496,
	10497,

	17007, // Leader of the Pack
	24858, // Moonkin

	// Windfury
	8512,
	10613,
	10614,
	29193, // Imp WF

	// Raid Debuffs
	8647,
	7386,
	7405,
	8380,
	11596,
	11597,

	770,
	778,
	9749,
	9907,
	11708,
	18181,

	26016,
	12879,
	9452,
	26021,
	16862,
	9747,
	9898,

	3043,
	14275,
	14276,
	14277,

	17800,
	17803,
	12873,
	28593,

	11374,
	15235,

	24977,
}

// If any of these match the item name, don't include it.
var DenyListNameRegexes = []*regexp.Regexp{
	regexp.MustCompile(`30 Epic`),
	regexp.MustCompile(`63 Blue`),
	regexp.MustCompile(`63 Green`),
	regexp.MustCompile(`66 Epic`),
	regexp.MustCompile(`90 Epic`),
	regexp.MustCompile(`90 Green`),
	regexp.MustCompile(`Boots 1`),
	regexp.MustCompile(`Boots 2`),
	regexp.MustCompile(`Boots 3`),
	regexp.MustCompile(`Bracer 1`),
	regexp.MustCompile(`Bracer 2`),
	regexp.MustCompile(`Bracer 3`),
	regexp.MustCompile(`DB\d`),
	regexp.MustCompile(`DEPRECATED`),
	regexp.MustCompile(`Deprecated: Keanna`),
	regexp.MustCompile(`Indalamar`),
	regexp.MustCompile(`Monster -`),
	regexp.MustCompile(`NEW`),
	regexp.MustCompile(`PH`),
	regexp.MustCompile(`QR XXXX`),
	regexp.MustCompile(`TEST`),
	regexp.MustCompile(`Test`),
	regexp.MustCompile(`zOLD`),
}

// foreverRelic is a statless relic (libram, idol or totem) in the ranged slot.
func foreverRelic(id int32, name, icon string, kind proto.RangedWeaponType, ilvl int32, quality proto.ItemQuality) *proto.UIItem {
	return &proto.UIItem{
		Id:               id,
		Name:             name,
		Icon:             icon,
		Type:             proto.ItemType_ItemTypeRanged,
		RangedWeaponType: kind,
		Ilvl:             ilvl,
		Quality:          quality,
		Stats:            stats.Stats{}.ToFloatArray(),
	}
}

// Forever's Tier 1 mage set: name and id are the client's ItemSet 2098,
// the item level its rows' 65.
const (
	manaflareRegaliaName = "Manaflare Regalia"
	manaflareRegaliaID   = 2098
)

func foreverMageTierPiece(id int32, name, icon string, slot proto.ItemType) *proto.UIItem {
	return &proto.UIItem{
		Id:             id,
		Name:           name,
		Icon:           icon,
		Type:           slot,
		ArmorType:      proto.ArmorType_ArmorTypeCloth,
		Ilvl:           65,
		Phase:          1,
		Quality:        proto.ItemQuality_ItemQualityEpic,
		ClassAllowlist: []proto.Class{proto.Class_ClassMage},
		SetName:        manaflareRegaliaName,
		SetId:          manaflareRegaliaID,
		Stats:          stats.Stats{}.ToFloatArray(),
	}
}
