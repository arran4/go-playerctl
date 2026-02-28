package playerctl

import "strings"

// PlayerName contains connection information that fully qualifies a potential connection to a player.
type PlayerName struct {
	Name     string
	Instance string
	Source   Source
}

// NewPlayerName creates a new PlayerName instance.
// instance is the complete name and instance of the player.
func NewPlayerName(instance string, source Source) *PlayerName {
	parts := strings.SplitN(instance, ".", 2)
	name := parts[0]
	return &PlayerName{
		Name:     name,
		Instance: instance,
		Source:   source,
	}
}

// Compare compares two PlayerNames. It returns 0 if they are equal, otherwise non-zero.
func (p *PlayerName) Compare(other *PlayerName) int {
	if p.Source != other.Source {
		return 1
	}
	if p.Instance == other.Instance {
		return 0
	}
	if p.Instance < other.Instance {
		return -1
	}
	return 1
}

// InstanceCompare compares a PlayerName to another PlayerName treating the second as an instance matcher.
// Returns 0 if they match, otherwise non-zero.
func (p *PlayerName) InstanceCompare(other *PlayerName) int {
	if p.Source != other.Source {
		return 1
	}
	return StringInstanceCompare(p.Instance, other.Instance)
}

// StringInstanceCompare compares a player instance string with a matcher string.
// Supports "%any" matcher and partial instance matching (e.g., "vlc" matches "vlc.instanceXXXX").
func StringInstanceCompare(name, instance string) int {
	if name == "%any" || instance == "%any" {
		return 0
	}

	exactMatch := name == instance

	instanceMatch := !exactMatch && ((strings.HasPrefix(instance, name) &&
		len(instance) > len(name) &&
		strings.HasPrefix(instance[len(name):], ".")) ||
		(strings.HasPrefix(name, instance) &&
		len(name) > len(instance) &&
		strings.HasPrefix(name[len(instance):], ".")))

	if exactMatch || instanceMatch {
		return 0
	}

	return 1
}
