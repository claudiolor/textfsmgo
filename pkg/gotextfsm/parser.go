package gotextfsm

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/claudiolor/gotextfsm/pkg/utils"
	"golang.org/x/exp/slices"
)

type TextFSMValue struct {
	fill     FillOption
	key      bool
	regex    string
	rtype    RecordType
	required bool
}

type TextFSMRule struct {
	regex     *regexp.Regexp
	line_op   LineOperation
	rec_op    RecordOperation
	new_state string
	error_str string // If present when the rule matches return an error
}

type TextFSM struct {
	template_parsed_line int
	state                string
	fillup_vals          []string
	required_vals        []string
	records              []map[string]ReturnVal
	current_record       *map[string]ReturnVal
	values               map[string]TextFSMValue
	rules                map[string][]TextFSMRule
}

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

func (t *TextFSM) ParseTextToDicts(text string) ([]map[string]ReturnVal, error) {
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

func (t *TextFSM) ResetFSM() {
	t.current_record = nil
	t.state = START_STATE
	t.records = []map[string]ReturnVal{}
}

func (t TextFSM) isEmpty(val valpointer, rtype RecordType) bool {
	if rtype == STRING_RECORD && *(val.(*string)) == "" {
		return true
	} else if rtype == LIST_RECORD && len(*(val.(*[]string))) == 0 {
		return true
	}
	return false
}

func (t *TextFSM) generateEmptyRecord() map[string]ReturnVal {
	new_record := map[string]ReturnVal{}
	for k, val_prop := range t.values {
		if val_prop.rtype == STRING_RECORD {
			new_val := ""
			if val_prop.fill == FILL_DOWN_OP {
				n_records := len(t.records)
				if n_records > 0 {
					new_val = *(t.records[n_records-1][k].val.(*string))
				}
			}
			new_record[k] = ReturnVal{
				val:   valpointer(&new_val),
				rtype: val_prop.rtype,
			}
		} else {
			// If the record is a list
			new_val := []string{}
			if val_prop.fill == FILL_DOWN_OP {
				n_records := len(t.records)
				if n_records > 0 {
					new_val = *(t.records[n_records-1][k].val.(*[]string))
				}
			}
			new_record[k] = ReturnVal{
				val:   valpointer(&new_val),
				rtype: val_prop.rtype,
			}
		}
	}
	return new_record
}

func (t *TextFSM) setValue(key string, val string, current_record *map[string]ReturnVal) *map[string]ReturnVal {
	if current_record == nil {
		new_record := t.generateEmptyRecord()
		current_record = &new_record
	}

	rtype := t.values[key].rtype
	if rtype == STRING_RECORD {
		*((*current_record)[key].val).(*string) = val
	} else {
		// If the record is a list
		list_val_ptr := (*current_record)[key].val.(*[]string)
		*list_val_ptr = append(*list_val_ptr, val)
	}
	return current_record
}

// The clearRecord function implements the Clear operation, so it
// clear all the values stored so far, filldown excluded
func (t *TextFSM) clearRecord(current_record *map[string]ReturnVal) {
	*current_record = t.generateEmptyRecord()
}

// The clearAllRecord function implements the ClearAll operation, so it
// clear all the values stored so far
func (t *TextFSM) clearAllRecord(current_record *map[string]ReturnVal) {
	if current_record != nil {
		for _, v := range *current_record {
			switch v.val.(type) {
			case string:
				*(v.val.(*string)) = ""
			case []string:
				*(v.val.(*[]string)) = []string{}
			}
		}
	}
}

// Store the current record
func (t *TextFSM) appendRecord(current_record *map[string]ReturnVal) *map[string]ReturnVal {
	if current_record != nil {
		// Do not store if required records are not present
		for _, req_key := range t.required_vals {
			val_props := t.values[req_key]
			if t.isEmpty((*current_record)[req_key].val, val_props.rtype) {
				return nil
			}
		}

		last_index := len(t.records) - 1
		// Add the new record
		t.records = append(t.records, *current_record)

		// Fill up values if any
		if last_index != -1 {
			for _, fup_key := range t.fillup_vals {
				val_props := t.values[fup_key]
				if t.isEmpty((*current_record)[fup_key].val, val_props.rtype) {
					continue
				}

				fill_val := (*current_record)[fup_key].val
				for i := last_index; i >= 0; i-- {
					if t.isEmpty(t.records[i][fup_key].val, val_props.rtype) {
						break
					}
					new_val := t.records[i][fup_key]
					new_val.val = fill_val
					t.records[i][fup_key] = new_val
				}
			}
		}
	}
	return nil
}

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
