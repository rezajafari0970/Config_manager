// Package smart composes independent, versioned multi-config exports.
package smart

import (
	"errors"
	"math/rand/v2"
	"sort"
)

const Version = "xray-26.7.28.smart.v2"

type Member struct {
	ID      string
	Source  string
	RawHash string
	Hash    string
	Expires int64
	Body    []byte
}
type Usage struct {
	Uses int64
	Last int64
}
type Ledger struct {
	Sources map[string]Usage
	Members map[string]Usage
}

func Key(m Member) string { return m.Source + "\x00" + m.ID }

// Select uses persistent per-source/per-member counts, not random sampling with
// replacement. A short source may cycle independently to keep its mandatory slot.
func Select(pool []Member, head string, count int, ledger Ledger) ([]Member, error) {
	if count < 1 {
		return nil, errors.New("smart_invalid_count")
	}
	groups := map[string][]Member{}
	leaders := []Member{}
	for _, m := range pool {
		groups[m.Source] = append(groups[m.Source], m)
		if m.ID == head {
			leaders = append(leaders, m)
		}
	}
	lessSource := func(a, b string) bool {
		x, y := ledger.Sources[a], ledger.Sources[b]
		if x.Uses != y.Uses {
			return x.Uses < y.Uses
		}
		if x.Last != y.Last {
			return x.Last < y.Last
		}
		return a < b
	}
	if len(leaders) == 0 {
		return nil, errors.New("smart_head_unavailable")
	}
	sort.Slice(leaders, func(i, j int) bool { return lessSource(leaders[i].Source, leaders[j].Source) })
	chosen := []Member{leaders[0]}
	ids := map[string]bool{head: true}
	hashes := map[string]bool{leaders[0].Hash: true}
	covered := map[string]bool{}
	cover := func(m Member) {
		for _, other := range pool {
			if other.ID == m.ID || other.Hash == m.Hash {
				covered[other.Source] = true
			}
		}
	}
	cover(leaders[0])
	sources := []string{}
	for source := range groups {
		sources = append(sources, source)
	}
	sort.Slice(sources, func(i, j int) bool { return lessSource(sources[i], sources[j]) })
	for source := range groups {
		sort.Slice(groups[source], func(i, j int) bool {
			a, b := groups[source][i], groups[source][j]
			x, y := ledger.Members[Key(a)], ledger.Members[Key(b)]
			if x.Uses != y.Uses {
				return x.Uses < y.Uses
			}
			if x.Last != y.Last {
				return x.Last < y.Last
			}
			return a.ID < b.ID
		})
	}
	pick := func(source string) bool {
		for _, m := range groups[source] {
			if !ids[m.ID] && !hashes[m.Hash] {
				chosen = append(chosen, m)
				ids[m.ID] = true
				hashes[m.Hash] = true
				cover(m)
				return true
			}
		}
		return false
	}
	// First cover distinct sources; shared canonical identities or identical bytes may cover several.
	for _, source := range sources {
		if len(chosen) >= count {
			break
		}
		if !covered[source] {
			pick(source)
		}
	}
	for len(chosen) < count {
		progress := false
		for _, source := range sources {
			if len(chosen) >= count {
				break
			}
			if pick(source) {
				progress = true
			}
		}
		if !progress {
			break
		}
	}
	// The head is fixed once per current cached profile; every other position is random.
	rand.Shuffle(len(chosen)-1, func(i, j int) { chosen[i+1], chosen[j+1] = chosen[j+1], chosen[i+1] })
	return chosen, nil
}
