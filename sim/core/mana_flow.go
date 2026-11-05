package core

import "time"

// DefaultManaFlowWindow is the trailing window an APL manaIncomePerSecond or
// manaSpentPerSecond value reads when its config leaves window_seconds at 0.
const DefaultManaFlowWindow = 10 * time.Second

type manaMove struct {
	at     time.Duration
	amount float64
}

// manaFlowLog keeps the mana a unit moved one way (gained, or spent) over the longest
// window any APL value reads. It records nothing until a value asks for it, so units whose
// rotations never read their mana flow pay nothing in AddMana or SpendMana.
type manaFlowLog struct {
	window time.Duration
	moves  []manaMove
}

func (log *manaFlowLog) track(window time.Duration) {
	log.window = max(log.window, window)
}

func (log *manaFlowLog) record(now time.Duration, amount float64) {
	if log.window == 0 || amount <= 0 {
		return
	}
	log.prune(now - log.window)
	log.moves = append(log.moves, manaMove{at: now, amount: amount})
}

// prune drops moves at or before the cutoff. Re-slicing keeps the backing array; append
// copies only the live tail when it grows, so the log stays bounded by the window.
func (log *manaFlowLog) prune(cutoff time.Duration) {
	first := 0
	for first < len(log.moves) && log.moves[first].at <= cutoff {
		first++
	}
	log.moves = log.moves[first:]
}

func (log *manaFlowLog) reset() {
	log.moves = log.moves[:0]
}

// perSecond is the mana moved in (now - window, now] over the window, or over the time
// elapsed when the fight is younger than the window.
func (log *manaFlowLog) perSecond(now time.Duration, window time.Duration) float64 {
	span := min(window, now)
	if span <= 0 {
		return 0
	}
	total := 0.0
	for _, move := range log.moves {
		if move.at > now-window {
			total += move.amount
		}
	}
	return total / span.Seconds()
}
