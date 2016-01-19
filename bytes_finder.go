package main

import "bytes"

// adapted from core strings/search.go

// BytesFinder efficiently finds a byte pattern in a source bytes. It's
// implemented using the Boyer-Moore string search algorithm:
// http://en.wikipedia.org/wiki/Boyer-Moore_string_search_algorithm
// http://www.cs.utexas.edu/~moore/publications/fstrpos.pdf (note: this aged
// document uses 1-based indexing)
type BytesFinder struct {
	// pattern is the string that we are searching for.
	pattern []byte

	// badCharSkip[b] contains the distance between the last byte of pattern
	// and the rightmost occurrence of b in pattern. If b is not in pattern,
	// badCharSkip[b] is len(pattern).
	//
	// Whenever a mismatch is found with byte b in the text, we can safely
	// shift the matching frame at least badCharSkip[b] until the next time
	// the matching char could be in alignment.
	badCharSkip [256]int

	// goodSuffixSkip[i] defines how far we can shift the matching frame given
	// that the suffix pattern[i+1:] matches, but the byte pattern[i] does
	// not. There are two cases to consider:
	//
	// 1. The matched suffix occurs elsewhere in pattern (with a different
	// byte preceding it that we might possibly match). In this case, we can
	// shift the matching frame to align with the next suffix chunk. For
	// example, the pattern "mississi" has the suffix "issi" next occurring
	// (in right-to-left order) at index 1, so goodSuffixSkip[3] ==
	// shift+len(suffix) == 3+4 == 7.
	//
	// 2. If the matched suffix does not occur elsewhere in pattern, then the
	// matching frame may share part of its prefix with the end of the
	// matching suffix. In this case, goodSuffixSkip[i] will contain how far
	// to shift the frame to align this portion of the prefix to the
	// suffix. For example, in the pattern "abcxxxabc", when the first
	// mismatch from the back is found to be in position 3, the matching
	// suffix "xxabc" is not found elsewhere in the pattern. However, its
	// rightmost "abc" (at position 6) is a prefix of the whole pattern, so
	// goodSuffixSkip[3] == shift+len(suffix) == 6+5 == 11.
	goodSuffixSkip []int

	// last is the index of the last character in the pattern.
	last int
}

func NewBytesFinder(pattern []byte) *BytesFinder {
	f := &BytesFinder{
		pattern: pattern,
		last:    len(pattern) - 1,
	}
	f.buildBadCharSkip()
	f.buildGoodSuffixSkip()
	return f
}

func (f *BytesFinder) buildBadCharSkip() {
	// Build bad character table.
	// Bytes not in the pattern can skip one pattern's length.
	for i := range f.badCharSkip {
		f.badCharSkip[i] = len(f.pattern)
	}
	// The loop condition is < instead of <= so that the last byte does not
	// have a zero distance to itself. Finding this byte out of place implies
	// that it is not in the last position.
	for i := 0; i < f.last; i++ {
		f.badCharSkip[f.pattern[i]] = f.last - i
	}
}

func (f *BytesFinder) buildGoodSuffixSkip() {
	// Build good suffix table.
	// First pass: set each value to the next index which starts a prefix of
	// pattern.
	f.goodSuffixSkip = make([]int, len(f.pattern))
	lastPrefix := f.last
	for i := f.last; i >= 0; i-- {
		if bytes.HasPrefix(f.pattern, f.pattern[i+1:]) {
			lastPrefix = i + 1
		}
		// lastPrefix is the shift, and (last-i) is len(suffix).
		f.goodSuffixSkip[i] = lastPrefix + f.last - i
	}
	// Second pass: find repeats of pattern's suffix starting from the front.
	for i := 0; i < f.last; i++ {
		lenSuffix := longestCommonSuffix(f.pattern, f.pattern[1:i+1])
		if f.pattern[i-lenSuffix] != f.pattern[f.last-lenSuffix] {
			// (last-i) is the shift, and lenSuffix is len(suffix).
			f.goodSuffixSkip[f.last-lenSuffix] = lenSuffix + f.last - i
		}
	}
}

func (f *BytesFinder) Find(data []byte) (bool, int) {
	// excluded is the number of prefix bytes that cannot contain a match
	excluded := 0

	i := f.last
	for i < len(data) {
		// Compare backwards from the end until the first unmatching character.
		j := f.last
		for j >= 0 && data[i] == f.pattern[j] {
			i--
			j--
		}
		if j < 0 {
			return true, i + 1 // match
		}
		excluded = max(0, i)
		i += max(f.badCharSkip[data[i]], f.goodSuffixSkip[j])
	}

	return false, excluded
}

// SplitUntil is a bufio.SplitFunc which will scan until the first occurance of
// pattern and return the bytes preceding the pattern as token.
func (f *BytesFinder) SplitUntil(data []byte, atEOF bool) (int, []byte, error) {
	if found, index := f.Find(data); found {
		return index, data[:index], nil
	} else if atEOF {
		return len(data), nil, nil
	} else {
		return index, nil, nil
	}
}

// SplitThru is a bufio.SplitFunc which will scan thru the first occurance of
// pattern and return the bytes preceding the pattern as token.
func (f *BytesFinder) SplitThru(data []byte, atEOF bool) (int, []byte, error) {
	if found, index := f.Find(data); found {
		end := index + len(f.pattern)
		token := data[:index]
		return end, token, nil
	} else if atEOF {
		return len(data), nil, nil
	} else {
		return index, nil, nil
	}
}

// SplitJust is a bufio.SplitFunc which will scan for the pattern token
func (f *BytesFinder) SplitJust(data []byte, atEOF bool) (int, []byte, error) {
	if found, index := f.Find(data); found {
		end := index + len(f.pattern)
		token := data[index:end]
		return end, token, nil
	} else if atEOF {
		return len(data), nil, nil
	} else {
		return index, nil, nil
	}
}

func longestCommonSuffix(a, b []byte) (i int) {
	for ; i < len(a) && i < len(b); i++ {
		if a[len(a)-1-i] != b[len(b)-1-i] {
			break
		}
	}
	return
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
