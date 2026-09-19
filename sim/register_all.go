package sim

import (
	"sync"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/druid/balance"
	"github.com/wowsims/classic/sim/paladin/retribution"
	dpsrogue "github.com/wowsims/classic/sim/rogue/dps_rogue"
	"github.com/wowsims/classic/sim/shaman/elemental"
	"github.com/wowsims/classic/sim/shaman/enhancement"
	"github.com/wowsims/classic/sim/shaman/warden"

	"github.com/wowsims/classic/sim/druid/feral"
	// restoDruid "github.com/wowsims/classic/sim/druid/restoration"
	// feralTank "github.com/wowsims/classic/sim/druid/tank"
	_ "github.com/wowsims/classic/sim/encounters"
	"github.com/wowsims/classic/sim/hunter"
	"github.com/wowsims/classic/sim/mage"

	// holyPaladin "github.com/wowsims/classic/sim/paladin/holy"
	"github.com/wowsims/classic/sim/paladin/protection"
	// "github.com/wowsims/classic/sim/paladin/retribution"
	// healingPriest "github.com/wowsims/classic/sim/priest/healing"
	"github.com/wowsims/classic/sim/priest/shadow"

	// restoShaman "github.com/wowsims/classic/sim/shaman/restoration"
	dpsWarlock "github.com/wowsims/classic/sim/warlock/dps"
	dpsWarrior "github.com/wowsims/classic/sim/warrior/dps_warrior"
	tankWarrior "github.com/wowsims/classic/sim/warrior/tank_warrior"
)

// registerOnce guards the agent-factory registrations, which are writes
// to package-level maps in core. The site's Cloud Run job calls
// RegisterAll from inside its per-run Execute rather than once in
// main(), so two concurrent runs reach this at the same time; the plain
// bool this replaced made that a data race and a possible double
// registration.
var registerOnce sync.Once

// RegisterAll registers every playable spec's agent factory. It is safe
// to call from more than one goroutine and does its work exactly once.
func RegisterAll() {
	registerOnce.Do(registerAll)
}

func registerAll() {
	balance.RegisterBalanceDruid()
	feral.RegisterFeralDruid()
	// feralTank.RegisterFeralTankDruid()
	// restoDruid.RegisterRestorationDruid()
	elemental.RegisterElementalShaman()
	enhancement.RegisterEnhancementShaman()
	warden.RegisterWardenShaman()
	// restoShaman.RegisterRestorationShaman()
	hunter.RegisterHunter()
	mage.RegisterMage()
	// healingPriest.RegisterHealingPriest()
	shadow.RegisterShadowPriest()
	dpsrogue.RegisterDpsRogue()
	dpsWarrior.RegisterDpsWarrior()
	tankWarrior.RegisterTankWarrior()
	// holyPaladin.RegisterHolyPaladin()
	protection.RegisterProtectionPaladin()
	retribution.RegisterRetributionPaladin()
	dpsWarlock.RegisterDpsWarlock()
}
