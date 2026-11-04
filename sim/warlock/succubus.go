package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

func (warlock *Warlock) makeSuccubus() *WarlockPet {
	cfg := PetConfig{
		Name:          "Succubus",
		PowerModifier: 0.77,
	}

	cfg.Stats, cfg.AutoAttacks = petSheetAtLevel(warlock.Level, succubusSheet)

	return warlock.makePet(cfg, warlock.Options.Summon == proto.WarlockOptions_Succubus)
}

// succubusSheet is the demon's base sheet at the levels the engine carries one
// for (petSheetLevels); petSheetAtLevel interpolates between them.
func succubusSheet(level int32) (cfg PetConfig) {
	switch level {
	case 25:
		cfg.Stats = stats.Stats{
			stats.Strength:  50,
			stats.Agility:   40,
			stats.Stamina:   87,
			stats.Intellect: 35,
			stats.Spirit:    61,
			stats.Mana:      119,
			stats.MP5:       0,
		}
		cfg.AutoAttacks = core.AutoAttackOptions{
			MainHand: core.Weapon{
				BaseDamageMin: 23,
				BaseDamageMax: 38,
				SwingSpeed:    2,
			},
			AutoSwingMelee: true,
		}
	case 40:
		cfg.Stats = stats.Stats{
			stats.Strength:  74,
			stats.Agility:   58,
			stats.Stamina:   148,
			stats.Intellect: 49,
			stats.Spirit:    97,
			stats.Mana:      521,
			stats.MP5:       0,
		}
		cfg.AutoAttacks = core.AutoAttackOptions{
			MainHand: core.Weapon{
				BaseDamageMin: 41,
				BaseDamageMax: 61,
				SwingSpeed:    2,
			},
			AutoSwingMelee: true,
		}
	case 50:
		cfg.Stats = stats.Stats{
			stats.Strength:  107,
			stats.Agility:   71,
			stats.Stamina:   190,
			stats.Intellect: 59,
			stats.Spirit:    123,
			stats.Mana:      912,
			stats.MP5:       0,
		}
		cfg.AutoAttacks = core.AutoAttackOptions{
			MainHand: core.Weapon{
				// Upstream carried the level-40 weapon here ("Not updated"),
				// so a level-50 Succubus hit no harder than a level-40 one.
				// Until a level-50 Forever Succubus log gives the real roll,
				// the weapon lies halfway between the carried 40 (41-61) and
				// 60 (95-131) sheets, the same linear rule petSheetAtLevel
				// applies to every level without a sheet (unconfirmed).
				BaseDamageMin: 68,
				BaseDamageMax: 96,
				SwingSpeed:    2,
			},
			AutoSwingMelee: true,
		}
	case 60:
		cfg.Stats = stats.Stats{
			stats.Strength:  129,
			stats.Agility:   85,
			stats.Stamina:   234,
			stats.Intellect: 70,
			stats.Spirit:    150,
			stats.Mana:      1066,
			stats.MP5:       0,
		}
		cfg.AutoAttacks = core.AutoAttackOptions{
			MainHand: core.Weapon{
				BaseDamageMin: 95,
				BaseDamageMax: 131,
				SwingSpeed:    2,
			},
			AutoSwingMelee: true,
		}
	}
	return cfg
}

// lashOfPainLevel is the level each Lash of Pain rank (7814-11780) is
// learned at, indexed by rank.
var lashOfPainLevel = [7]int{0, 20, 28, 36, 44, 52, 60}

func (wp *WarlockPet) registerSuccubusLashOfPainSpell() {
	// The highest rank the demon's level has learned.
	rank := topLearnedRank(lashOfPainLevel[1:], int(wp.owner.Level))

	spellCoeff := [7]float64{0, .429, .429, .429, .429, .429, .429}[rank]
	// Forever's flat amounts, half of Era's 33-99: client SpellEffect
	// 7814-11780 base 16/22/30/36/43/50, no variance or per-level growth;
	// Wowhead Forever's rank 6 tooltip "causing 50 Shadow damage".
	clientBaseDamage := [7]float64{0, 16, 22, 30, 36, 43, 50}[rank]
	baseDamage := clientBaseDamage * (1 + .10*float64(wp.owner.Talents.ImprovedSayaad))
	spellId := [7]int32{0, 7814, 7815, 7816, 11778, 11779, 11780}[rank]
	manaCost := [7]float64{0, 65, 80, 105, 125, 145, 160}[rank]
	level := lashOfPainLevel[rank]

	wp.primaryAbility = wp.RegisterSpell(core.SpellConfig{
		ActionID:         core.ActionID{SpellID: spellId},
		SpellSchool:      core.SpellSchoolShadow,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		RequiredLevel:    level,
		Rank:             rank,
		ClientBaseDamage: [2]float64{clientBaseDamage, clientBaseDamage},

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer: wp.NewTimer(),
				// FOREVER: Improved Lash of Pain is not in the client's trees.
				// Duration: time.Second * (12 - time.Duration(3*wp.owner.Talents.ImprovedLashOfPain)),
				Duration: time.Second * 12,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,
		// Malevolence's class mask (1310949) carries Lash of Pain's bit
		// (0x2000), as Improved Sayaad's does.
		BonusCritRating: float64(wp.owner.Talents.Malevolence) * core.CritRatingPerCritChance,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			wp.applyOwnerSpellHit(spell)
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
		},
	})
}
