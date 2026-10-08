package warrior

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
	"github.com/wowsims/classic/sim/core/spellconst"
)

// defensiveStancePassiveSpellID is the passive aura Defensive Stance applies.
const defensiveStancePassiveSpellID = 7376

const (
	clientEffectThreat       = 63 // SPELL_EFFECT_THREAT: flat threat
	clientAuraAttackPower    = 99 // Demoralizing Shout's attack power reduction
	clientAuraSlowAttack     = 319
	clientEffectApplyAura    = 6
	attackPowerAmountEpsilon = 1e-6
)

func clientWarrior(t *testing.T) spellconst.Class {
	t.Helper()
	return clientdamagetest.LoadClient(t, "..", "warrior")
}

// findEffect returns the first effect of spell with the given effect code
// and, when aura is non-zero, aura code.
func findEffect(t *testing.T, spell spellconst.Spell, effect, aura int32) spellconst.Effect {
	t.Helper()
	for _, e := range spell.Effects {
		if e.Effect == effect && (aura == 0 || e.Aura == aura) {
			return e
		}
	}
	t.Fatalf("%s (%d) has no effect %d/%d", spell.Name, spell.ID, effect, aura)
	return spellconst.Effect{}
}

// Sunder Armor's flat threat per rank is the client's effect 63 amount.
func TestSunderArmorThreatMatchesClient(t *testing.T) {
	client := clientWarrior(t)
	for rank := 1; rank <= SunderArmorRanks; rank++ {
		spell, ok := client.ByID(SunderArmorSpellId[rank])
		if !ok {
			t.Fatalf("Sunder Armor rank %d (%d) is not in the client table", rank, SunderArmorSpellId[rank])
		}
		if got, want := sunderArmorThreat[rank], findEffect(t, spell, clientEffectThreat, 0).Amount; got != want {
			t.Errorf("rank %d: threat %v, client %v", rank, got, want)
		}
	}
}

// The Thunder Clap family the engine registers is the one the trainer
// teaches: each id is on a SkillLineAbility row (the 4618xx reissue is
// not), costs 200, takes the rank's level, and carries the client's
// cooldown, duration and 20% slow.
func TestThunderClapRanksAreTheLearnableFamily(t *testing.T) {
	client := clientWarrior(t)
	for rank := 1; rank <= ThunderClapRanks; rank++ {
		spell, ok := client.ByID(thunderClapRankSpellID[rank])
		if !ok {
			t.Fatalf("Thunder Clap rank %d (%d) is not in the client table", rank, thunderClapRankSpellID[rank])
		}
		if spell.SpellLevel != ThunderClapLevel[rank] || spell.Rank != rank {
			t.Errorf("rank %d: client says rank %d level %d, engine level %d", rank, spell.Rank, spell.SpellLevel, ThunderClapLevel[rank])
		}
		if got, want := time.Duration(spell.EffectiveCooldownMS())*time.Millisecond, thunderClapCooldown; got != want {
			t.Errorf("rank %d: cooldown %v, client %v", rank, want, got)
		}
		if got, want := time.Duration(spell.DurationMS)*time.Millisecond, thunderClapRankDuration[rank]; got != want {
			t.Errorf("rank %d: duration %v, client %v", rank, want, got)
		}
		if got := findEffect(t, spell, clientEffectApplyAura, clientAuraSlowAttack).Amount; got != -thunderClapAttackSpeedSlowPercent {
			t.Errorf("rank %d: slow %v, client %v", rank, -thunderClapAttackSpeedSlowPercent, got)
		}
		if got, want := ThunderClapManaCost[rank], spell.Cost; got != want {
			t.Errorf("rank %d: cost %v, client %v", rank, got, want)
		}
	}
}

// The thunderClapRankSpellID table's damage is the generated table's:
// both families roll the same amount.
func TestThunderClapLearnableFamilyDamageMatchesClient(t *testing.T) {
	assertGenerated(t, clientdamagetest.Direct, "Thunder Clap", thunderClapRankSpellID[:], ThunderClapDamage[:])
}

