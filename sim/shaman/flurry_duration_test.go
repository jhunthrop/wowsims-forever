package shaman

import (
	"testing"
	"time"
)

// Flurry's buff (16257 at the talent's rank used here) carries SpellMisc
// DurationIndex 8, 15 s, in 1.60.1.70707: it ends after 15 s even with
// charges left.
func TestFlurryBuffLastsFifteenSeconds(t *testing.T) {
	if flurryBuffDuration != 15*time.Second {
		t.Errorf("Flurry's buff lasts %v, want 15s", flurryBuffDuration)
	}
}
