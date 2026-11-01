package balance

import "testing"

// Vengeance (16909) effect 0 is aura 108, misc 15 (crit damage), with
// SpellClassMask_0 0x600207 in 1.60.1.70707: Wrath 0x1, Moonfire 0x2,
// Starfire 0x4 and Insect Swarm 0x200000 (SpellClassOptions 24977), as the
// text's "Arcane and Nature spells" says. Insect Swarm's ticks crit, so
// they take the bonus too.
func TestVengeanceRaisesInsectSwarmCrits(t *testing.T) {
	const vengeanceField = 10
	plain, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{insectSwarmField: 1}))
	talented, _, _ := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{insectSwarmField: 1, vengeanceField: 5}))
	rank := len(talented.InsectSwarm) - 1
	if got, want := talented.InsectSwarm[rank].CritDamageBonus-plain.InsectSwarm[rank].CritDamageBonus, 1.0; !near(got, want) {
		t.Errorf("Vengeance 5/5 raises Insect Swarm's crit damage bonus by %v, want %v", got, want)
	}
}
