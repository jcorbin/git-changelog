package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var projectGithub = flag.String("github", "", "project github user/repo")

func main() {
	flag.Parse()

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
}
