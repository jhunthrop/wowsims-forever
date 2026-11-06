package core

import (
	"fmt"
	"strings"
	"testing"
)

// Round 26 read the class suites' "No item with id: 12620" panic as Era
// test gear the Forever item database lacks. Both 12620 and 12640 are in
// assets/database/db.json: the suites ran without --tags=with_db, so the
// database was empty and the first head item of the gear set panicked. An
// unknown item without the compiled-in database says so.
func TestAnUnknownItemWithoutTheDatabaseNamesTheBuildTag(t *testing.T) {
	if WITH_DB {
		t.Skip("the item database is compiled in")
	}
	defer func() {
		msg := fmt.Sprint(recover())
		if !strings.Contains(msg, "No item with id: 12620") || !strings.Contains(msg, "--tags=with_db") {
			t.Fatalf("NewItem panic = %q, want the id and the --tags=with_db hint", msg)
		}
	}()
	NewItem(ItemSpec{ID: 12620})
}
