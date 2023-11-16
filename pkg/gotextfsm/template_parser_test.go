package gotextfsm

import (
	"bufio"
	"bytes"
	"fmt"
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

var ruleTestVars = map[string]TextFSMValue{
	"var1": {
		regex: "(?P<var1>.*)",
	},
	"var2": {
		regex: "(?P<var2>.*)",
	},
}

var ruleTestCases = []struct {
	description        string
	line               string
	exp_err            string
	exp_data_structure map[string]TextFSMRule
}{
	// Invalid use cases
	{
		description: "Test missing ^ before the rule",
		line:        "hello ${var1}",
		exp_err:     `.*missing \^ in rule definition.*`,
	},
	{
		description: "Test not existing variable",
		line:        "^hello ${donotexist}",
		exp_err:     `.*unknown variable.*`,
	},
	{
		description: "Test circular state reference",
		line:        "^hello ${var1} -> curstate",
		exp_err:     `.*circular pointer.*`,
	},
	{
		description: "Test new state with line op different than next",
		line:        "^hello ${var1} -> Continue newstate",
		exp_err:     `.*new state cannot be specified with line operation.*`,
	},
	{
		description: "Test badly constructed rule",
		line:        "^hello ${var1} -> Record,Continue",
		exp_err:     `.*badly formatted rule.*`,
	},
	{
		description: "Test match group in regex",
		line:        ` ^hello (?P<name>\w+) ${var2}`,
		exp_err:     `.*match groups are not allowed.*`,
	},
	{
		description: "Test invalid regex",
		line:        ` ^hello (?P<name>\w+`,
		exp_err:     `.*invalid regex.*`,
	},
	// Valid use cases
	{
		description:        "Test comment",
		line:               ` # This is a comment`,
		exp_data_structure: map[string]TextFSMRule{},
	},
	{
		description: "Test no operation",
		line:        "^Hello ${var1}",
		exp_data_structure: map[string]TextFSMRule{
			"curstate": {
				regex: regexp.MustCompile(`^Hello (?P<var1>.*)`),
			},
		},
	},
	{
		description: "Test Next line operation",
		line:        fmt.Sprintf("^Hello ${var1} -> %s", NEXT_LINE_OP),
		exp_data_structure: map[string]TextFSMRule{
			"curstate": {
				regex:   regexp.MustCompile(`^Hello (?P<var1>.*)`),
				line_op: NEXT_LINE_OP,
			},
		},
	},
	{
		description: "Test Continue line operation",
		line:        fmt.Sprintf("^Hello ${var1} -> %s", CONTINUE_LINE_OP),
		exp_data_structure: map[string]TextFSMRule{
			"curstate": {
				regex:   regexp.MustCompile(`^Hello (?P<var1>.*)`),
				line_op: CONTINUE_LINE_OP,
			},
		},
	},
	{
		description: "Test Record record operation",
		line:        fmt.Sprintf("^Hello ${var1} -> %s", RECORD_REC_OP),
		exp_data_structure: map[string]TextFSMRule{
			"curstate": {
				regex:  regexp.MustCompile(`^Hello (?P<var1>.*)`),
				rec_op: RECORD_REC_OP,
			},
		},
	},
	{
		description: "Test Clear record operation",
		line:        fmt.Sprintf("^Hello ${var1} -> %s", CLEAR_REC_OP),
		exp_data_structure: map[string]TextFSMRule{
			"curstate": {
				regex:  regexp.MustCompile(`^Hello (?P<var1>.*)`),
				rec_op: CLEAR_REC_OP,
			},
		},
	},
	{
		description: "Test Clear All record operation",
		line:        fmt.Sprintf("^Hello ${var1} -> %s", CLEAR_ALL_REC_OP),
		exp_data_structure: map[string]TextFSMRule{
			"curstate": {
				regex:  regexp.MustCompile(`^Hello (?P<var1>.*)`),
				rec_op: CLEAR_ALL_REC_OP,
			},
		},
	},
	{
		description: "Test NoRecord record operation",
		line:        fmt.Sprintf("^Hello ${var1} -> %s", NO_RECORD_REC_OP),
		exp_data_structure: map[string]TextFSMRule{
			"curstate": {
				regex:  regexp.MustCompile(`^Hello (?P<var1>.*)`),
				rec_op: NO_RECORD_REC_OP,
			},
		},
	},
	{
		description: "Test Error operation NO message",
		line:        "^Hello ${var1} -> Error",
		exp_data_structure: map[string]TextFSMRule{
			"curstate": {
				regex:     regexp.MustCompile(`^Hello (?P<var1>.*)`),
				error_str: "NoMessage",
			},
		},
	},
	{
		description: "Test Error operation with message",
		line:        `^Hello ${var1} -> Error "This is an error"`,
		exp_data_structure: map[string]TextFSMRule{
			"curstate": {
				regex:     regexp.MustCompile(`^Hello (?P<var1>.*)`),
				error_str: `"This is an error"`,
			},
		},
	},
	{
		description: "Test Line + Record operation",
		line:        fmt.Sprintf("^Hello ${var1} -> %s.%s", CONTINUE_LINE_OP, NO_RECORD_REC_OP),
		exp_data_structure: map[string]TextFSMRule{
			"curstate": {
				regex:   regexp.MustCompile(`^Hello (?P<var1>.*)`),
				rec_op:  NO_RECORD_REC_OP,
				line_op: CONTINUE_LINE_OP,
			},
		},
	},
	{
		description: "Test Record + New state operation",
		line:        fmt.Sprintf("^Hello ${var1} -> %s new_state", RECORD_REC_OP),
		exp_data_structure: map[string]TextFSMRule{
			"curstate": {
				regex:     regexp.MustCompile(`^Hello (?P<var1>.*)`),
				rec_op:    RECORD_REC_OP,
				new_state: "new_state",
			},
		},
	},
	{
		description: "Test Next line op + New state operation",
		line:        fmt.Sprintf("^Hello ${var1} -> %s new_state", NEXT_LINE_OP),
		exp_data_structure: map[string]TextFSMRule{
			"curstate": {
				regex:     regexp.MustCompile(`^Hello (?P<var1>.*)`),
				line_op:   NEXT_LINE_OP,
				new_state: "new_state",
			},
		},
	},
}

func TestParseTemplateRules(t *testing.T) {
	textFSM := TextFSM{
		values: map[string]TextFSMValue{},
	}

	for _, tc := range ruleTestCases {
		t.Log(tc.description)
		textFSM.values = ruleTestVars
		textFSM.rules = map[string][]TextFSMRule{}
		reader := bytes.NewBufferString(tc.line)
		scanner := bufio.NewScanner(reader)

		if err := textFSM.parseStateRules("curstate", scanner); err != nil {
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
		// Check whether the expected data structure is matched
		for state, data_structure := range tc.exp_data_structure {
			if _, present := textFSM.rules[state]; !present {
				t.Errorf("Error in '%s': value %s not created: %+v",
					tc.description, state, textFSM.rules)
				continue
			}

			got_ds := textFSM.rules[state]
			// Check if the two objects are equal
			if len(got_ds) != 1 {
				t.Errorf("Error in '%s': expected 1 rule in %s ruleset, got %d",
					tc.description, state, len(got_ds))
			}
			got_rule := got_ds[0]

			if got_rule.regex.String() != data_structure.regex.String() {
				t.Errorf("Error in '%s': expected regex '%s', got '%s'",
					tc.description, data_structure.regex.String(), data_structure.regex.String())
			}

			if got_rule.line_op != data_structure.line_op {
				t.Errorf("Error in '%s': expected line operation '%s', got '%s'",
					tc.description, got_rule.line_op, data_structure.line_op)
			}

			if got_rule.rec_op != data_structure.rec_op {
				t.Errorf("Error in '%s': expected rule operation '%s', got '%s'",
					tc.description, got_rule.rec_op, data_structure.rec_op)
			}

			if got_rule.new_state != data_structure.new_state {
				t.Errorf("Error in '%s': expected new state '%s', got '%s'",
					tc.description, got_rule.new_state, data_structure.new_state)
			}

			if got_rule.error_str != data_structure.error_str {
				t.Errorf("Error in '%s': expected error '%s', got '%s'",
					tc.description, got_rule.error_str, data_structure.error_str)
			}
		}
	}
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
