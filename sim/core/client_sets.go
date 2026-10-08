package core

import (
	"fmt"
	"slices"
	"sort"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// The client's item set rows (client_sets_gen.go) are the one source for
// what a Forever set bonus is. NewClientItemSet builds a set from them:
// every bonus spell that is a flat stat is applied from its row, so the
// amount is never typed twice; every other spell must be modelled by hand
// or declared as having no sim effect, with a reason. A bonus spell that
// is none of the three stops the engine at init.

// ClientEffect is one effect row of a client spell.
type ClientEffect struct {
	Effect    int32
	Aura      int32
	Misc0     int32
	Misc1     int32
	Points    float64
	Trigger   int32
	PeriodMS  int32
	ClassMask [4]uint32
}

// ClientSpell is the part of a client spell row the set models read.
type ClientSpell struct {
	Name       string
	DurationMS int32
	CooldownMS int32
	// ProcChance is SpellAuraOptions.ProcChance when the client states one
	// other than the "always" 101.
	ProcChance int32
	// InternalCooldownMS is SpellAuraOptions.ProcCategoryRecovery: how long
	// the spell's proc stays down after it fires.
	InternalCooldownMS int32
	Effects            []ClientEffect
}

// ClientSetBonus is one ItemSetSpell row.
type ClientSetBonus struct {
	Threshold int32
	SpellID   int32
}

// ClientSet is one ItemSet row with its bonuses.
type ClientSet struct {
	Name string
	// PhaseOnePieces is the most pieces of the set any one class can wear
	// from Phase 1 sources; a bonus above it cannot be reached in Phase 1.
	PhaseOnePieces int32
	Bonuses        []ClientSetBonus
}

// The client's aura and effect kinds the flat-stat decoder reads.
const (
	clientEffectApplyAura int32 = 6

	clientAuraModDamageDone int32 = 13
	clientAuraModResistance int32 = 22
	// ClientAuraFlatArmor is build 70291's renumbering of the aura 22 that
	// carried flat armor (Sunder Armor, Expose Armor, Faerie Fire, Curse of
	// Recklessness, Devotion Aura, Mark and Gift of the Wild): same
	// effect rows, same values, new aura id.
	ClientAuraFlatArmor            int32 = 674
	clientAuraModStat              int32 = 29
	clientAuraModSkill             int32 = 30
	clientAuraModParry             int32 = 47
	clientAuraModBlock             int32 = 51
	clientAuraModCrit              int32 = 52
	clientAuraModHit               int32 = 54
	clientAuraModSpellHit          int32 = 55
	clientAuraModSpellCrit         int32 = 57
	clientAuraModDamageVsCreature  int32 = 59
	clientAuraModCastHaste         int32 = 65
	clientAuraModManaRegen         int32 = 85
	clientAuraModAttackPower       int32 = 99
	clientAuraModAPVsCreature      int32 = 102
	clientAuraModRangedAttackPower int32 = 124
	clientAuraModRangedAPVsCreat   int32 = 131
	clientAuraModHealingDone       int32 = 135
	clientAuraModRating            int32 = 189
	clientAuraModCritAll           int32 = 290
	clientAuraModMeleeHaste        int32 = 342
	clientAuraModExpertise         int32 = 593

	// misc values
	clientSkillDefense      int32 = 95
	clientSchoolMaskPhys    int32 = 1
	clientSchoolMaskHoly    int32 = 2
	clientSchoolMaskFire    int32 = 4
	clientSchoolMaskNature  int32 = 8
	clientSchoolMaskFrost   int32 = 16
	clientSchoolMaskShadow  int32 = 32
	clientSchoolMaskArcane  int32 = 64
	clientSchoolMaskAllMagc int32 = 126
	clientRatingExpertise   int32 = 1 << 23

	// Expertise as a rating: the client prints tenths of a percent.
	clientExpertiseRatingTenths = 10
)

// clientMobTypes maps the client's creature-type mask bits to the engine's
// MobType.
var clientMobTypes = map[int32]proto.MobType{
	1:   proto.MobType_MobTypeBeast,
	2:   proto.MobType_MobTypeDragonkin,
	4:   proto.MobType_MobTypeDemon,
	8:   proto.MobType_MobTypeElemental,
	16:  proto.MobType_MobTypeGiant,
	32:  proto.MobType_MobTypeUndead,
	64:  proto.MobType_MobTypeHumanoid,
	256: proto.MobType_MobTypeMechanical,
}

// clientStatIndex maps a mod-stat aura's misc value to the engine's stat.
var clientStatIndex = map[int32]stats.Stat{
	0: stats.Strength,
	1: stats.Agility,
	2: stats.Stamina,
	3: stats.Intellect,
	4: stats.Spirit,
}

// clientSchoolPower maps a damage-done school bit to the engine's
// school-specific spell power stat.
var clientSchoolPower = map[int32]stats.Stat{
	clientSchoolMaskHoly:   stats.HolyPower,
	clientSchoolMaskFire:   stats.FirePower,
	clientSchoolMaskNature: stats.NaturePower,
	clientSchoolMaskFrost:  stats.FrostPower,
	clientSchoolMaskShadow: stats.ShadowPower,
	clientSchoolMaskArcane: stats.ArcanePower,
}

var clientSchoolResistance = map[int32]stats.Stat{
	clientSchoolMaskFire:   stats.FireResistance,
	clientSchoolMaskNature: stats.NatureResistance,
	clientSchoolMaskFrost:  stats.FrostResistance,
	clientSchoolMaskShadow: stats.ShadowResistance,
	clientSchoolMaskArcane: stats.ArcaneResistance,
}

// ClientSetRow returns the client's row for an ItemSet id.
func ClientSetRow(id int32) (ClientSet, bool) {
	row, ok := clientSetRows[id]
	return row, ok
}

// ClientSetRows is every Phase 1 client set, by ItemSet id. The map is
// shared; callers must not modify it.
func ClientSetRows() map[int32]ClientSet {
	return clientSetRows
}

// ClientSpellRow returns the client's row for a spell id.
func ClientSpellRow(id int32) (ClientSpell, bool) {
	row, ok := clientSpellRows[id]
	return row, ok
}

// MustClientSpellRow is ClientSpellRow for a spell the caller has named
// from a set row: a missing row is a stale generated table, not data.
func MustClientSpellRow(id int32) ClientSpell {
	row, ok := clientSpellRows[id]
	if !ok {
		panic(fmt.Sprintf("core: client spell %d is not in client_sets_gen.go; regenerate it with tools/gen_set_rows.py", id))
	}
	return row
}

// ClientBonusKind says how the engine models one client set bonus spell.
type ClientBonusKind int

const (
	// ClientBonusStat is a flat bonus applied from the client's row.
	ClientBonusStat ClientBonusKind = iota
	// ClientBonusModelled is a hand-written effect.
	ClientBonusModelled
	// ClientBonusNoSim is a bonus the sim has no use for, with a reason.
	ClientBonusNoSim
)

// ClientBonusModel is the engine's answer for one bonus spell.
type ClientBonusModel struct {
	Threshold int32
	SpellID   int32
	Kind      ClientBonusKind
	// Reason is the NoSim justification.
	Reason string
}

// ClientSetModel names how the engine models a client set's bonuses.
type ClientSetModel struct {
	ID int32
	// Effects models, by bonus spell id, every bonus that is not a flat stat.
	Effects map[int32]ApplyEffect
	// NoSim declares, by bonus spell id, the bonuses the sim does not model
	// and why (utility, a target the sim never fights, a trigger the sim
	// never produces). The reason is printed in the set conformance golden.
	NoSim map[int32]string
}

var clientSetModels = map[int32][]ClientBonusModel{}

// ClientSetModels is what each set registered through NewClientItemSet
// models, by ItemSet id. The map is shared; callers must not modify it.
func ClientSetModels() map[int32][]ClientBonusModel {
	return clientSetModels
}

// NewClientItemSet registers an item set from the client's rows. See
// ClientSetModel. Panics, at init, when a bonus spell is not a flat stat
// and is in neither Effects nor NoSim, or when either names a spell the
// set does not carry.
func NewClientItemSet(model ClientSetModel) *ItemSet {
	row, ok := clientSetRows[model.ID]
	if !ok {
		panic(fmt.Sprintf("core: client item set %d is not in client_sets_gen.go; regenerate it with tools/gen_set_rows.py", model.ID))
	}
	spellSet := map[int32]bool{}
	for _, bonus := range row.Bonuses {
		spellSet[bonus.SpellID] = true
	}
	for spellID := range model.Effects {
		if !spellSet[spellID] {
			panic(fmt.Sprintf("core: client item set %d (%s) has no bonus spell %d (Effects)", model.ID, row.Name, spellID))
		}
		if _, dup := model.NoSim[spellID]; dup {
			panic(fmt.Sprintf("core: client item set %d (%s) bonus spell %d is both modelled and NoSim", model.ID, row.Name, spellID))
		}
	}
	for spellID, reason := range model.NoSim {
		if !spellSet[spellID] {
			panic(fmt.Sprintf("core: client item set %d (%s) has no bonus spell %d (NoSim)", model.ID, row.Name, spellID))
		}
		if reason == "" {
			panic(fmt.Sprintf("core: client item set %d (%s) bonus spell %d is NoSim without a reason", model.ID, row.Name, spellID))
		}
	}

	byThreshold := map[int32][]ApplyEffect{}
	models := make([]ClientBonusModel, 0, len(row.Bonuses))
	for _, bonus := range row.Bonuses {
		apply, kind := resolveClientBonus(model, row, bonus)
		models = append(models, ClientBonusModel{
			Threshold: bonus.Threshold,
			SpellID:   bonus.SpellID,
			Kind:      kind,
			Reason:    model.NoSim[bonus.SpellID],
		})
		byThreshold[bonus.Threshold] = append(byThreshold[bonus.Threshold], apply)
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].Threshold != models[j].Threshold {
			return models[i].Threshold < models[j].Threshold
		}
		return models[i].SpellID < models[j].SpellID
	})
	clientSetModels[model.ID] = models

	bonuses := make(map[int32]ApplyEffect, len(byThreshold))
	for threshold, effects := range byThreshold {
		bonuses[threshold] = func(agent Agent) {
			for _, effect := range effects {
				effect(agent)
			}
		}
	}
	// The id and name come from the client's own table, so the vanilla
	// database check NewItemSet makes (its pieces exist) would only fail
	// for a set whose items live in the site's item database alone.
	return addItemSet(ItemSet{ID: model.ID, Name: row.Name, Bonuses: bonuses})
}

