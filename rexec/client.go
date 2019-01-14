package rexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
)

type Cmd struct {
	SrvAddr string
	SrvPath string
	Args    []string
	Dir     string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

func Command(addr, user, command string, args []string) (*Cmd, error) {
	fmt.Println("new cmd:", addr, user, command, args)
	return &Cmd{
		SrvAddr: addr,
		SrvPath: path.Join(user, command),
		Args:    args,
		Stdin:   bytes.NewReader([]byte{}),
	}, nil
}

// var unixClient = &http.Client{
// 	Transport: &http.Transport{
// 		Proxy: http.ProxyFromEnvironment,
// 		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
// 			fmt.Println(addr[:len(addr)-3])
// 			return (&net.Dialer{
// 				Timeout:   30 * time.Second,
// 				KeepAlive: 30 * time.Second,
// 				DualStack: true,
// 			}).DialContext(ctx, "unix", addr[:len(addr)-3])
// 		},
// 		MaxIdleConns:          100,
// 		IdleConnTimeout:       90 * time.Second,
// 		TLSHandshakeTimeout:   10 * time.Second,
// 		ExpectContinueTimeout: 1 * time.Second,
// 	},
// }

func (cmd *Cmd) Run() error {
	// add args to query
	q := url.Values{}
	for _, v := range cmd.Args {
		q.Add("arg", v)
	}
	cmd.SrvPath += q.Encode()
	fmt.Println("running: ", cmd.SrvAddr, cmd.SrvPath)

	clt := http.DefaultClient
	// if strings.HasPrefix(cmd.SrvAddr, "unix://") {
	if true {
		cmd.SrvAddr = strings.Replace(cmd.SrvAddr, "unix", "http", 1)
		clt = &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					fmt.Println("addr: ", addr)
					fmt.Println("srvaddr: ", cmd.SrvAddr)

					//fmt.Println("addr: ", u.Path)
					return net.Dial("unix", cmd.SrvAddr)
				},
			},
		}
	}

	req, err := http.NewRequest(http.MethodPost, "http://unix.sock/"+cmd.SrvPath, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Workdir", cmd.Dir)

	res, err := clt.Do(req)
	if err != nil {
		return err
	}

	byt, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(string(byt))
	}

	fmt.Println("RES: ", string(byt))

	r3pid := string(byt)

	defer http.NewRequest(http.MethodDelete, r3pid, nil)

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func(outPipe string) {
		res, err := clt.Get(outPipe)
		if err != nil {
			fmt.Println(err)
		} else {
			io.Copy(cmd.Stdout, res.Body)
			res.Body.Close()
		}

		wg.Done()
		fmt.Println("done")
	}("http://unix.sock/" + r3pid + ".out.pipe")

	go func(errPipe string) {
		res, err := clt.Get(errPipe)
		if err != nil {
			fmt.Println(err)
		} else {
			io.Copy(cmd.Stderr, res.Body)
			res.Body.Close()
		}

		wg.Done()
		fmt.Println("done")
	}("http://unix.sock/" + r3pid + ".err.pipe")

	go func(inPipe string) {
		defer wg.Done()
		req, err := http.NewRequest(http.MethodPut, inPipe, cmd.Stdin)
		if err != nil {
			fmt.Println(err)
			return
		}

		res, err := clt.Do(req)
		if err != nil {
			fmt.Println(err)
		} else {
			res.Body.Close()
		}

		fmt.Println("done")

	}("http://unix.sock/" + r3pid + ".in.pipe")

	wg.Wait()
	return nil
}
