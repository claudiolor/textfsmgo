package utils

import "regexp"

func GetRegexpNamedGroups(reg *regexp.Regexp, submatch []string) map[string]string {
	// Return nil in case of no submatch
	if submatch == nil {
		return nil
	}

	// Stores the named matches in a dictionary
	matches := map[string]string{}
	for i, gname := range reg.SubexpNames() {
		if i != 0 && gname != "" {
			matches[gname] = submatch[i]
		}
	}

	// If no named matches return nil
	if len(matches) == 0 {
		return nil
	}
	return matches
}
