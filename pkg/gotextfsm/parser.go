package gotextfsm

import (
	"fmt"
	"regexp"
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

func (t TextFSM) validateFSM() error {
	// Check that the Start state is always present
	if _, present := t.rules["Start"]; !present {
		return fmt.Errorf("Invalid FSM: 'Start' should always be present")
	}

	// Check that End and EOF state are always empty
	// when explicitly declared
	for _, s := range [2]string{"End", "EOF"} {
		if rules, present := t.rules[s]; present && len(rules) > 0 {
			return fmt.Errorf("Invalid FSM: State '%s', if declared, should be empty", s)
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
				return fmt.Errorf("Invalid FSM: Pointer to unknown state '%s' in rules of state %s",
					r.new_state,
					state)
			}
		}
	}

	return nil
}
