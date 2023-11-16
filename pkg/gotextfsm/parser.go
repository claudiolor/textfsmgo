// Package gotextfsm implements parsing of text via the textfsm templates
package gotextfsm

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/claudiolor/gotextfsm/pkg/utils"
	"golang.org/x/exp/slices"
)

// TextFSMValue is a representation of a Value of the template file
type TextFSMValue struct {
	fill     FillOption // Tells if the value should be filled if empty
	key      bool       // Tells if the value contribute to the unique identifier for a row
	regex    string     // The regex to match the value
	rtype    RecordType // Tells if the value is a string or a list
	required bool       // Tells if the value is required or not
}

// TextFSMRule is a representation of a rule in a textfsm state
type TextFSMRule struct {
	regex     *regexp.Regexp  // The regex to match the row
	line_op   LineOperation   // The line operation to perform when the rule is matched
	rec_op    RecordOperation // The record operation to perform when the rule is matched
	new_state string          // The new state to land on when the rule is matched
	error_str string          // If present when the rule matches return an error
}

// TextFSM is a representation of the state machine to perform parsing of semi-formatted
// text
type TextFSM struct {
	template_parsed_line int                      // last parsed line of the template
	state                string                   // current state of the fsm
	fillup_vals          []string                 // list of values with the fillup option enabled
	required_vals        []string                 // list of the required values of a row
	records              []map[string]interface{} // all the collected records
	current_record       *map[string]interface{}  // the record that the fsm is currently filling
	values               map[string]TextFSMValue  // the collection of values declared in the template
	rules                map[string][]TextFSMRule // the list of rules to match line against
}

// NewTextFsmParser(string) creates a new TextFSM object. The function gets the path to
// the template file describing the FSM. An error is returned when the template file is
// not valid.
// example: NewTextFSMParser(/path/to/template_file)
func NewTextFSMParser(template_file string) (*TextFSM, error) {
	new_parser := TextFSM{
		values: map[string]TextFSMValue{},
	}

	// Parse the template file and produce the FSM
	if err := new_parser.parseTemplateFile(template_file); err != nil {
		return nil, err
	}

	// Validate the state machine after the template parsing
	if err := new_parser.validateFSM(); err != nil {
		return nil, err
	}

	return &new_parser, nil
}

// ParseTextToDicts(string) parse the string provided as argument.
// Returns a map slice of maps with all the retrieved records
func (t *TextFSM) ParseTextToDicts(text string) ([]map[string]interface{}, error) {
	// We will first need to reset the state machine
	t.ResetFSM()
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if err := t.parseLine(line); err != nil {
			return nil, err
		}

		if slices.Contains(STOP_STATES, t.state) {
			break
		}
	}

	if _, eof_overwritten := t.rules["EOF"]; t.state != "End" && !eof_overwritten {
		t.appendRecord(t.current_record)
	}

	return t.records, nil
}

// ResetFSM() resets the FSM
func (t *TextFSM) ResetFSM() {
	t.current_record = nil
	t.state = START_STATE
	t.records = []map[string]interface{}{}
}

// isEmpty(interface{}, RecordType) returns a boolean telling if the given interface{}
// contains an empty value
func (t TextFSM) isEmpty(val interface{}) bool {
	switch val := val.(type) {
	case string:
		return val == ""
	case []string:
		return len(val) == 0
	}
	return false
}

// generateEmptyRecord() returns a map of a new record, filling the fields with all the
// "filldown" values if present, otherwise they are left blank
func (t *TextFSM) generateEmptyRecord() map[string]interface{} {
	new_record := map[string]interface{}{}
	for k, val_prop := range t.values {
		if val_prop.rtype == STRING_RECORD {
			new_val := ""
			if val_prop.fill == FILL_DOWN_OP {
				n_records := len(t.records)
				if n_records > 0 {
					new_val = t.records[n_records-1][k].(string)
				}
			}
			new_record[k] = new_val
		} else {
			// If the record is a list
			new_val := []string{}
			if val_prop.fill == FILL_DOWN_OP {
				n_records := len(t.records)
				if n_records > 0 {
					new_val = t.records[n_records-1][k].([]string)
				}
			}
			new_record[k] = new_val
		}
	}
	return new_record
}

