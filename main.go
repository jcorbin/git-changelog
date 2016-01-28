package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
)

var projectGithub = flag.String("github", "", "project github user/repo")

func main() {
	flag.Parse()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		// Scan a new log entry...
		ent := NewEnt()
		if ok, err := ent.Scan(scanner); err != nil {
			panic(err)
		} else if !ok {
			break
		}

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
}
