package gotextfsm

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/claudiolor/gotextfsm/pkg/utils"
	"golang.org/x/exp/slices"
)

const VALUE_FORMAT = "Value VARNAME [Flags, comma separated (no spaces)] (regex surraunded by round brackets)"

var STATE_NAME_REGEX = regexp.MustCompile(`^\w+$`)
var RULE_REGEX = regexp.MustCompile(`(?P<match>.*)\s->(?P<action>.*)`)
var VARIABLE_REGEX = regexp.MustCompile(`\${\w+}`)

var STATE_ACTION_REGEX_STR = `(?P<newstate>\w+)`

var LINE_REC_ACTION_REGEX = regexp.MustCompile(
	fmt.Sprintf(`^(?P<lineop>%s)(\.(?P<recop>%s))?(\s+%s)?$`,
		strings.Join(LINE_OP, "|"),
		strings.Join(RECORD_OP, "|"),
		STATE_ACTION_REGEX_STR,
	),
)

var LINE_ACTION_REGEX = regexp.MustCompile(
	fmt.Sprintf(`^(?P<recop>%s)(\s+%s)?$`,
		strings.Join(RECORD_OP, "|"),
		STATE_ACTION_REGEX_STR,
	),
)

var ERROR_ACTION_REGEX = regexp.MustCompile(`^Error(?: (\".*\"|\w+))?$`)

var STATE_ACTION_REGEX = regexp.MustCompile(
	fmt.Sprintf(`^%s$`, STATE_ACTION_REGEX_STR),
)

func isComment(line *string) bool {
	return strings.HasPrefix(*line, "#")
}

func (t *TextFSM) getNextLine(t_file_scanner *bufio.Scanner) (string, int, int) {
	t.template_parsed_line += 1
	line_no := t.template_parsed_line
	current_line := strings.TrimSpace(t_file_scanner.Text())
	return current_line, line_no, len(current_line)
}

func (t *TextFSM) parseTemplateFile(template_file string) error {
	t_file, err := os.Open(template_file)
	if err != nil {
		return err
	}
	defer t_file.Close()

	t_file_scanner := bufio.NewScanner(t_file)
	if err := t.parseTemplateFileValues(t_file_scanner); err != nil {
		return err
	}

	if err := t.parseTemplateFileStates(t_file_scanner); err != nil {
		return err
	}

	return nil
}

func (t *TextFSM) parseStateRules(state_name string, t_file_scanner *bufio.Scanner) error {
	t.rules[state_name] = []TextFSMRule{}
	for t_file_scanner.Scan() {
		current_line, line_no, line_size := t.getNextLine(t_file_scanner)
		if line_size == 0 {
			// If the line is empty we can skip to the next section
			break
		}

		// Ignore comments
		if isComment(&current_line) {
			continue
		}

		new_rule := TextFSMRule{}

		if !strings.HasPrefix(current_line, "^") {
			return fmt.Errorf("error in line %d: missing ^ in rule definition", line_no)
		}

		rule_match := RULE_REGEX.FindStringSubmatch(current_line)
		regex_str := current_line
		actions_str := ""

		if len(rule_match) != 0 {
			regex_str = rule_match[1]
			actions_str = strings.TrimSpace(rule_match[2])
		}

		// Replace the variables in the regex
		variables := VARIABLE_REGEX.FindAllString(regex_str, -1)
		for _, v := range variables {
			var_reg, found := t.values[v[2:len(v)-1]]
			if !found {
				return fmt.Errorf(
					"error in line %d: unknown variable %s in %s", line_no, v, current_line)
			}
			regex_str = strings.Replace(regex_str, v, var_reg.regex, -1)
		}

		// Compile the regex and check its validity
		regex, err := regexp.Compile(regex_str)
		if err != nil {
			return fmt.Errorf("error in line %d: invalid regex %s", line_no, err)
		}
		new_rule.regex = regex

		// Parse the actions if provided
		var actions map[string]string
		if actions_str != "" {
			// Parse all the possible combinations of actions formats
			if submatch := LINE_REC_ACTION_REGEX.FindStringSubmatch(actions_str); submatch != nil {
				actions = utils.GetRegexpNamedGroups(LINE_REC_ACTION_REGEX, submatch)
			} else if submatch := LINE_ACTION_REGEX.FindStringSubmatch(actions_str); submatch != nil {
				actions = utils.GetRegexpNamedGroups(LINE_ACTION_REGEX, submatch)
			} else if submatch := ERROR_ACTION_REGEX.FindStringSubmatch(actions_str); submatch != nil {
				actions = make(map[string]string)
				if submatch[1] != "" {
					new_rule.error_str = submatch[1]
				} else {
					new_rule.error_str = "NoMessage"
				}
			} else if submatch := STATE_ACTION_REGEX.FindStringSubmatch(actions_str); submatch != nil {
				actions = utils.GetRegexpNamedGroups(STATE_ACTION_REGEX, submatch)
			} else {
				return fmt.Errorf(
					"error in line %d: badly formatted rule in %s", line_no, current_line)
			}

			// Store all the actions
			if line_op, present := actions["lineop"]; present {
				new_rule.line_op = LineOperation(line_op)
			}

			if rec_op, present := actions["recop"]; present {
				new_rule.rec_op = RecordOperation(rec_op)
			}

			if new_state, present := actions["newstate"]; present {
				// Return an error for circular references, they have no effects but it is a signal
				// of a user error. Better pointing it out
				if state_name == new_state {
					return fmt.Errorf(
						"error in line %d: circular pointer to new state %s in %s",
						line_no,
						new_state,
						current_line)
				}

				new_rule.new_state = new_state
			}

			// Validate the provided actions
			// A new state can be provided only with the Next line operation
			if new_rule.line_op != "" &&
				new_rule.line_op != NEXT_LINE_OP &&
				new_rule.new_state != "" {
				return fmt.Errorf(
					"error in line %d: a new state cannot be specified with line operation %s in %s",
					line_no,
					new_rule.line_op,
					current_line)
			}
		}
		t.rules[state_name] = append(t.rules[state_name], new_rule)
	}
	return nil
}

