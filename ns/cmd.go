package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	var cmd *exec.Cmd
	var err error

	fmt.Println(os.Args)

	switch os.Args[1] {
	case "run":
		cmd = parent()
	case "child":
		cmd, err = child()
		if err != nil {
			panic(err)
		}
	default:
				panic("subcommand not recognized")
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err = cmd.Run(); err != nil {
		panic(err)
	}
}

func Cmd(name string, args []string) *exec.Cmd {
	cmd := exec.Command(name, append([]string{"child"}, os.Args[2:]...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
	}
	return cmd
}

func child() (*exec.Cmd, error) {

	err := syscall.Mount("rootfs", "rootfs", "", syscall.MS_BIND, "")
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll("rootfs/oldrootfs", 0700)
	if err != nil {
		return nil, err
	}

	err = syscall.PivotRoot("rootfs", "rootfs/oldrootfs")
	if err != nil {
		return nil, err
	}

	err = os.Chdir("/")
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(os.Args[2], os.Args[3:]...)

	return cmd, nil
}
