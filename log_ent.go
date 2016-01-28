package main

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strings"
)

type LogEntScanner struct {
	scanner *bufio.Scanner
	err     error
	ent     LogEnt
	done    bool
}

func NewLogEntScanner(r io.Reader) *LogEntScanner {
	return &LogEntScanner{
		scanner: bufio.NewScanner(r),
	}
}

func (entScanner *LogEntScanner) Ent() *LogEnt {
	return &entScanner.ent
}

func (entScanner *LogEntScanner) Err() error {
	if entScanner.err != nil {
		return entScanner.err
	}
	return entScanner.scanner.Err()
}

func (entScanner *LogEntScanner) Scan() bool {
	if entScanner.done {
		return false
	}

	entScanner.ent.Reset()

	if !entScanner.scanCommit() {
		return false
	}

	if !entScanner.scanAttrs() {
		return false
	}

	if !entScanner.scanSubject() {
		return false
	}

	if !entScanner.scanMess() {
		return true
	}

	return true
}

func (entScanner *LogEntScanner) scanOne(f bufio.SplitFunc) bool {
	if entScanner.done {
		return false
	}
	entScanner.scanner.Split(f)
	if !entScanner.scanner.Scan() {
		entScanner.done = true
	}
	return !entScanner.done
}

func (entScanner *LogEntScanner) scanCommit() bool {
	// scan thru "commit "
	if !entScanner.scanOne(commitFinder.SplitJust) {
		return false
	}

	// scan thru space or newline
	if !entScanner.scanOne(spaceNLSplit.Split) {
		return false
	}
	entScanner.ent.commit = entScanner.scanner.Text()

	// consume the rest of the line if necessary
	if c, _ := spaceNLSplit.Last(); c != '\n' {
		if !entScanner.scanOne(bufio.ScanLines) {
			return false
		}
	}

	return true
}

func (entScanner *LogEntScanner) scanAttrs() bool {
	// scan all "key: val..." contiguous lines
	for {
		// scan "key: "
		if !entScanner.scanOne(keySplit.Split) {
			return false
		}
		keyBytes := entScanner.scanner.Bytes()
		if len(keyBytes) == 0 {
			// empty key means keySplit hit a newline before a :
			break
		}

		// scan value until newline
		if !entScanner.scanOne(bufio.ScanLines) {
			return false
		}

		entScanner.ent.attrs[string(keyBytes)] = entScanner.scanner.Text()
	}

	return true
}

func (entScanner *LogEntScanner) scanSubject() bool {
	// scan subject line
	str, ok := scanMessagePart(entScanner.scanner, " ")
	if !ok {
		return false
	}

	// if we have a PR merge, extract more annotations, and promote the next
	// message part
	if prMatch := prRegex.FindStringSubmatch(str); prMatch != nil {
		entScanner.ent.attrs["prNumber"] = prMatch[1]
		entScanner.ent.attrs["prFrom"] = prMatch[2]

		// TODO: would be nice to not accidentally skip any "^commit "
		if next, ok := scanMessagePart(entScanner.scanner, " "); ok {
			str = next
		}
	}

	entScanner.ent.subject = str
	return true
}

func (entScanner *LogEntScanner) scanMess() bool {
	// scan message until next "commit "
	if !entScanner.scanOne(commitFinder.SplitUntil) {
		return false
	}

	// scan message paragraphs
	messScanner := bufio.NewScanner(bytes.NewBuffer(entScanner.scanner.Bytes()))
	for {
		if str, ok := scanMessagePart(messScanner, "\n"); !ok {
			return len(entScanner.ent.mess) > 0
		} else {
			entScanner.ent.mess = append(entScanner.ent.mess, str)
		}
	}
}

func scanMessagePart(scanner *bufio.Scanner, sep string) (string, bool) {
	scanner.Split(bufio.ScanLines)
	var parts []string
	for scanner.Scan() {
		// TODO: consistent de-indent by first-line detection
		line := strings.TrimLeft(scanner.Text(), " ")
		if len(line) == 0 {
			return strings.Join(parts, sep), true
		}
		parts = append(parts, line)
	}
	if len(parts) > 0 {
		return strings.Join(parts, sep), true
	}
	return "", false
}

type LogEnt struct {
	commit  string
	subject string
	attrs   map[string]string
	mess    []string
}

func NewEnt() *LogEnt {
	return &LogEnt{
		attrs: make(map[string]string),
	}
}

var commitFinder = NewBytesFinder([]byte("commit "))
var spaceNLSplit = NewAnySplit([]byte(" \n"))
var prRegex = regexp.MustCompile(`Merge pull request #(\d+) from ([^ ]+)`)
var keySplit = NewByteDelim([]byte{':'}, []byte{' '}, []byte{'\n'})

func (ent *LogEnt) Reset() {
	ent.commit = ""
	ent.subject = ""
	ent.attrs = make(map[string]string)
	ent.mess = nil
}
