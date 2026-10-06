package warrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Spearing Strike deals 40% weapon damage, and an additional 80% (120%
// total) against Giants, Dragonkin and mounted targets. This engine has
// no "mounted" state on a target, so only the MobType half of that
// clause applies here; the split is a pure function precisely so it is
// checkable without rolling weapon damage.
func TestSpearingStrikeDamagePercentByMobType(t *testing.T) {
	cases := []struct {
		mobType proto.MobType
		want    float64
	}{
		{proto.MobType_MobTypeUnknown, 0.4},
		{proto.MobType_MobTypeHumanoid, 0.4},
		{proto.MobType_MobTypeUndead, 0.4},
		{proto.MobType_MobTypeGiant, 1.2},
		{proto.MobType_MobTypeDragonkin, 1.2},
	}
	for _, c := range cases {
		target := &core.Unit{MobType: c.mobType}
		if got := spearingStrikeDamagePercent(target); got != c.want {
			t.Errorf("MobType %v: damage percent = %v, want %v", c.mobType, got, c.want)
		}
	}
}