func resolveClientBonus(model ClientSetModel, row ClientSet, bonus ClientSetBonus) (ApplyEffect, ClientBonusKind) {
	if effect, ok := model.Effects[bonus.SpellID]; ok {
		return effect, ClientBonusModelled
	}
	if _, ok := model.NoSim[bonus.SpellID]; ok {
		return func(Agent) {}, ClientBonusNoSim
	}
	spell := MustClientSpellRow(bonus.SpellID)
	flat, ok := DecodeClientFlatBonus(spell)
	if !ok {
		panic(fmt.Sprintf("core: client item set %d (%s) %dP bonus %d (%s) is not a flat stat: model it in Effects or declare it in NoSim",
			model.ID, row.Name, bonus.Threshold, bonus.SpellID, spell.Name))
	}
	return func(agent Agent) { flat.apply(agent.GetCharacter(), bonus.SpellID) }, ClientBonusStat
}

// MobTypeBonus is a flat amount that applies against some creature types.
type MobTypeBonus struct {
	MobTypes []proto.MobType
	Amount   float64
}

// ClientFlatBonus is what a flat bonus spell gives: stats, plus attack
// power and spell damage against creature types.
type ClientFlatBonus struct {
	Stats         stats.Stats
	AttackPowerVs []MobTypeBonus
	SpellDamageVs []MobTypeBonus
}

