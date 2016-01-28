package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

var projectGithub = flag.String("github", "", "project github user/repo")

func main() {
	flag.Parse()

	sinceRef := "ORIG_HEAD"
	untilRef := "HEAD"

	args := flag.Args()

	if len(args) > 0 {
		sinceRef, args = args[0], args[1:]
	}

	if len(args) > 0 {
		untilRef, args = args[0], args[1:]
	}

	// TODO: simplify scanning by forcing such things as --no-decorate
	refspec := strings.Join([]string{sinceRef, untilRef}, "..")
	cmd := exec.Command("git", "log", "--first-parent", refspec)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	scanner := NewLogEntScanner(stdout)

	for scanner.Scan() {
		ent := scanner.Ent()

		// ...and print a changelog entry
		fmt.Printf("- %v", ent.subject)

		prNumber, ok := ent.attrs["prNumber"]
		if ok {
			if *projectGithub != "" {
				fmt.Printf(" PR: https://github.com/%v/pull/%v", *projectGithub, prNumber)
			} else {
				prFrom := ent.attrs["prFrom"]
				fmt.Printf(" PR: #%v from %v", prNumber, prFrom)
			}
		}
		fmt.Printf("\n")

		// author, ok := ent.attrs["Author"]
		// if ok {
		// 	fmt.Printf("  - Author: %v\n", author)
		// }

		// fmt.Printf("%#v\n", ent)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	if err := cmd.Wait(); err != nil {
		fmt.Printf("git-log %v\n", err)
		os.Exit(1)
	}
}
