package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	usersDir = "/dev/ear7h"
)

var user string
var root string

func init() {
	fmt.Println("os args: ", os.Args)
}

func main() {
	var subCmd string

	subCmd, user, root = parseArgs()

	switch subCmd {
	case "run":
		parent()
	case "child":
		child()
	}

	fmt.Println("sucess ", subCmd)
}

func parseArgs() (subCmd, user, root string) {
	arr := strings.Split(os.Args[1], ":")
	return arr[0], arr[1], arr[2]
}

func parent() {
	cmd := exec.Command("/proc/self/exe",
		append([]string{
			fmt.Sprintf("%s:%s:%s", "child", user, root)},
			os.Args[2:]...)...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID,
	}

	if err := cmd.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func pivotRoot(newRoot string) error {
	putOld := filepath.Join(newRoot, "/.pivot_root")

	// bind mount newroot to itself
	// this is a work around for a pivot_root requirement
	err := syscall.Mount(
		newRoot,
		newRoot,
		"",
		syscall.MS_BIND|syscall.MS_REC,
		"")

	if err != nil {
		return err
	}

	err = os.MkdirAll(putOld, 0700)
	if err != nil {
		return err
	}

	err = syscall.PivotRoot(newRoot, putOld)
	if err != nil {
		return err
	}

	err = os.Chdir("/")
	if err != nil {
		return err
	}

	putOld = "/.pivot_root"
	err = syscall.Unmount(putOld, syscall.MNT_DETACH)
	if err != nil {
		return err
	}

	err = os.RemoveAll(putOld)
	if err != nil {
		return err
	}

	return nil
}

func child() {
	var err error

	fmt.Println("root dir: ", root)

	var dst string
	//bind the bin dir
	// dst = filepath.Join(root, "bin/")
	// os.MkdirAll(dst, 0755)
	// syscall.Mount("/bin/", dst, "",
	// 	syscall.MS_BIND|syscall.MS_REC, "")
	// exit(err)

	// mount user dir with overlayfs
	dst = filepath.Join(usersDir, user)

	//make dirs just in case
	for _, v := range []string{"home", "mount/proc", "work", "mount"} {
		os.MkdirAll(filepath.Join(dst, v), 0755)
	}

	// has a valid upperdir filesystem
	cmdStr := fmt.Sprintf("-t overlay overlay -olowerdir=%s,upperdir=%s,workdir=%s %s",
		filepath.Join(root),
		filepath.Join(dst, "home"),
		filepath.Join(dst, "work"),
		filepath.Join(dst, "mount"))

	err = exec.Command("mount", strings.Split(cmdStr, " ")...).Run()
	exit(err)

	// mount proc dir
	procDir := filepath.Join(dst, "mount", "proc")
	// procDir := filepath.Join(root, "proc")
	os.Mkdir(dst, 0755)
	err = syscall.Mount("/proc", procDir, "proc", 0, "")
	exit(err)

	// fmt.Println("proc target", procDir)

	// pivot root
	err = pivotRoot(filepath.Join(dst, "mount"))
	// err = pivotRoot(filepath.Join(root))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = syscall.Sethostname([]byte("huhhhh"))
	exit(err)

	arr, _ := filepath.Glob("/*/**")
	fmt.Println(arr)

	// fd, err := os.OpenFile(os.Args[2], os.O_RDONLY, 0755)
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	b := make([]byte, 10)
	// 	_, err := fd.Read(b)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 	} else {
	// 		os.Stdout.Write(b)
	// 		os.Stdout.Write([]byte{'\n'})
	// 	}
	// }

	var args []string
	if len(os.Args) > 3 {
		args = os.Args[3:]
	}
	fmt.Println([]byte(os.Args[2]), args)
	fmt.Printf("%q\n", os.Args[2])
	cmd := exec.Command(os.Args[2], args...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// fmt.Println(*cmd)

	if err := cmd.Run(); err != nil {
		fmt.Println("exec:", err)
		os.Exit(1)
	}
}

func exit(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
