package utils_test

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/claudiolor/textfsmgo/pkg/utils"
)

var testCases = []struct {
	description        string
	regex              string
	sentence           string
	exp_data_structure map[string]string
}{
	{
		description: "Simple regex with named groups",
		regex:       `(?P<name>\w+) I'm your (?P<relative>\w+)`,
		sentence:    "Luke I'm your cousin",
		exp_data_structure: map[string]string{
			"name":     "Luke",
			"relative": "cousin",
		},
	},
	{
		description: "Simple regex with simple and named groups",
		regex:       `(?P<name>\w+) I'm (your) (?P<relative>\w+)`,
		sentence:    "Luke I'm your cousin",
		exp_data_structure: map[string]string{
			"name":     "Luke",
			"relative": "cousin",
		},
	},
	{
		description:        "No match",
		regex:              `(?P<name>\w+) I'm your (?P<relative>\w+)`,
		sentence:           "Gianni",
		exp_data_structure: nil,
	},
	{
		description:        "No named match",
		regex:              `(\w+) I'm your (\w+)`,
		sentence:           "Gianni I'm your mum",
		exp_data_structure: nil,
	},
}

func TestGetRegexpNamedGroups(t *testing.T) {
	for _, tc := range testCases {
		t.Log(tc.description)
		re := regexp.MustCompile(tc.regex)
		res := utils.GetRegexpNamedGroups(re, re.FindStringSubmatch(tc.sentence))
		if !reflect.DeepEqual(tc.exp_data_structure, res) {
			t.Errorf("Error in '%s': expected %+v got %+v",
				tc.description, tc.exp_data_structure, res)
		}
	}
}
