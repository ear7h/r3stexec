package main

import (
	"fmt"
	"os"

	"github.com/ear7h/r3stexec/ns"
)

const usage = `ns run user rootDir command [args ...]`

func init() {
	fmt.Println("os args: ", os.Args)
}

func main() {

	if len(os.Args) < 5 {
		fmt.Fprintf(os.Stderr, "incorrect number of args %d, needs at least 5", len(os.Args))
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	subCmd, user, rootDir,
		spawnCmd, args :=
		os.Args[1], os.Args[2], os.Args[3],
		os.Args[4], os.Args[5:]

	fmt.Println([]byte(os.Args[2]), args)
	fmt.Printf("%q\n", os.Args[2])

	switch subCmd {
	case "run":
		ns.Parent(user, rootDir, spawnCmd, args)
	case "child":
		ns.Child(user, rootDir, spawnCmd, args)
	default:
		fmt.Println("unknon command ".subCmd)
		os.Exit(1)
	}

	fmt.Println("sucess ", subCmd)
}
