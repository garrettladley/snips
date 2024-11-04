package snips

import "strings"

func ContainsDotCodeDot(name string) bool {
	index := strings.LastIndex(name, ".code.")
	return index != -1 && index < len(name)-6
}