func (flat ClientFlatBonus) apply(character *Character, spellID int32) {
	character.AddStats(flat.Stats)
	for _, bonus := range flat.AttackPowerVs {
		ApplyMobTypeAttackPower(character, spellID, bonus.MobTypes, bonus.Amount)
	}
	for _, bonus := range flat.SpellDamageVs {
		ApplyMobTypeSpellPower(character, spellID, bonus.MobTypes, bonus.Amount)
	}
}

func creatureTypes(mask int32) ([]proto.MobType, bool) {
	var out []proto.MobType
	for bit, mobType := range clientMobTypes {
		if mask&bit != 0 {
			out = append(out, mobType)
			mask &^= bit
		}
	}
	slices.Sort(out)
	return out, len(out) > 0 && mask == 0
}

// flatDecoder accumulates one spell's flat effects.
type flatDecoder struct {
	flat ClientFlatBonus

	damageAll, healing, hit, crit float64
	haste                         bool
}

// DecodeClientFlatBonus reads a client spell as a flat bonus. ok is false
// when any effect of the spell is not one it reads, so a proc, a modifier
// or a dummy never decodes as nothing.
func DecodeClientFlatBonus(spell ClientSpell) (ClientFlatBonus, bool) {
	var d flatDecoder
	for _, effect := range spell.Effects {
		if effect.Effect != clientEffectApplyAura || !d.read(effect) {
			return ClientFlatBonus{}, false
		}
	}
	d.finish()
	return d.flat, true
}

