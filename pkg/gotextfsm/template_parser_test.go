package gotextfsm

import (
	"bufio"
	"bytes"
	"reflect"
	"regexp"
	"testing"
)

var valTestCases = []struct {
	description        string
	line               string
	exp_err            string
	exp_data_structure map[string]TextFSMValue
}{
	{
		description: "Test value without options",
		line:        "Value myval (.*)",
		exp_data_structure: map[string]TextFSMValue{
			"myval": {
				regex: "(?P<myval>.*)",
			},
		},
	},
	{
		description: "Test value with additional brackets",
		line:        `Value myval (\(.*\) welcome!)`,
		exp_data_structure: map[string]TextFSMValue{
			"myval": {
				regex: `(?P<myval>\(.*\) welcome!)`,
			},
		},
	},
	{
		description: "Test if comments are ignored in Value section",
		line:        "# This is a comment \n Value myval (.*)",
		exp_data_structure: map[string]TextFSMValue{
			"myval": {
				regex: "(?P<myval>.*)",
			},
		},
	},
	// Test options
	{
		description: "Test value with list option",
		line:        "Value List myval (.*)",
		exp_data_structure: map[string]TextFSMValue{
			"myval": {
				regex: "(?P<myval>.*)",
				rtype: LIST_RECORD,
			},
		},
	},
	{
		description: "Test value with Fillup option",
		line:        "Value Fillup myval (.*)",
		exp_data_structure: map[string]TextFSMValue{
			"myval": {
				regex: "(?P<myval>.*)",
				fill:  FILL_UP_OP,
			},
		},
	},
	{
		description: "Test value with Filldown option",
		line:        "Value Filldown myval (.*)",
		exp_data_structure: map[string]TextFSMValue{
			"myval": {
				regex: "(?P<myval>.*)",
				fill:  FILL_DOWN_OP,
			},
		},
	},
	{
		description: "Test value with Required option",
		line:        "Value Required myval (.*)",
		exp_data_structure: map[string]TextFSMValue{
			"myval": {
				regex:    "(?P<myval>.*)",
				required: true,
			},
		},
	},
	{
		description: "Test option combination",
		line:        "Value Required,Fillup,List myval (.*)",
		exp_data_structure: map[string]TextFSMValue{
			"myval": {
				regex:    "(?P<myval>.*)",
				required: true,
				fill:     FILL_UP_OP,
				rtype:    LIST_RECORD,
			},
		},
	},
	// Invalid formats
	{
		description: "Test completly wrong format",
		line:        "Wrong wrong wrong!",
		exp_err:     ".*expected Value token.*",
	},
	{
		description: "Test match groups in regex",
		line:        "Value Required myval (Hostname (/s).*)",
		exp_err:     ".*match groups in values' regex are not supported.*",
	},
}

func TestParseTemplateFileValues(t *testing.T) {
	textFSM := TextFSM{
		values: map[string]TextFSMValue{},
	}

	for _, tc := range valTestCases {
		t.Log(tc.description)
		textFSM.values = map[string]TextFSMValue{}
		reader := bytes.NewBufferString(tc.line)
		scanner := bufio.NewScanner(reader)

		if err := textFSM.parseTemplateFileValues(scanner); err != nil {
			if tc.exp_err == "" {
				t.Errorf("Error in '%s': unexpected error '%s'", tc.description, err)
			} else {
				err_regex, reerr := regexp.Compile(tc.exp_err)
				if reerr != nil {
					t.Fatalf("Invalid test %s: regex is invalid '%s'", tc.description, reerr)
				}

				if !err_regex.MatchString(err.Error()) {
					t.Errorf("Error in '%s': '%s' error does not match pattern '%s'",
						tc.description, err, tc.exp_err)
				}
			}
			continue
		} else if tc.exp_err != "" {
			t.Errorf("Error in '%s': expected error '%s', no errors got",
				tc.description, tc.exp_err)
			continue
		}

		// Check if the values have been created
		if len(tc.exp_data_structure) != len(textFSM.values) {
			t.Errorf("Error in '%s': expected %d values got %d in %+v",
				tc.description, len(tc.exp_data_structure), len(textFSM.values), textFSM.values)
			continue
		}

		for name, data_structure := range tc.exp_data_structure {
			if _, present := textFSM.values[name]; !present {
				t.Errorf("Error in '%s': value %s not created: %+v",
					tc.description, name, textFSM.values)
				continue
			}

			got_ds := textFSM.values[name]
			if !reflect.DeepEqual(data_structure, got_ds) {
				t.Errorf("Error in '%s': expected %+v got %+v",
					tc.description, data_structure, got_ds)
			}
		}
	}
}
