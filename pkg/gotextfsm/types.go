package gotextfsm

type valpointer interface{}

type ReturnVal struct {
	val   valpointer
	rtype RecordType
}

func (r ReturnVal) GetString() (string, bool) {
	if r.rtype != STRING_RECORD {
		return "", false
	}
	return *(r.val.(*string)), true
}

func (r ReturnVal) GetSliceOfStrings() ([]string, bool) {
	if r.rtype != LIST_RECORD {
		return nil, false
	}
	return *(r.val.(*[]string)), true
}
