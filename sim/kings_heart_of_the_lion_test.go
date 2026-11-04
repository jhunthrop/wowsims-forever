package sim

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Blessing of Kings and Heart of the Lion's area buff (409583) are the same
// +10% to every stat and do not stack: in Season of Discovery, whose spell
// Forever ships unchanged, Kings replaces the shared Lion aura's stats on the
// target and only the hunter's own extra 10% (409580) stays (Blizzard forums
// t/1738967, 2023-12-11; Game Rant's SoD rune ranking). The Lion aura's
// attack power is a separate effect and stays.
func withKings(p *proto.Player) *proto.Player {
	p.Buffs = &proto.IndividualBuffs{BlessingOfKings: true}
	return p
}

func TestKingsAndHeartOfTheLionDoNotStack(t *testing.T) {
	_, base := firstPlayerStats(t, &proto.RaidBuffs{}, priestPlayer(60))
	_, kingsOnly := firstPlayerStats(t, &proto.RaidBuffs{}, withKings(priestPlayer(60)))
	assertStatRatio(t, "priest with Kings", base, kingsOnly, 1.10)

	_, lionOnly := firstPlayerStats(t, &proto.RaidBuffs{HeartOfTheLion: true}, priestPlayer(60))
	_, both := firstPlayerStats(t, &proto.RaidBuffs{HeartOfTheLion: true}, withKings(priestPlayer(60)))
	assertStatRatio(t, "priest with Kings and the Lion", base, both, 1.10)
	if both[stats.RangedAttackPower] != lionOnly[stats.RangedAttackPower] {
		t.Errorf("ranged attack power with Kings and the Lion = %v, want the Lion's %v", both[stats.RangedAttackPower], lionOnly[stats.RangedAttackPower])
	}

	// The hunter keeps his own extra 10% on top of one shared 10%.
	hunterBase, hunterAlone := firstPlayerStats(t, &proto.RaidBuffs{}, hunterPlayer(60))
	_, hunterKings := firstPlayerStats(t, &proto.RaidBuffs{}, withKings(hunterPlayer(60)))
	assertStatRatio(t, "hunter with Kings", hunterBase, hunterKings, 1.21)
	if hunterKings != hunterAlone {
		t.Errorf("Kings changed a hunter already in his own Lion aura")
	}
}
