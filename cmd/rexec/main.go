package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/ear7h/r3stexec/rexec"
)

var addr string
var workdir string
var host, user string

func main() {

	flag.StringVar(&addr, "a", "tcp://:8080", "address")
	flag.StringVar(&user, "u", "pleb", "user")
	flag.StringVar(&host, "h", "tcp://:8080", "host")
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

	http.Serve(l, &rexec.Server{})
}

func client() {
	fmt.Println(flag.Args())
	cmd, err := rexec.Command(host, user,
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