func (t *TextFSM) parseTemplateFileStates(t_file_scanner *bufio.Scanner) error {
	t.rules = map[string][]TextFSMRule{}
	for t_file_scanner.Scan() {
		current_line, line_no, line_size := t.getNextLine(t_file_scanner)

		// Ignore comments and empty lines
		if line_size == 0 || isComment(&current_line) {
			continue
		}

		if !STATE_NAME_REGEX.MatchString(current_line) ||
			slices.Contains(LINE_OP, current_line) ||
			slices.Contains(RECORD_OP, current_line) ||
			slices.Contains(WITH_ARGUMENT_OP, current_line) {
			return fmt.Errorf("error in line %d: invalid state name %s", line_no, current_line)
		}

		if err := t.parseStateRules(current_line, t_file_scanner); err != nil {
			return err
		}

	}
	return nil
}

func (t *TextFSM) parseTemplateFileValues(t_file_scanner *bufio.Scanner) error {
	t.fillup_vals = []string{}
	t.required_vals = []string{}
	for t_file_scanner.Scan() {
		current_line, line_no, line_size := t.getNextLine(t_file_scanner)
		if line_size == 0 {
			// If the line is empty we can skip to the next section
			break
		}

		// Ignore comments
		if isComment(&current_line) {
			continue
		}

		// Validate the Value line
		if !strings.HasPrefix(current_line, "Value") {
			return fmt.Errorf("error in line %d: expected Value token, got: %s", line_no, current_line)
		}

		tokens := strings.Split(current_line, " ")
		if token_n := len(tokens); token_n < 3 {
			return fmt.Errorf("error in line %d: the Value declaration doesn't follow the format: %s", line_no, VALUE_FORMAT)
		}

		var name string
		var options []string = nil
		var regex string
		if !strings.HasPrefix(tokens[2], "(") {
			name = tokens[2]
			// Probably some options have been provided
			options = strings.Split(tokens[1], ",")
			regex = strings.Join(tokens[3:], " ")
		} else {
			name = tokens[1]
			regex = strings.Join(tokens[2:], " ")
		}

		// Parse options
		fill_op := NO_FILL_OP
		required_op := false
		key_op := false
		rtype := STRING_RECORD
		for _, op := range options {
			if op == "Fillup" {
				if fill_op == NO_FILL_OP {
					fill_op = FILL_UP_OP
					t.fillup_vals = append(t.fillup_vals, name)
				} else {
					return fmt.Errorf("error in line %d: conflicting option %s", line_no, op)
				}
			} else if op == "Filldown" {
				if fill_op == NO_FILL_OP {
					fill_op = FILL_DOWN_OP
				} else {
					return fmt.Errorf("error in line %d: conflicting option %s", line_no, op)
				}
			} else if op == "Required" {
				required_op = true
				t.required_vals = append(t.required_vals, name)
			} else if op == "Key" {
				key_op = true
			} else if op == "List" {
				rtype = LIST_RECORD
			} else {
				return fmt.Errorf("error in line %d: unknown option %s", line_no, op)
			}
		}

		// Validate regex
		if regex[0] != '(' || regex[len(regex)-1] != ')' {
			return fmt.Errorf("error in line %d: regex should be enclosed by ()", line_no)
		}

		if _, err := regexp.Compile(regex); err != nil {
			return fmt.Errorf("error in line %d: invalid regex %s", line_no, err)
		}
		// Create a named match group
		regex = fmt.Sprintf("(?P<%s>%s)", name, regex[1:len(regex)-1])

		t.values[name] = TextFSMValue{
			regex:    regex,
			fill:     FillOption(fill_op),
			required: required_op,
			key:      key_op,
			rtype:    RecordType(rtype),
		}
	}
	return nil
}
