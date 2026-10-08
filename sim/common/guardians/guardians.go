package guardians

import "github.com/wowsims/classic/sim/core"

func ConstructGuardians(character *core.Character) {
	constructEmeralDragonWhelps(character)
	constructCoreHound(character)
}
