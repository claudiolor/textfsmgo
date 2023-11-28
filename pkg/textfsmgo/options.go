package textfsmgo

// Enum for the available filling options
type FillOption int

const (
	NO_FILL_OP   = 0
	FILL_UP_OP   = 1
	FILL_DOWN_OP = 2
)

// Enum for the available line operations
type LineOperation string

const (
	CONTINUE_LINE_OP = "Continue"
	NEXT_LINE_OP     = "Next"
)

// Enum for the available record operations
type RecordOperation string

const (
	CLEAR_REC_OP     = "Clear"
	CLEAR_ALL_REC_OP = "Clearall"
	RECORD_REC_OP    = "Record"
	NO_RECORD_REC_OP = "NoRecord"
)

// Enum representing the type of value
type RecordType int

const (
	STRING_RECORD = 0
	LIST_RECORD   = 1
)

const START_STATE = "Start"

var LINE_OP = []string{CONTINUE_LINE_OP, NEXT_LINE_OP}
var WITH_ARGUMENT_OP = []string{"Error"}
var RECORD_OP = []string{CLEAR_REC_OP, CLEAR_ALL_REC_OP, RECORD_REC_OP, NO_RECORD_REC_OP}

var STOP_STATES = []string{"End", "EOF"}
