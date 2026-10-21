package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/spellconst"
	"github.com/wowsims/classic/sim/core/stats"
)

// Effect 1 of every Hunter's Mark rank is aura 127, ranged attack power
// against the target; the client states a 120 s duration on all of them.
const (
	huntersMarkEffectIndex    = 1
	huntersMarkRangedAPAura   = 127
	huntersMarkClientDuration = 120000
)

func TestHuntersMarkRanksMatchClient(t *testing.T) {
	client, err := spellconst.Load(clientHunterSpellconst)
	if err != nil {
		t.Fatalf("loading the client file: %v", err)
	}
	for rank := 1; rank <= HunterSMarkRanks; rank++ {
		spell, ok := client.ByID(HunterSMarkSpellId[rank])
		if !ok {
			t.Fatalf("rank %d: spell %d is not in the client file", rank, HunterSMarkSpellId[rank])
		}
		effect := spell.Effects[huntersMarkEffectIndex]
		if effect.Aura != huntersMarkRangedAPAura {
			t.Errorf("rank %d: effect %d is aura %d, want %d", rank, huntersMarkEffectIndex, effect.Aura, huntersMarkRangedAPAura)
		}
		if effect.Amount != huntersMarkRangedAttackPower[rank] {
			t.Errorf("rank %d: engine ranged attack power %v, client %v", rank, huntersMarkRangedAttackPower[rank], effect.Amount)
		}
		if spell.SpellLevel != HunterSMarkLevel[rank] || spell.Cost != HunterSMarkManaCost[rank] || spell.DurationMS != huntersMarkClientDuration {
			t.Errorf("rank %d: level/cost/duration %d/%v/%d, client %d/%v/%d", rank,
				HunterSMarkLevel[rank], HunterSMarkManaCost[rank], huntersMarkClientDuration,
				spell.SpellLevel, spell.Cost, spell.DurationMS)
		}
	}
}

func TestHuntersMarkRankAtLevel(t *testing.T) {
	for _, tc := range []struct {
		level  int32
		wantID int32
	}{{5, 0}, {6, 1130}, {21, 1130}, {22, 14323}, {40, 14324}, {57, 14324}, {58, 1213268}, {60, 1213268}} {
		_, hunter, _ := newBareHunterAtLevel(t, tc.level)
		switch {
		case tc.wantID == 0 && hunter.HuntersMark != nil:
			t.Errorf("level %d: Hunter's Mark registered, want none", tc.level)
		case tc.wantID != 0 && hunter.HuntersMark == nil:
			t.Errorf("level %d: no Hunter's Mark, want spell %d", tc.level, tc.wantID)
		case tc.wantID != 0 && hunter.HuntersMark.ActionID.SpellID != tc.wantID:
			t.Errorf("level %d: Hunter's Mark is spell %d, want %d", tc.level, hunter.HuntersMark.ActionID.SpellID, tc.wantID)
		}
	}
}

// A landed cast puts the rank's debuff on the target: it carries the
// shared HuntersMark tag and raises the ranged attack power every attacker
// has against that target by the rank's amount.
func TestHuntersMarkCastDebuffsTheTarget(t *testing.T) {
	sim, hunter, target := newBareHunterAtLevel(t, 60)
	table := hunter.AttackTables[target.UnitIndex][proto.CastType_CastTypeRanged]
	before := table.BonusAttackPowerTaken

	for attempt := 0; attempt < 50 && !target.HasActiveAuraWithTag(core.HuntersMarkAuraTag); attempt++ {
		hunter.HuntersMark.ApplyEffects(sim, target, hunter.HuntersMark)
	}
	if !target.HasActiveAuraWithTag(core.HuntersMarkAuraTag) {
		t.Fatal("no cast of Hunter's Mark landed in 50 attempts")
	}
	if got, want := table.BonusAttackPowerTaken-before, huntersMarkRangedAttackPower[HunterSMarkRanks]; got != want {
		t.Errorf("ranged attack power taken moved by %v, want %v", got, want)
	}
}

// Aspect of the Falcon is level 60 only and grants melee and ranged attack
// power equal to the best Aspect of the Hawk rank.
func TestAspectOfTheFalconGrantsHawkAttackPowerToBothAttackKinds(t *testing.T) {
	_, belowCap, _ := newBareHunterAtLevel(t, 59)
	if belowCap.AspectOfTheFalcon != nil {
		t.Fatal("level 59 hunter has Aspect of the Falcon")
	}

	sim, hunter, _ := newBareHunterAtLevel(t, 60)
	if hunter.AspectOfTheFalcon == nil {
		t.Fatal("level 60 hunter has no Aspect of the Falcon")
	}
	want := hunter.getMaxAspectOfTheHawkAttackPower(hunter.getMaxHawkRank())
	attackPower := hunter.GetStat(stats.AttackPower)
	rangedAttackPower := hunter.GetStat(stats.RangedAttackPower)

	hunter.AspectOfTheFalcon.ApplyEffects(sim, nil, hunter.AspectOfTheFalcon)

	if got := hunter.GetStat(stats.AttackPower) - attackPower; got != want {
		t.Errorf("melee attack power moved by %v, want %v", got, want)
	}
	if got := hunter.GetStat(stats.RangedAttackPower) - rangedAttackPower; got != want {
		t.Errorf("ranged attack power moved by %v, want %v", got, want)
	}
}
