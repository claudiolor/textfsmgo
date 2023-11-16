package utils

import (
	"encoding/json"
	"regexp"
)

// GetRegexpNamedGroups(*regexp.Regexp, []string) given a regular expression and the resulting submatch
// of the FindStringSubmatch() function, it returns a map with for each named group the
// corresponding match
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

// ConvertResToJson(*[]map[string]interface{}, bool) given the result of the textfsm parsed data
// returns the json output. When indent is true, the output will be indented
func ConvertResToJson(map_res *[]map[string]interface{}, indent bool) ([]byte, error) {
	var byteRes []byte
	var err error
	if indent {
		byteRes, err = json.MarshalIndent(*map_res, "", "  ")
	} else {
		byteRes, err = json.Marshal(*map_res)
	}

	if err != nil {
		return nil, err
	}

	return byteRes, nil
}
