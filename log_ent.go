package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
)

// LogEnt represents data extracted from a git log entry.
type LogEnt struct {
	commit  string
	subject string
	attrs   map[string]string
	mess    []byte
}

// LogEntScanner scans log entries.
type LogEntScanner struct {
	scanner *bufio.Scanner
	err     error
	ent     LogEnt
	next    [][]byte
}

// NewLogEntScanner creates a new log entry scanner around an io.Reader.
func NewLogEntScanner(r io.Reader) *LogEntScanner {
	return &LogEntScanner{
		scanner: bufio.NewScanner(r),
	}
}

// Ent returns teh last scanned entry.
func (es *LogEntScanner) Ent() LogEnt {
	return es.ent
}

// Err returns any scanning error.
func (es *LogEntScanner) Err() error {
	if es.err != nil {
		return es.err
	}
	return es.scanner.Err()
}

var (
	commitPattern = regexp.MustCompile(`commit ([0-9a-fA-F]+)`)
	keyValPattern = regexp.MustCompile(`(.+?):\s+(.+)`)
	prMergeRegex  = regexp.MustCompile(`Merge .*#(\d+) from ([^ ]+)`)
	prSquashRegex = regexp.MustCompile(`(.+) +\(#(\d+)\)$`)
)

// Scan try to scan a log entry, returning true if another log entry may be
// scanned. When false is returned, either an error is set or EOF has been hit.
func (es *LogEntScanner) Scan() bool {
	if es.err != nil {
		return false
	}

	es.ent = LogEnt{
		attrs: make(map[string]string),
	}

	if es.next == nil {
		// scan "commit SHA.*" line
		for es.scanner.Scan() {
			if m := commitPattern.FindSubmatch(es.scanner.Bytes()); m != nil {
				es.next = m
				break
			}
		}
		if es.next == nil {
			return false
		}
	}
	es.ent.commit = string(es.next[1])

	return es.scanKeyVals()
}

func (es *LogEntScanner) scanKeyVals() bool {
	// scan all "key: val..." contiguous lines
	for es.scanner.Scan() {
		if len(es.scanner.Bytes()) == 0 {
			return es.scanSubject()
		}
		if m := keyValPattern.FindSubmatch(es.scanner.Bytes()); m != nil {
			es.ent.attrs[string(m[1])] = string(m[2])
			continue
		}
		es.err = fmt.Errorf("expected `key: val` line, instead of %q", es.scanner.Bytes())
		break
	}
	return false
}

func (es *LogEntScanner) scanSubject() bool {
	var buf bytes.Buffer
	for {
		// scan a message paragraph, normalizing newlines
		buf.Reset()
		for es.scanner.Scan() {
			b := es.scanner.Bytes()
			b = bytes.TrimLeft(b, " ")
			if len(b) == 0 {
				break
			}
			if buf.Len() > 0 {
				buf.Write([]byte{' '})
			}
			buf.Write(b)
		}
		if buf.Len() == 0 {
			return false
		}

		// extract an PR annotations, and then scan another paragraph for the subject
		if m := prMergeRegex.FindSubmatch(buf.Bytes()); m != nil {
			es.ent.attrs["prNumber"] = string(m[1])
			es.ent.attrs["prFrom"] = string(m[2])
			continue
		}

		if m := prSquashRegex.FindSubmatch(buf.Bytes()); m != nil {
			es.ent.attrs["prNumber"] = string(m[2])
			es.ent.subject = string(m[1])
			return es.scanMessage()
		}

		// otherwise we've found the subject, and we can move on to the rest of the message
		es.ent.subject = string(buf.String())
		return es.scanMessage()
	}
}

func (es *LogEntScanner) scanMessage() bool {
	// scan until next "commit SHA.*" line, collecting message bytes
	var buf bytes.Buffer
	for es.scanner.Scan() {
		if m := commitPattern.FindSubmatch(es.scanner.Bytes()); m != nil {
			es.ent.mess = buf.Bytes()
			es.next = m
			return true
		}
		buf.Write(es.scanner.Bytes())
		buf.Write([]byte{'\n'})
	}
	return false
}
