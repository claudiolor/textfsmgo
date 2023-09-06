package gotextfsm

// valpointer type is a pointer to a value of a record
type valpointer interface{}

// ReturnVal is the type assigned to each value of a record
type ReturnVal struct {
	val   valpointer // pointer to the real value
	rtype RecordType // the type of value
}

// GetString() returns the string contained in the ReturnVal object and a boolean telling
// if the value has been correctly retrieved. If the ReturnVal does not contain a string,
// then an empty string is returned and a false telling that it was not possible to
// retrieve a string.
func (r ReturnVal) GetString() (string, bool) {
	if r.rtype != STRING_RECORD {
		return "", false
	}
	return *(r.val.(*string)), true
}

// GetSliceOfStrings() returns the slice of strings contained in the ReturnVal object and
// a boolean telling if the value has been correctly retrieved. If the ReturnVal does not
// contain a slice of strings, then nil is returned and a false telling that it
// was not possible to retrieve a string.
func (r ReturnVal) GetSliceOfStrings() ([]string, bool) {
	if r.rtype != LIST_RECORD {
		return nil, false
	}
	return *(r.val.(*[]string)), true
}
