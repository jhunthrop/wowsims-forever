package healing

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/healsim"
)

// drainMana spends all but a tenth of the priest's mana so a test can read
// what an ability returns without the bar's cap in the way.
func drainMana(sim *core.Simulation, agent *HealingPriest) {
	agent.SpendMana(sim, agent.CurrentMana()-0.1*agent.MaxMana(), agent.NewManaMetrics(core.ActionID{OtherID: 1}))
}

// runFor steps the sim for the duration after now.
func runFor(sim *core.Simulation, duration time.Duration) {
	end := sim.CurrentTime + duration
	for sim.CurrentTime < end {
		if sim.Step() {
			return
		}
	}
}

// manaAfter drains a fresh healer's mana, runs act, steps 16 s and returns
// the mana and health it ends with. Running it with and without the act
// leaves the sim's own regeneration out of the difference.
func manaAfter(t *testing.T, act func(*core.Simulation, *HealingPriest)) (mana, health float64) {
	t.Helper()
	sim, agent := healerSim(t, 60, "")
	drainMana(sim, agent)
	if act != nil {
		act(sim, agent)
	}
	runFor(sim, 16*time.Second)
	return agent.CurrentMana(), agent.CurrentHealth()
}

// TestShadowfiendReturnsFivePercentOfManaPerSwing holds the fiend to the
// client's 15 s summon, the 5 percent Mana Leech and the stated 1.5 s swing:
// ten returns, 50 percent of the bar, and the five minute cooldown.
func TestShadowfiendReturnsFivePercentOfManaPerSwing(t *testing.T) {
	idle, _ := manaAfter(t, nil)
	var maxMana float64
	withFiend, _ := manaAfter(t, func(sim *core.Simulation, agent *HealingPriest) {
		fiend := agent.Shadowfiend
		if fiend == nil || fiend.SpellID != 401977 {
			t.Fatalf("the priest has no Shadowfiend (401977): %v", fiend)
		}
		maxMana = agent.MaxMana()
		fiend.ApplyEffects(sim, &agent.Unit, fiend)
	})
	want := 10 * 0.05 * maxMana
	if got := withFiend - idle; math.Abs(got-want) > 1 {
		t.Errorf("Shadowfiend returned %.1f mana, want %.1f (ten swings of 5 percent)", got, want)
	}
}

func TestShadowfiendHasAFiveMinuteCooldown(t *testing.T) {
	sim, agent := healerSim(t, 60, "")
	fiend := agent.Shadowfiend
	if !fiend.CanCast(sim, &agent.Unit) {
		t.Fatal("a fresh Shadowfiend cannot be cast")
	}
	fiend.Cast(sim, &agent.Unit)
	if got := fiend.CD.Duration; got != 5*time.Minute {
		t.Errorf("Shadowfiend cooldown is %v, want 5m", got)
	}
	if fiend.IsReady(sim) {
		t.Error("Shadowfiend is ready right after the cast")
	}
}

// TestDarkSacrificeTradesHealthForManaPlusSpirit holds the top rank to the
// client's five 3 s ticks of 320 health for 320 mana, with the caster's
// Spirit added to the mana over the five ticks.
func TestDarkSacrificeTradesHealthForManaPlusSpirit(t *testing.T) {
	idleMana, idleHealth := manaAfter(t, nil)
	var spirit float64
	mana, health := manaAfter(t, func(sim *core.Simulation, agent *HealingPriest) {
		spell := agent.DarkSacrifice[5]
		if spell == nil || spell.SpellID != 1277328 {
			t.Fatalf("the level 60 priest has no Dark Sacrifice rank 5 (1277328): %v", spell)
		}
		spirit = agent.GetStat(stats.Spirit)
		spell.ApplyEffects(sim, &agent.Unit, spell)
	})
	wantHealth := 5 * 320.0
	if got := idleHealth - health; math.Abs(got-wantHealth) > 1e-6 {
		t.Errorf("Dark Sacrifice cost %.1f health, want %.1f", got, wantHealth)
	}
	if got, want := mana-idleMana, wantHealth+spirit; math.Abs(got-want) > 1e-6 {
		t.Errorf("Dark Sacrifice returned %.1f mana, want %.1f (a mana for each health plus Spirit)", got, want)
	}
}

func TestDarkSacrificeRanksFollowTheClientLevels(t *testing.T) {
	_, agent := healerSim(t, 40, "")
	for rank, want := range map[int]bool{1: true, 2: true, 3: true, 4: false, 5: false} {
		if got := agent.DarkSacrifice[rank] != nil; got != want {
			t.Errorf("level 40 priest has rank %d = %v, want %v", rank, got, want)
		}
	}
}

// TestRotationsCastTheManaAbilitiesBehindTheirGates holds both written
// priest lists to casting Shadowfiend and Dark Sacrifice in a long fight,
// where the bar falls through the lists' gates.
func TestRotationsCastTheManaAbilitiesBehindTheirGates(t *testing.T) {
	for _, c := range rotationCases {
		t.Run(c.name, func(t *testing.T) {
			player := healer(60, c.talents, &proto.HealingPriest_Options{UseInnerFire: true}, loadRotation(t, c.file))
			summary, err := healsim.Run(healsim.Request(player, healsim.TestProfile(), 600, 5))
			if err != nil {
				t.Fatal(err)
			}
			for name, id := range map[string]int32{"Shadowfiend": 401977, "Dark Sacrifice": 1277328} {
				if summary.Casts[id] == 0 {
					t.Errorf("the %s list never cast %s", c.name, name)
				}
			}
		})
	}
}
