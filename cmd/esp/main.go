package main

import (
	"fmt"
	"os"

	"github.com/ear7h/r3stexec/esp"
)

const usage = `usage:
esp listen address
esp exec address command args`

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "incorrect number of args %d, needs at least 3", len(os.Args))
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	addr := os.Args[2]
	switch os.Args[1] {
	case "listen":
		listen(addr)
	case "exec":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "incorrect number of args %d, exec needs at least 4", len(os.Args))
			fmt.Fprintln(os.Stderr, usage)
			os.Exit(1)
		}
		exec(addr, os.Args[3], os.Args[4:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %s", os.Args[1])
	}
}

func listen(addr string) {
	os.Remove(addr)
	l, err := esp.Listen(addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	for conn, err := l.AcceptSlave(); err == nil; conn, err = l.AcceptSlave() {
		err = conn.HandleExec()
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func exec(addr, cmd string, args []string) {
	proc, err := esp.Exec(addr, cmd, args...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	proc.Stdin = os.Stdin
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr

	i, err := proc.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	os.Exit(i)
}
