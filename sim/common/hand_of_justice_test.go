package common

import "testing"

// Hand of Justice (11815): ItemEffect 99238 is 15600, whose 70790 row reads
// "${$h/3}% chance on Melee hit to gain $s1 extra attack. Attacks against
// Dwarves are $s2 times as likely to activate this effect." with
// SpellAuraOptions 128409 ProcChance 3 (2 sec recovery) and effect 1 (aura 4)
// $s2 = 3. Wowhead Forever renders "1% chance on Melee hit ... (Proc chance:
// 3%, 2s cooldown)". The 3% is the rate against Dwarves; no published target
// is a Dwarf, so every hit rolls 1%.
func TestHandOfJusticeProcsAtTheClientsNonDwarfRate(t *testing.T) {
	const clientProcChance, dwarfMultiplier = 0.03, 3.0
	if handOfJusticeProcChance != clientProcChance/dwarfMultiplier {
		t.Errorf("Hand of Justice procs %.3f a hit, want the client's %.3f", handOfJusticeProcChance, clientProcChance/dwarfMultiplier)
	}
}
