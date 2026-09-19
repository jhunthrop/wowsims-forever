import { Player } from '../core/player.js';
import * as PresetUtils from '../core/preset_utils.js';
import {
	Alcohol,
	Conjured,
	Consumes,
	Debuffs,
	FirePowerBuff,
	Flask,
	Food,
	IndividualBuffs,
	ManaRegenElixir,
	Potions,
	Profession,
	RaidBuffs,
	SaygesFortune,
	ShadowPowerBuff,
	SpellPowerBuff,
	TristateEffect,
	WeaponImbue,
	ZanzaBuff,
} from '../core/proto/common';
import { SavedTalents } from '../core/proto/ui.js';
import {
	WarlockOptions as WarlockOptions,
	WarlockOptions_Armor as Armor,
	WarlockOptions_Summon as Summon,
	WarlockOptions_WeaponImbue as WarlockWeaponImbue,
} from '../core/proto/warlock.js';
// apls
import ForeverAfflictionAPL from './apls/forever_affliction.apl.json';
import ForeverDemonologyAPL from './apls/forever_demonology.apl.json';
import ForeverDestructionAPL from './apls/forever_destruction.apl.json';
import BasicRotation from './apls/rotation.apl.json';
// gear
import BlankGear from './gear_sets/blank.gear.json';
import MCGear from './gear_sets/mc.gear.json';
import PreBisGear from './gear_sets/prebis.gear.json';

///////////////////////////////////////////////////////////////////////////
//                                 Gear Presets
///////////////////////////////////////////////////////////////////////////

export const GearBlank = PresetUtils.makePresetGear('Blank', BlankGear);
export const GearPreBis = PresetUtils.makePresetGear('Pre-BIS', PreBisGear);
export const GearMC = PresetUtils.makePresetGear('MC', MCGear);

export const GearPresets = [
	GearBlank,
	GearPreBis,
	GearMC,
];

export const DefaultGear = GearPreBis;

///////////////////////////////////////////////////////////////////////////
//                                 APL Presets
///////////////////////////////////////////////////////////////////////////

// The Forever rotations below are the launch defaults. Each is a copy of
// data/curated/apl/<class>-<spec>.json's `rotation` in the Forever Sixty site
// repository, written here by that repository's `make apl-sync` and proved by
// its `make apl-check`; edit the curated file, never this copy. The Era lists
// are kept because a Forever character can still be compared against them.
// One warlock package serves all three specs, so all three Forever rotations
// live here. Destruction is the default: it is the spec this package's own
// Era preset is named and written for.
// P1
export const RotationSB = PresetUtils.makePresetAPLRotation('Destruction', BasicRotation);
export const AplForeverAffliction = PresetUtils.makePresetAPLRotation('Forever Affliction', ForeverAfflictionAPL);
export const AplForeverDemonology = PresetUtils.makePresetAPLRotation('Forever Demonology', ForeverDemonologyAPL);
export const AplForeverDestruction = PresetUtils.makePresetAPLRotation('Forever Destruction', ForeverDestructionAPL);

export const APLPresets = [RotationSB, AplForeverAffliction, AplForeverDemonology, AplForeverDestruction];

export const DefaultAPL = AplForeverDestruction;

///////////////////////////////////////////////////////////////////////////
//                                 Talent Presets
///////////////////////////////////////////////////////////////////////////

// Default talents. Uses the wowhead calculator format, make the talents on
// https://wowhead.com/classic/talent-calc and copy the numbers in the url.

export const TalentsSMRuid = {
	name: 'SM/Ruin',
	data: SavedTalents.create({ talentsString: '5502203112201105--52500051020001' }),
};

export const TalentsDSRuin = {
	name: 'DS/Ruin',
	data: SavedTalents.create({ talentsString: '25002-2050300152201-52500051020001' }),
};

export const TalentPresets = [TalentsSMRuid, TalentsDSRuin];

export const DefaultTalents = TalentsDSRuin;

///////////////////////////////////////////////////////////////////////////
//                                 Options
///////////////////////////////////////////////////////////////////////////

export const DefaultOptions = WarlockOptions.create({
	armor: Armor.DemonArmor,
	summon: Summon.Succubus,
	weaponImbue: WarlockWeaponImbue.NoWeaponImbue,
});

export const DefaultConsumes = Consumes.create({
	alcohol: Alcohol.AlcoholRumseyRumBlackLabel,
	defaultPotion: Potions.MajorManaPotion,
	defaultConjured: Conjured.ConjuredDemonicRune,
	flask: Flask.FlaskOfSupremePower,
	firePowerBuff: FirePowerBuff.ElixirOfFirepower,
	food: Food.FoodRunnTumTuberSurprise,
	// mainHandImbue: WeaponImbue.BrilliantWizardOil,
	manaRegenElixir: ManaRegenElixir.MagebloodPotion,
	spellPowerBuff: SpellPowerBuff.GreaterArcaneElixir,
	shadowPowerBuff: ShadowPowerBuff.ElixirOfShadowPower,
	zanzaBuff: ZanzaBuff.CerebralCortexCompound,
});

export const DefaultRaidBuffs = RaidBuffs.create({
	arcaneBrilliance: true,
	divineSpirit: true,
	fireResistanceAura: true,
	fireResistanceTotem: true,
	giftOfTheWild: TristateEffect.TristateEffectImproved,
	manaSpringTotem: TristateEffect.TristateEffectRegular,
	moonkinAura: true,
	powerWordFortitude: TristateEffect.TristateEffectImproved,
});

export const DefaultIndividualBuffs = IndividualBuffs.create({
	blessingOfKings: true,
	blessingOfWisdom: TristateEffect.TristateEffectImproved,
	fengusFerocity: true,
	moldarsMoxie: true,
	rallyingCryOfTheDragonslayer: true,
	// saygesFortune: SaygesFortune.SaygesDamage,
	slipkiksSavvy: true,
	songflowerSerenade: true,
	// spiritOfZandalar: true,
	warchiefsBlessing: true,
});

export const DefaultDebuffs = Debuffs.create({
	exposeArmor: TristateEffect.TristateEffectImproved,
	faerieFire: true,
	improvedScorch: true,
	judgementOfWisdom: true,
	shadowWeaving: true,
	sunderArmor: true,
});

export const OtherDefaults = {
	distanceFromTarget: 25,
	profession1: Profession.Enchanting,
	profession2: Profession.Tailoring,
	channelClipDelay: 150,
};
