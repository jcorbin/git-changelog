package main

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"
)

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

func (ent *LogEnt) Scan(scanner *bufio.Scanner) (bool, error) {
	for _, f := range []func(*bufio.Scanner) (bool, error){
		ent.scanCommit,
		ent.scanAttrs,
		ent.scanSubject,
		ent.scanMess,
	} {
		if ok, err := f(scanner); !ok || err != nil {
			return ok, err
		}
	}

	return true, nil
}

func (ent *LogEnt) scanCommit(scanner *bufio.Scanner) (bool, error) {
	// scan thru "commit "
	scanner.Split(commitFinder.SplitJust)
	if !scanner.Scan() {
		return false, scanner.Err()
	}

	// scan thru space or newline
	scanner.Split(spaceNLSplit.Split)
	if !scanner.Scan() {
		return false, scanner.Err()
	}
	ent.commit = scanner.Text()

	// consume the rest of the line if necessary
	if c, _ := spaceNLSplit.Last(); c != '\n' {
		scanner.Split(bufio.ScanLines)
		if !scanner.Scan() {
			return false, scanner.Err()
		}
	}

	return true, nil
}

func (ent *LogEnt) scanAttrs(scanner *bufio.Scanner) (bool, error) {
	// scan all "key: val..." contiguous lines
	for {
		// scan "key: "
		scanner.Split(keySplit.Split)
		if !scanner.Scan() {
			return false, scanner.Err()
		}
		keyBytes := scanner.Bytes()
		if len(keyBytes) == 0 {
			// empty key means keySplit hit a newline before a :
			break
		}

		// scan value until newline
		scanner.Split(bufio.ScanLines)
		if !scanner.Scan() {
			return false, scanner.Err()
		}

		ent.attrs[string(keyBytes)] = scanner.Text()
	}
	return true, nil
}

func (ent *LogEnt) scanSubject(scanner *bufio.Scanner) (bool, error) {
	// scan subject line
	scanner.Split(bufio.ScanLines)
	str, err := scanMessagePart(scanner, " ")
	if err != nil {
		return false, err
	}

	// if we have a PR merge, extract more annotations, and promote the next
	// message part
	if prMatch := prRegex.FindStringSubmatch(str); prMatch != nil {
		ent.attrs["prNumber"] = prMatch[1]
		ent.attrs["prFrom"] = prMatch[2]

		// TODO: would be nice to not accidentally skip any "^commit "
		if next, err := scanMessagePart(scanner, " "); err != nil {
			return false, err
		} else {
			str = next
		}
	}

	ent.subject = str
	return true, nil
}

func (ent *LogEnt) scanMess(scanner *bufio.Scanner) (bool, error) {
	// scan message until next "commit "
	scanner.Split(commitFinder.SplitUntil)
	if !scanner.Scan() {
		return false, scanner.Err()
	}

	// scan message paragraphs
	messScanner := bufio.NewScanner(bytes.NewBuffer(scanner.Bytes()))
	for {
		if str, err := scanMessagePart(messScanner, "\n"); err != nil {
			return true, err
		} else if len(str) == 0 {
			return true, nil
		} else {
			ent.mess = append(ent.mess, str)
		}
	}
}

func scanMessagePart(scanner *bufio.Scanner, sep string) (string, error) {
	var parts []string
	for scanner.Scan() {
		// TODO: consistent de-indent by first-line detection
		line := strings.TrimLeft(scanner.Text(), " ")
		if len(line) == 0 {
			break
		}
		parts = append(parts, line)
	}
	part := strings.Join(parts, sep)
	return part, scanner.Err()
}