// setValue(string, string, *map[string]interface{}) set the given value on the key of the
// map provided as argument. If the pointer to the map is null, a new one is created from
// scratch. The function returns back a pointer to the map where the value as been added
func (t *TextFSM) setValue(key string, val string, current_record *map[string]interface{}) *map[string]interface{} {
	if current_record == nil {
		new_record := t.generateEmptyRecord()
		current_record = &new_record
	}

	rtype := t.values[key].rtype
	if rtype == STRING_RECORD {
		(*current_record)[key] = val
	} else {
		// If the record is a list
		(*current_record)[key] = append((*current_record)[key].([]string), val)
	}
	return current_record
}

// clearRecord(*map[string]interface{}) implements the Clear operation, so it clear all the
// values stored so far, filldown excluded
func (t *TextFSM) clearRecord(current_record *map[string]interface{}) {
	*current_record = t.generateEmptyRecord()
}

// clearAllRecord(*map[string]interface{}) implements the ClearAll operation, so it
// clears all the values stored so far
func (t *TextFSM) clearAllRecord(current_record *map[string]interface{}) {
	if current_record != nil {
		for k, v := range *current_record {
			switch v.(type) {
			case string:
				(*current_record)[k] = ""
			case []string:
				(*current_record)[k] = []string{}
			}
		}
	}
}

// appendRecord(*map[string]interface{}) append the record filled so far to the list of
// records. It applies the fillup if any. The function returns a pointer to the new
// current value.
func (t *TextFSM) appendRecord(current_record *map[string]interface{}) *map[string]interface{} {
	if current_record != nil {
		// Do not store if required records are not present
		for _, req_key := range t.required_vals {
			if t.isEmpty((*current_record)[req_key]) {
				return nil
			}
		}

		last_index := len(t.records) - 1
		// Add the new record
		t.records = append(t.records, *current_record)

		// Fill up values if any
		if last_index != -1 {
			for _, fup_key := range t.fillup_vals {
				if t.isEmpty((*current_record)[fup_key]) {
					continue
				}

				fill_val := (*current_record)[fup_key]
				for i := last_index; i >= 0; i-- {
					if !t.isEmpty(t.records[i][fup_key]) {
						break
					}
					t.records[i][fup_key] = fill_val
				}
			}
		}
	}
	return nil
}

// parseLine(string) parses the line provided as argument checking if it matches one of
// the rules defined in the template, if so it fills the values in the current record
// and perform the related actions
func (t *TextFSM) parseLine(line string) error {
	for _, rule := range t.rules[t.state] {
		submatch := rule.regex.FindStringSubmatch(line)

		// Check if the next rule matches
		if submatch == nil {
			continue
		}

		detected_vars := utils.GetRegexpNamedGroups(rule.regex, submatch)

		// Check if we need to raise an error
		if rule.error_str != "" {
			return fmt.Errorf("state error raised by FSM: %s in %s", rule.error_str, line)
		}

		// Store the variables, if any
		for key, val := range detected_vars {
			t.current_record = t.setValue(key, val, t.current_record)
		}

		// Handle the record options
		switch rule.rec_op {
		case RECORD_REC_OP:
			t.current_record = t.appendRecord(t.current_record)
		case CLEAR_REC_OP:
			t.clearRecord(t.current_record)
		case CLEAR_ALL_REC_OP:
			t.clearAllRecord(t.current_record)
		}

		// Handle the line options
		if rule.line_op != CONTINUE_LINE_OP {
			// Apply the new state if needed
			if rule.new_state != "" {
				t.state = rule.new_state
			}
			break // parse the next line
		}

	}
	return nil
}

// validateFSM() checks if the defined FSM is valid and returns an error if this is the case
func (t TextFSM) validateFSM() error {
	// Check that the Start state is always present
	if _, present := t.rules[START_STATE]; !present {
		return fmt.Errorf("invalid FSM: '%s' should always be present", START_STATE)
	}

	// Check that End and EOF state are always empty
	// when explicitly declared
	for _, s := range STOP_STATES {
		if rules, present := t.rules[s]; present && len(rules) > 0 {
			return fmt.Errorf("invalid FSM: State '%s', if declared, should be empty", s)
		}
	}

	// Check that the pointers to states are all valid
	for state, rules := range t.rules {
		for _, r := range rules {
			// End or EOF are implicit states, no need to check if they have been declared
			if r.new_state == "End" || r.new_state == "EOF" || r.new_state == "" {
				continue
			}

			// Check if the new state is valid
			if _, present := t.rules[r.new_state]; !present {
				return fmt.Errorf("invalid FSM: Pointer to unknown state '%s' in rules of state %s",
					r.new_state,
					state)
			}
		}
	}

	return nil
}
