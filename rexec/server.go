package rexec

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

type Server struct {
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	fmt.Println("new request")

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

	w.WriteHeader(http.StatusOK)

	// todo: add trailers
	cmd := exec.Command(cmdPath, cmdArgs...)
	cmd.Dir = r.Header.Get("Workdir")
	cmd.Stdin = r.Body
	defer r.Body.Close()
	cmd.Stdout = w
	cmd.Stderr = w

	cmd.Run()
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
