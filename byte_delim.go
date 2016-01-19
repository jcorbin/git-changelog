package main

type ByteDelim struct {
	delim, skip, stop []byte
	isDelim           [256]int
	isSkip            [256]int
	isStop            [256]int
	lastDelim         int
	lastStop          int
}

func NewByteDelim(delim, skip, stop []byte) *ByteDelim {
	bd := &ByteDelim{
		delim: delim,
		skip:  skip,
		stop:  stop,
	}

	for i := range bd.isDelim {
		bd.isDelim[i] = -1
	}
	for i, c := range delim {
		bd.isDelim[c] = i
	}

	for i := range bd.isSkip {
		bd.isSkip[i] = -1
	}
	for i, c := range skip {
		bd.isSkip[c] = i
	}

	for i := range bd.isStop {
		bd.isStop[i] = -1
	}
	for i, c := range stop {
		bd.isStop[c] = i
	}

	return bd
}

func (bd *ByteDelim) Delim() (byte, bool) {
	if bd.lastDelim < 0 {
		return 0, false
	} else {
		return bd.delim[bd.lastDelim], true
	}
}

func (bd *ByteDelim) Stop() (byte, bool) {
	if bd.lastStop < 0 {
		return 0, false
	} else {
		return bd.stop[bd.lastStop], true
	}
}

func (bd *ByteDelim) Split(data []byte, atEOF bool) (int, []byte, error) {
	i := 0

	// find first delim
delimscan:
	for i < len(data) {
		c := data[i]
		bd.lastDelim = bd.isDelim[c]
		bd.lastStop = bd.isStop[c]
		if bd.lastDelim >= 0 {
			break delimscan
		} else if bd.lastStop >= 0 {
			// found stop byte, consume and skip
			j := i + 1
			return j, data[i:i], nil
		}
		i++
	}

	// find first non-skip byte
	j := i + 1
	for j < len(data) {
		c := data[j]
		if bd.isSkip[c] < 0 {
			break
		}
		j++
	}

	if j >= len(data) {
		// if data ends with a skip byte, we have to go again (or find no match
		// at eof)
		if atEOF {
			return len(data), nil, nil
		} else {
			return 0, nil, nil
		}
	}

	return j, data[:i], nil
}
