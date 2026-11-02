package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

const usage = "usage: loop-review post <clone> <number> <draft.json>"

func run(args []string) error {
	if len(args) != 4 || args[0] != "post" {
		return errors.New(usage)
	}
	clone, numberArg, draftPath := args[1], args[2], args[3]
	number, err := strconv.Atoi(numberArg)
	if err != nil || number < 1 {
		return fmt.Errorf("pull request number %q is not a positive integer\n%s", numberArg, usage)
	}
	return post(ghCLI{}, clone, number, draftPath, os.Stdout)
}
