package root

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func parseCorpusHeader(t *testing.T, text string) (seed uint64, order []Faction, first Faction, body []string) {
	sc := bufio.NewScanner(strings.NewReader(text))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "%") {
			body = append(body, line)
			continue
		}
		f := strings.Fields(line)
		switch f[0] {
		case "%Game":
			s := f[1]
			if i := strings.LastIndex(s, "-"); i >= 0 {
				s = s[i+1:]
			}
			n, _ := strconv.ParseUint(s, 10, 64)
			seed = n
		case "%Faction":
			order = append(order, Faction(f[1]))
		case "%First":
			first = Faction(f[1])
		}
	}
	return
}

// TestApplyRMNGoldenCorpus replays every generated game through ApplyRMN and
// checks the final state hash matches. It also reports handler coverage.
func TestApplyRMNGoldenCorpus(t *testing.T) {
	files, _ := filepath.Glob("../../testdata/corpus/*.rmn")
	if len(files) == 0 {
		t.Skip("no corpus")
	}
	rmnHandlerHits, rmnFallbackHits = 0, 0
	unresolved := map[string]int{}
	for _, f := range files {
		text, _ := os.ReadFile(f)
		seed, order, first, body := parseCorpusHeader(t, string(text))
		g := NewGame(order, first, seed)
		BeginSetup(g)
		for _, line := range body {
			if err := g.ApplyRMN(line); err != nil {
				unresolved[strings.Fields(line)[3]]++
				t.Fatalf("%s: %q: %v", filepath.Base(f), line, err)
			}
		}
		want, _ := os.ReadFile(strings.TrimSuffix(f, ".rmn") + ".hash")
		got, _ := Snapshot(g)["hash"].(string)
		if got != strings.TrimSpace(string(want)) {
			t.Fatalf("%s: hash %s != %s", filepath.Base(f), got, strings.TrimSpace(string(want)))
		}
	}
	t.Logf("validated %d games; via handler=%d via fallback=%d", len(files), rmnHandlerHits, rmnFallbackHits)
	for k, v := range rmnFallbackIntents {
		t.Logf("  fallback intent %-20s %d", k, v)
	}
}
