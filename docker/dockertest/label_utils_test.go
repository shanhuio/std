package dockertest

import "testing"

func TestMatchesLabel(t *testing.T) {
	for _, c := range []struct {
		name string
		have map[string]string
		want string
		ok   bool
	}{
		{"key-only match", map[string]string{"role": "frontend"}, "role", true},
		{"key-only miss", map[string]string{"role": "frontend"}, "tier", false},
		{"key-only on nil map", nil, "role", false},
		{"key=value match", map[string]string{"role": "frontend"}, "role=frontend", true},
		{"key=value wrong value", map[string]string{"role": "frontend"}, "role=backend", false},
		{"key=value missing key", map[string]string{"role": "frontend"}, "tier=edge", false},
		{"key= matches empty value", map[string]string{"role": ""}, "role=", true},
		{"key= rejects non-empty value", map[string]string{"role": "frontend"}, "role=", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := matchesLabel(c.have, c.want); got != c.ok {
				t.Errorf(
					"matchesLabel(%v, %q) = %t, want %t",
					c.have, c.want, got, c.ok,
				)
			}
		})
	}
}

func TestMatchesAllLabels(t *testing.T) {
	have := map[string]string{"role": "frontend", "tier": "edge"}
	for _, c := range []struct {
		name string
		want []string
		ok   bool
	}{
		{"empty filter matches anything", nil, true},
		{"single match", []string{"role=frontend"}, true},
		{"single miss", []string{"role=backend"}, false},
		{"all match", []string{"role=frontend", "tier=edge"}, true},
		{"one of two misses", []string{"role=frontend", "tier=core"}, false},
		{"mix of key-only and key=value", []string{"role", "tier=edge"}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := matchesAllLabels(have, c.want); got != c.ok {
				t.Errorf(
					"matchesAllLabels(%v, %v) = %t, want %t",
					have, c.want, got, c.ok,
				)
			}
		})
	}
}
