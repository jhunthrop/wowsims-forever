package warlock

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// registerTestWarlockAgent wires proto.Player_Warlock to a bare *Warlock
// agent, the way sim/warlock/dps's RegisterDpsWarlock wires it to a
// *DpsWarlock. That package can't be imported here (it imports this
// package, so importing it back would be a cycle), and these tests only
// need a warlock that can register and cast spells, not a rotation, so
// the bare Warlock is enough. init() runs once per test binary, so this
// only registers once regardless of how many _test.go files in this
// package call WithSpec.
func init() {
	core.RegisterAgentFactory(
		proto.Player_Warlock{},
		proto.Spec_SpecWarlock,
		func(character *core.Character, options *proto.Player) core.Agent {
			return NewWarlock(character, options, options.GetWarlock().Options)
		},
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_Warlock)
			if !ok {
				panic("Invalid spec value for Warlock!")
			}
			player.Spec = playerSpec
		},
	)
}
