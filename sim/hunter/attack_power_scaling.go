package hunter

// Forever's server adds a share of the hunter's ranged attack power to
// three spells whose client SpellEffect rows carry BonusCoefficientFromAP
// 0 on every rank (1.60.1.70755; the same rows in 70790), so the client
// tables cannot show it. The sources, in the order the owner ranks them:
//
//   - Blizzard, WoW Forever Beta Development Notes, updated 3 November
//     2026 (us.forums.blizzard.com): "Arcane Shot: Base damage on all
//     ranks increased by ~30% and Attack Power coefficient raised from
//     0.1 to 0.15." and "Serpent Sting: Attack Power coefficient per tick
//     increased to 5% from 3%. Base damage on all ranks increased ~20%."
//     The notes of 16 October 2026: "The Hunter's Fire traps now
//     correctly scale with Attack Power."
//   - foreverlogs.gg beta reports 2695-2764 (2-10 October 2026, eight
//     hunters, levels 25-35): against the pre-push bases (Serpent Sting
//     rank 3/4 at 12/22 a tick, Arcane Shot rank 3/4 at 39/65) every
//     hunter's Serpent Sting tick sits 3% of its ranged attack power
//     above the base and its Arcane Shot hit 10% above, with the ranged
//     attack power read off the same hunter's Multi-Shot against Auto
//     Shot (sim-accuracy round 20, hunter.md). Lone Wolf hunters fit
//     with the 1.2 on the base only.
//   - The forever-hunter wiki's Attack Formulas page (in-game tests):
//     Serpent Sting 3% a tick (before the push), Arcane Shot 10% (before
//     the push), Immolation Trap 2% a tick. The trap's coefficient has no
//     second source yet: every beta report predates the 16 October fix.
//
// The share is read when the spell is cast or the periodic effect is
// applied. Blizzard's notes and the wiki say the ticks recalculate from
// the hunter's current attack power; the sim's ranged attack power moves
// only on short trinket and racial uses, so the snapshot is the same
// number for the fight.
const (
	serpentStingTickAPCoeff   = 0.05
	arcaneShotAPCoeff         = 0.15
	immolationTrapTickAPCoeff = 0.02
)
