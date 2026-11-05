package core

import "time"

// DefaultManaIncomeWindow is the trailing window an APL manaIncomePerSecond value reads when
// its config leaves window_seconds at 0.
const DefaultManaIncomeWindow = 10 * time.Second

type manaGain struct {
	at     time.Duration
	amount float64
}

// manaIncomeLog keeps the mana a unit gained over the longest window any APL value reads.
// It records nothing until a value asks for it, so units whose rotations never read their
// income pay nothing in AddMana.
type manaIncomeLog struct {
	window time.Duration
	gains  []manaGain
}

func (log *manaIncomeLog) track(window time.Duration) {
	log.window = max(log.window, window)
}

func (log *manaIncomeLog) record(now time.Duration, amount float64) {
	if log.window == 0 || amount <= 0 {
		return
	}
	log.prune(now - log.window)
	log.gains = append(log.gains, manaGain{at: now, amount: amount})
}

// prune drops gains at or before the cutoff. Re-slicing keeps the backing array; append
// copies only the live tail when it grows, so the log stays bounded by the window.
func (log *manaIncomeLog) prune(cutoff time.Duration) {
	first := 0
	for first < len(log.gains) && log.gains[first].at <= cutoff {
		first++
	}
	log.gains = log.gains[first:]
}

func (log *manaIncomeLog) reset() {
	log.gains = log.gains[:0]
}

// perSecond is the mana gained in (now - window, now] over the window, or over the time
// elapsed when the fight is younger than the window.
func (log *manaIncomeLog) perSecond(now time.Duration, window time.Duration) float64 {
	span := min(window, now)
	if span <= 0 {
		return 0
	}
	total := 0.0
	for _, gain := range log.gains {
		if gain.at > now-window {
			total += gain.amount
		}
	}
	return total / span.Seconds()
}
