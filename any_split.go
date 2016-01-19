package main

type AnySplit struct {
	any   []byte
	match [256]int
	last  int
}

func NewAnySplit(any []byte) *AnySplit {
	as := &AnySplit{any: any, last: -1}
	for i := range as.match {
		as.match[i] = -1
	}
	for i, c := range any {
		as.match[c] = i
	}
	return as
}

func (as *AnySplit) Last() (byte, bool) {
	if as.last < 0 {
		return 0, false
	} else {
		return as.any[as.last], true
	}
}

func (as *AnySplit) Split(data []byte, atEOF bool) (int, []byte, error) {
	for i, c := range data {
		as.last = as.match[c]
		if as.last >= 0 {
			return i + 1, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
