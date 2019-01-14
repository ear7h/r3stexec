package rexec

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

type Server struct {
	PipeDir string
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	fmt.Println("new request", *r)

	if r.Method != http.MethodPost {
		arr := strings.Split(r.URL.Path, "/")
		if len(arr) < 1 {
			http.Error(w, "bad url", http.StatusBadRequest)
			return
		}

		r3pid := arr[1]

		if r.Method == http.MethodDelete {
			i, err := strconv.Atoi(r3pid)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			err = syscall.Kill(procs[i], syscall.SIGKILL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
			return
		}

		//outPipe := filepath.Join(srv.pipeDir, pidStr+".out.pipe")

		if r.Method == http.MethodGet {
			fd, err := os.OpenFile(filepath.Join(srv.PipeDir, r3pid), os.O_RDWR, 0600)
			fmt.Println("fopen", r3pid)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			io.Copy(w, fd)

			return
		}

		if r.Method == http.MethodPut {
			fd, err := os.OpenFile(filepath.Join(srv.PipeDir, r3pid), os.O_RDWR, 0600)
			fmt.Println("fopen", r3pid)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			io.Copy(fd, r.Body)
			r.Body.Close()

			return
		}

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	paths := strings.Split(r.URL.Path, "/")
	if len(paths) < 3 { // ["", "user", "cmd"]
		fmt.Println(paths)
		http.Error(w, "path not valid", http.StatusBadRequest)
		return
	}

	cmdUser, err := user.Lookup(paths[1])
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	cmdPath := filepath.Join(append([]string{"/"}, paths[2:]...)...)
	cmdArgs := r.URL.Query()["arg"]
	// if err != nil {
	// 	http.Error(w, "query not valid "+r.URL.RawQuery, http.StatusBadRequest)
	// 	return
	// }

	authUser, ok := auth(r.Header.Get("Authorization"))
	var mode os.FileMode
	if !ok {
		// todo: if auth fail it should be not authorized
		mode = 0005
	} else if cmdUser.Name != authUser {
		usr, err := user.Lookup(authUser)
		if err != nil {
			//should not happen
			http.Error(w, "could not authenticate", http.StatusInternalServerError)
			return
		}

		arr, err := usr.GroupIds()
		if err != nil {
			//should not happen
			http.Error(w, "could not authenticate", http.StatusInternalServerError)
			return
		}

		// todo: optimize this
		// look up groups
		for _, v := range arr {
			if v == cmdUser.Uid {
				mode = 0050
				goto L // we gucci
			}
		}

		http.Error(w, "not authorized", http.StatusForbidden)
		return
	} else {
		mode = 0500
	}

L:

	fi, err := os.Stat(cmdPath)
	if err == os.ErrNotExist {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	} else if err != nil {
		fmt.Println(err)
		http.Error(w, "could not stat file", http.StatusBadRequest)
		return
	}

	if fi.Mode()&mode == 0 {
		http.Error(w, "not authorized", http.StatusForbidden)
		return
	}

	r3pid, ok := getProc()
	if !ok {
		http.Error(w, "busy", http.StatusServiceUnavailable)
		return
	}

	pidStr := strconv.Itoa(r3pid)

	fmt.Println("opening files", pidStr)

	outPipe := filepath.Join(srv.PipeDir, pidStr+".out.pipe")
	fmt.Println("mkfifo: ", outPipe)
	err = syscall.Mkfifo(outPipe, 0600)
	if err != nil {
		os.Remove(outPipe)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	errPipe := filepath.Join(srv.PipeDir, pidStr+".err.pipe")
	err = syscall.Mkfifo(errPipe, 0600)
	if err != nil {
		os.Remove(outPipe)
		os.Remove(errPipe)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	inPipe := filepath.Join(srv.PipeDir, pidStr+".in.pipe")
	err = syscall.Mkfifo(inPipe, 0600)
	if err != nil {
		os.Remove(outPipe)
		os.Remove(errPipe)
		os.Remove(inPipe)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println(filepath.Glob(srv.PipeDir + "/*"))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(pidStr))

	go func() {
		var outF, errF, inF *os.File

		fmt.Println("openfile: ", outPipe)
		outF, err = os.OpenFile(outPipe, os.O_RDWR, 0600)
		if err != nil {
			os.Remove(outPipe)
			os.Remove(errPipe)
			os.Remove(inPipe)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Println("openfile: ", errPipe)
		errF, err = os.OpenFile(errPipe, os.O_RDWR, 0600)
		if err != nil {
			os.Remove(outPipe)
			os.Remove(errPipe)
			os.Remove(inPipe)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Println("openfile: ", inPipe)
		inF, err = os.OpenFile(inPipe, os.O_RDWR, 0600)
		if err != nil {
			os.Remove(outPipe)
			os.Remove(errPipe)
			os.Remove(inPipe)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Println("files created")

		// todo: add trailers
		cmd := exec.Command(cmdPath, cmdArgs...)
		cmd.Dir = r.Header.Get("Workdir")
		cmd.Stdin = inF
		cmd.Stdout = outF
		cmd.Stderr = errF

		fmt.Println("starting")

		err = cmd.Start()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		procs[r3pid] = cmd.Process.Pid

		fmt.Println("PROC: ", cmd.Process.Pid)
	}()

}

const maxProcs = 100

// mapping of r3stos procs to linux procs
var procs = map[int]int{}
var procNum = 1
var lock = sync.Mutex{}

func getProc() (int, bool) {
	lock.Lock()

	if len(procs) >= maxProcs {
		return -1, false
	}

	_, ok := procs[procNum]
	for ok {
		procNum++
		_, ok = procs[procNum]
	}

	ret := procNum
	lock.Unlock()
	return ret, true
}

func returnProc(pn int) {
	lock.Lock()
	delete(procs, pn)
	lock.Unlock()
}

func auth(str string) (string, bool) {
	arr := strings.Split(str, " ")
	if len(arr) != 2 {
		return "", false
	}

	switch arr[0] {
	case "Basic":
		arr = strings.SplitN(str, ":", 2)
		if len(arr) != 2 {
			return "", false
		}
		//TODO: change this
		return arr[0], true

	case "Token":
		fallthrough

	default:
		return "", false
	}
}

func parseArgs(qs string) ([]string, error) {
	var err error
	arr := strings.Split(qs, "+")
	fmt.Println(arr)
	for k, v := range arr {
		arr[k], err = url.QueryUnescape(v)
		if err != nil {
			return nil, err
		}
	}

	return arr, nil
}
