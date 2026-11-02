package core

import (
	"encoding/csv"
	"os"
	"strconv"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Round 9 (casters lane): every level's base mana is the client's BaseMP
// table (basemp.txt, 1.60.1.70707). The per-level table came from wowhead's
// gear planner, which disagreed at 8 class levels: shaman 1, 22, 26, 30 and
// 52 (718 against the client's 665 at 30), priest 2 and 12, warlock 1.
func TestBaseManaAtEveryLevelIsTheClients(t *testing.T) {
	f, err := os.Open("../../assets/db_inputs/gametables/1.60.1.70707/basemp.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = '\t'
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	columns := map[string]proto.Class{
		"Paladin": proto.Class_ClassPaladin, "Hunter": proto.Class_ClassHunter, "Priest": proto.Class_ClassPriest,
		"Shaman": proto.Class_ClassShaman, "Mage": proto.Class_ClassMage, "Warlock": proto.Class_ClassWarlock,
		"Druid": proto.Class_ClassDruid,
	}
	for col, name := range rows[0] {
		class, ok := columns[name]
		if !ok {
			continue
		}
		for _, row := range rows[1:] {
			level, _ := strconv.Atoi(row[0])
			if level < 1 || level > CharacterMaxLevel {
				continue
			}
			want, _ := strconv.ParseFloat(row[col], 64)
			if got := baseStatsByClassLevel[class][level][stats.Mana]; got != want {
				t.Errorf("%s level %d base mana = %v, want the client's %v", name, level, got, want)
			}
		}
	}
}
