import {
	Consumes,
	Flask,
	Food,
} from '../core/proto/common.js';
import { SavedTalents } from '../core/proto/ui.js';

import {
	PaladinAura,
	PaladinOptions as HolyPaladinOptions,
} from '../core/proto/paladin.js';

import * as PresetUtils from '../core/preset_utils.js';

import BlankGear from './gear_sets/blank.gear.json';

// Preset options for this spec.
// Eventually we will import these values for the raid sim too, so its good to
// keep them in a separate file.

export const DefaultGear = PresetUtils.makePresetGear('Blank', BlankGear);

// Default talents: the Forever Sixty guide's Holy build on the client
// 1.60.1.70707 trees.

export const StandardTalents = {
	name: 'Standard',
	data: SavedTalents.create({
		talentsString: '50320215012101251-50322-0503',
	}),
};

export const DefaultOptions = HolyPaladinOptions.create({
	aura: PaladinAura.DevotionAura,
});

export const DefaultConsumes = Consumes.create({
	flask: Flask.FlaskUnknown,
	food: Food.FoodUnknown,
});
