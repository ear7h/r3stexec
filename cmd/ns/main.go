package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ear7h/r3stexec/ns"
)

var user string
var root string

func init() {
	fmt.Println("os args: ", os.Args)
}

func main() {
	var subCmd string

	var args []string
	if len(os.Args) > 3 {
		args = os.Args[3:]
	}
	cmdStr := os.Args[2]
	fmt.Println([]byte(os.Args[2]), args)
	fmt.Printf("%q\n", os.Args[2])

	subCmd, user, root := parseArgs()

	switch subCmd {
	case "run":
		ns.Parent(user, root, cmdStr, args)
	case "child":
		ns.Child(user, root, cmdStr, args)
	}

	fmt.Println("sucess ", subCmd)
}

func parseArgs() (subCmd, user, root string) {
	arr := strings.Split(os.Args[1], ":")
	return arr[0], arr[1], arr[2]
}
