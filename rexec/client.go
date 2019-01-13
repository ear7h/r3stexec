package rexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"
)

type Cmd struct {
	URL    *url.URL
	Args   []string
	Dir    string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func Command(host, user, command string, args []string) (*Cmd, error) {
	// unix:///rpc.sock/user/bin/ls?arg=-la&arg=/home
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	u.Path = path.Join(u.Path, user, command)

	return &Cmd{
		URL:   u,
		Args:  args,
		Stdin: bytes.NewReader([]byte{}),
	}, nil
}

var unixClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			fmt.Println(addr[:len(addr)-3])
			return (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext(ctx, "unix", addr[:len(addr)-3])
		},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

func (cmd *Cmd) Run() error {
	// add args to query
	q := cmd.URL.Query()
	for _, v := range cmd.Args {
		q.Add("arg", v)
	}
	cmd.URL.RawQuery = q.Encode()
	fmt.Println("running: ", cmd.URL)

	clt := http.DefaultClient
	if cmd.URL.Scheme == "unix" {
		cmd.URL.Scheme = "http"
		clt = unixClient
	}

	req, err := http.NewRequest("GET", cmd.URL.String(), cmd.Stdin)
	if err != nil {
		return err
	}

	req.Header.Add("Workdir", cmd.Dir)

	res, err := clt.Do(req)
	if err != nil {
		return err
	}

	_, err = io.Copy(cmd.Stdout, res.Body)
	if err != nil {
		return err
	}

	return res.Body.Close()
}
