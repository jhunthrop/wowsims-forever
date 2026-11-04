package warrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Spearing Strike deals 75% weapon damage, and an additional 75% (150%
// total) against Giants, Dragonkin and mounted targets (1.60.1.70755;
// 40% and 80% before). This engine has no "mounted" state on a target,
// so only the MobType half of that clause applies here; the split is a
// pure function precisely so it is checkable without rolling weapon
// damage.
func TestSpearingStrikeDamagePercentByMobType(t *testing.T) {
	cases := []struct {
		mobType proto.MobType
		want    float64
	}{
		{proto.MobType_MobTypeUnknown, 0.75},
		{proto.MobType_MobTypeHumanoid, 0.75},
		{proto.MobType_MobTypeUndead, 0.75},
		{proto.MobType_MobTypeGiant, 1.5},
		{proto.MobType_MobTypeDragonkin, 1.5},
	}
	for _, c := range cases {
		target := &core.Unit{MobType: c.mobType}
		if got := spearingStrikeDamagePercent(target); got != c.want {
			t.Errorf("MobType %v: damage percent = %v, want %v", c.mobType, got, c.want)
		}
	}
}

// The two percentages are the client row's: 1310222 e1 (effect 31,
// weapon percent damage) 75, e2 1, the tooltip's $s2 and $s2*$s3.
func TestSpearingStrikeMatchesTheClient(t *testing.T) {
	row, ok := clientWarrior(t).ByID(TalentSpellIDs["spearing_strike"][0])
	if !ok || len(row.Effects) < 3 {
		t.Fatalf("Spearing Strike 1310222 is not in the client table with three effects")
	}
	if got, want := spearingStrikeWeaponDamagePercent*100, row.Effects[1].Amount; got != want {
		t.Errorf("Spearing Strike deals %v%% weapon damage, client e1 %v", got, want)
	}
	if got, want := spearingStrikeBonusMultiplier, row.Effects[2].Amount; got != want {
		t.Errorf("Spearing Strike's bonus multiplier %v, client e2 %v", got, want)
	}
	if got, want := int32(SpearingStrikeCooldownMS[0]), row.CooldownMS; got != want {
		t.Errorf("Spearing Strike cooldown %d ms, client %d", got, want)
	}
}