// Demoralizing Shout's attack power reduction is the client's aura 99
// amount at the rank's own level, growing 1.4 a level to the rank's cap.
func TestDemoralizingShoutReductionMatchesClient(t *testing.T) {
	client := clientWarrior(t)
	for rank := 1; rank <= 5; rank++ {
		id := []int32{0, 1160, 6190, 11554, 11555, 11556}[rank]
		spell, ok := client.ByID(id)
		if !ok {
			t.Fatalf("Demoralizing Shout rank %d (%d) is not in the client table", rank, id)
		}
		effect := findEffect(t, spell, clientEffectApplyAura, clientAuraAttackPower)
		if got, want := time.Duration(spell.DurationMS)*time.Millisecond, demoralizingShoutDuration; got != want {
			t.Errorf("rank %d: duration %v, client %v", rank, want, got)
		}
		if got, want := demoralizingShoutRageCost*10, spell.Cost; got != want {
			t.Errorf("rank %d: cost %v, client %v", rank, got, want)
		}
		for _, level := range []int32{int32(spell.SpellLevel), 60, 64, 70} {
			capped := math.Min(float64(level), float64(spell.MaxLevel))
			want := -(effect.Amount + effect.PointsPerLevel*math.Max(0, capped-float64(spell.SpellLevel)))
			if got := demoralizingShoutAttackPowerReduction(rank, level); math.Abs(got-want) > attackPowerAmountEpsilon {
				t.Errorf("rank %d at level %d: reduces %v attack power, client %v", rank, level, got, want)
			}
		}
	}
}

// Shield Block, Shield Wall and the talents' cooldowns read the client's
// own rows.
func TestShieldAbilitiesMatchClient(t *testing.T) {
	client := clientWarrior(t)

	block, _ := client.ByID(ShieldBlockSpellId[0])
	if got := time.Duration(block.DurationMS) * time.Millisecond; got != shieldBlockDuration {
		t.Errorf("Shield Block lasts %v, client %v", shieldBlockDuration, got)
	}
	if got := findEffect(t, block, clientEffectApplyAura, 51).Amount; got != shieldBlockChancePercent {
		t.Errorf("Shield Block adds %v%% block, client %v", float64(shieldBlockChancePercent), got)
	}

	wall, _ := client.ByID(ShieldWallSpellId[0])
	if got := time.Duration(wall.DurationMS) * time.Millisecond; got != shieldWallDuration {
		t.Errorf("Shield Wall lasts %v, client %v", shieldWallDuration, got)
	}
	if got := findEffect(t, wall, clientEffectApplyAura, 87).Amount; got != -100*(1-shieldWallDamageMultiple) {
		t.Errorf("Shield Wall cuts damage by %v%%, client %v", 100*(1-shieldWallDamageMultiple), got)
	}
	if got := int32(ShieldWallCooldownMS[0]); got != wall.EffectiveCooldownMS() {
		t.Errorf("Shield Wall cooldown %vms, client %vms", got, wall.EffectiveCooldownMS())
	}

	stance, _ := client.ByID(defensiveStancePassiveSpellID)
	if got := findEffect(t, stance, clientEffectApplyAura, 10).Amount; got != 100*defensiveStanceThreatBonus {
		t.Errorf("Defensive Stance adds %v%% threat, client %v", 100*defensiveStanceThreatBonus, got)
	}
	if got := findEffect(t, stance, clientEffectApplyAura, 87).Amount; got != -100*(1-defensiveStanceDamageMultiplier) {
		t.Errorf("Defensive Stance cuts damage taken by %v%%, client %v", 100*(1-defensiveStanceDamageMultiplier), got)
	}
}

// Bloodrage: an instant 10 rage and 1 a second, in tenths in the client.
func TestBloodrageMatchesClient(t *testing.T) {
	client := clientWarrior(t)
	cast, _ := client.ByID(2687)
	if got := findEffect(t, cast, 30, 0).Amount; got != bloodrageInstantRage*10 {
		t.Errorf("Bloodrage instant rage %v, client %v tenths", bloodrageInstantRage, got)
	}
	tick, _ := client.ByID(29131)
	if got := findEffect(t, tick, clientEffectApplyAura, 24).Amount; got != bloodrageRagePerSecond*10 {
		t.Errorf("Bloodrage rage a second %v, client %v tenths", bloodrageRagePerSecond, got)
	}
}
