package dockertest

import "strings"

func matchesAllLabels(have map[string]string, want []string) bool {
	for _, l := range want {
		if !matchesLabel(have, l) {
			return false
		}
	}
	return true
}

// matchesLabel matches a single Docker label filter. A bare "key" matches any
// container with that label set; "key=value" requires the exact pair.
func matchesLabel(have map[string]string, want string) bool {
	if k, v, ok := strings.Cut(want, "="); ok {
		return have[k] == v
	}
	_, ok := have[want]
	return ok
}
