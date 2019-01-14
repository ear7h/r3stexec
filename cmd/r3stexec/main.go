package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var sockets = sync.Map{}

func main() {
	fmt.Println("hello")
	http.ListenAndServe(":8080", &server{})
}

type server struct{}

var unixClient = &http.Client{
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			fname := filepath.Join("/home", addr[:len(addr)-3], "rpc.sock")
			fmt.Println("addr: ", fname)
			return net.Dial("unix", fname)
		},
	},
}

func (srv *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL)

	arr := strings.Split(r.URL.Path, "/")[1:]
	fmt.Println(arr)
	if len(arr) <= 2 {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}

	v, ok := sockets.Load(arr[0])
	if !ok {
		mng := &socketManager{
			user: arr[0],
			errc: make(chan error),
		}
		go mng.start()

		err := mng.err()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sockets.Store(arr[0], mng)
	} else {
		v.(*socketManager).keepLive()
	}

	fmt.Println("forwarding")

	r.URL.Host = arr[0]
	r.URL.Scheme = "http"

	fmt.Printf("%v\n", r.URL)

	res, err := unixClient.Get(r.URL.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	w.WriteHeader(http.StatusOK)

	fmt.Println(io.Copy(w, res.Body))
}

func homeDir(user string) string {
<<<<<<< Updated upstream
	return fmt.Sprintf("/dev/ear7h/%s/root/home/%s", user, user)
=======
	return fmt.Sprintf("/dev/ear7h/%s/root/home/%s/", user, user)
>>>>>>> Stashed changes
}

type socketManager struct {
	user     string
	timeout  time.Duration
	liveChan chan struct{}
	errc     chan error
}

const root = "/go/src/github.com/ear7h/r3stexec/stretch"

func (mng *socketManager) start() {
	defer sockets.Delete(mng.user)

	//mng.errc = make(chan error)
	mng.liveChan = make(chan struct{})
	if mng.timeout == time.Duration(0) {
		mng.timeout = 60 * time.Second
	}

	cmd := exec.Command("/bin/ns", "run:"+mng.user+":"+root,
		"rexec", "-a", "unix://rpc.sock", "serve")
	// if err != nil {
	// 	mng.errc <- err
	// 	return
	// }

	cmd.Dir = fmt.Sprintf("/dev/ear7h/%s/root/home/%s", mng.user, mng.user)

	fmt.Println("running cmd: ", *cmd)

	err := cmd.Start()
	fmt.Println("sending error", err)
	if err != nil {
		mng.errc <- err
		fmt.Println("returning")
		return
	} else {
		mng.errc <- nil
	}
	fmt.Println("error sent", err)

	for {
		select {
		case <-time.After(mng.timeout):
			break
		case <-mng.liveChan:
		}
	}
	fmt.Println("dying")
}

func (mng *socketManager) err() error {
	fmt.Println("waiting for error")
	err := <-mng.errc
	fmt.Println("error recieved", err)
	return err
}

func (mng *socketManager) keepLive() {
	mng.liveChan <- struct{}{}
}