func (d *flatDecoder) read(e ClientEffect) bool {
	switch e.Aura {
	case clientAuraModAttackPower:
		d.flat.Stats[stats.AttackPower] += e.Points
	case clientAuraModRangedAttackPower:
		d.flat.Stats[stats.RangedAttackPower] += e.Points
	case clientAuraModAPVsCreature:
		return d.readVsCreature(&d.flat.AttackPowerVs, e)
	case clientAuraModRangedAPVsCreat:
		// The ranged half of a melee/ranged pair; the melee row carries it.
		_, ok := creatureTypes(e.Misc0)
		return ok
	case clientAuraModDamageVsCreature:
		return d.readVsCreature(&d.flat.SpellDamageVs, e)
	case clientAuraModDamageDone:
		return d.readSpellDamage(e)
	case clientAuraModHealingDone:
		if e.Misc0 != clientSchoolMaskAllMagc {
			return false
		}
		d.healing += e.Points
	case clientAuraModStat:
		stat, ok := clientStatIndex[e.Misc0]
		if !ok {
			return false
		}
		d.flat.Stats[stat] += e.Points
	case clientAuraModResistance, ClientAuraFlatArmor:
		return d.readResistance(e)
	case clientAuraModSkill:
		if e.Misc0 != clientSkillDefense {
			return false
		}
		d.flat.Stats[stats.Defense] += e.Points * DefenseRatingPerDefense
	case clientAuraModHit, clientAuraModSpellHit:
		d.hit = max(d.hit, e.Points)
	case clientAuraModCrit, clientAuraModSpellCrit, clientAuraModCritAll:
		d.crit = max(d.crit, e.Points)
	case clientAuraModMeleeHaste:
		d.flat.Stats[stats.MeleeHaste] += e.Points * HasteRatingPerHastePercent
	case clientAuraModCastHaste:
		d.flat.Stats[stats.SpellHaste] += e.Points * HasteRatingPerHastePercent
	case clientAuraModManaRegen:
		if e.Misc0 != 0 {
			return false
		}
		d.flat.Stats[stats.MP5] += e.Points
	case clientAuraModParry:
		d.flat.Stats[stats.Parry] += e.Points
	case clientAuraModBlock:
		d.flat.Stats[stats.Block] += e.Points
	case clientAuraModExpertise:
		// Dodge and parry reduction arrive as two rows of the same size.
		d.flat.Stats[stats.Expertise] = max(d.flat.Stats[stats.Expertise], -e.Points*ExpertiseRatingPerExpertiseChance)
	case clientAuraModRating:
		if e.Misc0 != clientRatingExpertise {
			return false
		}
		d.flat.Stats[stats.Expertise] += e.Points / clientExpertiseRatingTenths * ExpertiseRatingPerExpertiseChance
	default:
		return false
	}
	return true
}

func (d *flatDecoder) readVsCreature(into *[]MobTypeBonus, e ClientEffect) bool {
	types, ok := creatureTypes(e.Misc0)
	if !ok {
		return false
	}
	*into = append(*into, MobTypeBonus{MobTypes: types, Amount: e.Points})
	return true
}

func (d *flatDecoder) readSpellDamage(e ClientEffect) bool {
	if e.Misc0 == clientSchoolMaskAllMagc {
		d.damageAll += e.Points
		return true
	}
	stat, ok := clientSchoolPower[e.Misc0]
	if !ok {
		return false
	}
	d.flat.Stats[stat] += e.Points
	return true
}

func (d *flatDecoder) readResistance(e ClientEffect) bool {
	switch e.Misc0 {
	case clientSchoolMaskPhys:
		d.flat.Stats[stats.Armor] += e.Points
	case clientSchoolMaskAllMagc:
		for _, stat := range clientSchoolResistance {
			d.flat.Stats[stat] += e.Points
		}
	case clientSchoolMaskHoly:
		// The engine has no holy resistance.
	default:
		stat, ok := clientSchoolResistance[e.Misc0]
		if !ok {
			return false
		}
		d.flat.Stats[stat] += e.Points
	}
	return true
}

// finish folds the accumulated damage, healing, hit and crit into stats:
// equal damage and healing is the engine's SpellPower.
func (d *flatDecoder) finish() {
	switch {
	case d.damageAll > 0 && d.damageAll == d.healing:
		d.flat.Stats[stats.SpellPower] += d.damageAll
	default:
		d.flat.Stats[stats.SpellDamage] += d.damageAll
		d.flat.Stats[stats.HealingPower] += d.healing
	}
	d.flat.Stats[stats.Hit] += d.hit * HitRatingPerHitChance
	d.flat.Stats[stats.Crit] += d.crit * CritRatingPerCritChance
}
