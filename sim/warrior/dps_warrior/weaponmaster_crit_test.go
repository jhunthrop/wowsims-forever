package dpswarrior

import (
	"math"
	"sort"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Round 10 (physical lane): Weaponmaster's axe and polearm crit. The client
// row (1290261) is a server dummy; its text gives the crit to "your melee
// weapon attacks", so it is melee-table crit. It must not reach the weapon
// procs (Lightning, Fatal Wound), which are DefenseType 1 and roll spell
// crit.
func TestWeaponmasterAxeCritIsMeleeOnly(t *testing.T) {
	axe := firstWeaponOfType(t, proto.WeaponType_WeaponTypeAxe)
	talents := talentStringWithRank(t, emptyWarriorTalents, "weaponmaster", 5)

	with := buildWarriorWithWeapons(t, talents, axe, 0)
	without := buildWarriorWithWeapons(t, emptyWarriorTalents, axe, 0)

	if got := with.GetStat(stats.MeleeCrit) - without.GetStat(stats.MeleeCrit); math.Abs(got-5*core.CritRatingPerCritChance) > 1e-9 {
		t.Errorf("Weaponmaster 5/5 with an axe raised melee crit by %v, want 5%%", got)
	}
	if got := with.GetStat(stats.Crit) - without.GetStat(stats.Crit); got != 0 {
		t.Errorf("Weaponmaster raised every-table crit by %v; spell-table weapon procs must not take it", got)
	}
}

func firstWeaponOfType(t *testing.T, weaponType proto.WeaponType) int32 {
	t.Helper()
	var ids []int32
	for id, item := range core.ItemsByID {
		if item.Type == proto.ItemType_ItemTypeWeapon && item.WeaponType == weaponType && item.HandType == proto.HandType_HandTypeTwoHand && item.WeaponDamageMax > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		t.Fatalf("no two-handed %v in the item database; this test needs --tags=with_db", weaponType)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids[0]
}
