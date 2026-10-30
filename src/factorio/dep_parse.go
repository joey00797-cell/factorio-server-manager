package factorio

import (
	"regexp"
	"strconv"
	"strings"
)

// depSpec is one parsed entry of the "dependencies" list of a mod info.json.
type depSpec struct {
	Name         string
	Op           string
	Version      string
	Optional     bool
	Incompatible bool
}

var depVersionRe = regexp.MustCompile(`^(.*?)\s*(>=|<=|>|<|=)\s*([0-9][^\s]*)\s*$`)

// parseDepSpec parses "prefix name op version". Names may contain spaces.
// A "+" prefix is treated as optional: it is neither validated nor downloaded.
func parseDepSpec(dep string) (depSpec, bool) {
	dep = strings.TrimSpace(dep)
	var s depSpec
	for {
		switch {
		case strings.HasPrefix(dep, "(?)"):
			s.Optional = true
			dep = strings.TrimSpace(dep[3:])
			continue
		case strings.HasPrefix(dep, "!"):
			s.Incompatible = true
			dep = strings.TrimSpace(dep[1:])
			continue
		case strings.HasPrefix(dep, "?"), strings.HasPrefix(dep, "~"), strings.HasPrefix(dep, "+"):
			s.Optional = true
			dep = strings.TrimSpace(dep[1:])
			continue
		}
		break
	}
	if dep == "" {
		return s, false
	}
	if m := depVersionRe.FindStringSubmatch(dep); m != nil {
		s.Name, s.Op, s.Version = strings.TrimSpace(m[1]), m[2], m[3]
	} else {
		s.Name = dep
	}
	return s, s.Name != ""
}

func parseVersionParts(v string) ([]int, bool) {
	var out []int
	for _, p := range strings.Split(strings.TrimSpace(v), ".") {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, false
		}
		out = append(out, n)
	}
	return out, len(out) > 0
}

// versionSatisfies reports whether have <op> want holds.
// When a version cannot be parsed it returns true (no false alarms).
func versionSatisfies(have, op, want string) bool {
	a, ok1 := parseVersionParts(have)
	b, ok2 := parseVersionParts(want)
	if !ok1 || !ok2 {
		return true
	}
	cmp := 0
	for i := 0; i < len(a) || i < len(b); i++ {
		x, y := 0, 0
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			if x < y {
				cmp = -1
			} else {
				cmp = 1
			}
			break
		}
	}
	switch op {
	case ">=":
		return cmp >= 0
	case ">":
		return cmp > 0
	case "<=":
		return cmp <= 0
	case "<":
		return cmp < 0
	case "=":
		return cmp == 0
	}
	return true
}
