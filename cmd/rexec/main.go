package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/ear7h/r3stexec/rexec"
)

var addr string
var workdir string
var host, user string

func main() {

	flag.StringVar(&addr, "a", "tcp://:8080", "address")
	flag.StringVar(&user, "u", "plebe", "user")
	flag.StringVar(&workdir, "wd", ".", "working directory")
	flag.Parse()

	switch flag.Args()[0] {
	case "serve":
		serve()
	case "client":
		client()
	default:
		fmt.Println("invalid subcommand ", flag.Args()[0])
	}
}

func serve() {
	fmt.Println("server")
	u, err := url.Parse(addr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	l, err := net.Listen(u.Scheme, u.Host)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	pipeDir := path.Join(os.Getenv("HOME"), "rexecsocks")
	err = os.MkdirAll(pipeDir, 0700)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	http.Serve(l, &rexec.Server{
		PipeDir: pipeDir,
	})
}

func client() {
	fmt.Println(flag.Args())
	cmd, err := rexec.Command(addr, user,
		flag.Args()[1], flag.Args()[2:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cmd.Dir = workdir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
