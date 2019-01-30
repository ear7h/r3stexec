package ns

import (
	"fmt"
	"os"
	"os/exec"
	userlib "os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

const (
	usersDir = "/dev/ear7h"
)

func Parent(user, root, cmdStr string, args []string) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cmd := exec.Command("/proc/self/exe",
		append([]string{"child", user, root, cmdStr},
			args...)...)

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

func Child(user, root, cmdStr string, args []string) {
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
	for _, v := range []string{"root", "root/home/" + user, "mount/etc", "mount/proc", "work", "mount"} {
		fmt.Println(filepath.Join(dst, v))
		err := os.MkdirAll(filepath.Join(dst, v), 0755)
		if err != nil && os.IsExist(err) {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	// has a valid upperdir filesystem
	overlayCmd := fmt.Sprintf("-t overlay overlay -olowerdir=%s,upperdir=%s,workdir=%s %s",
		filepath.Join(root),
		filepath.Join(dst, "root"),
		filepath.Join(dst, "work"),
		filepath.Join(dst, "mount"))

	err = exec.Command("mount", strings.Split(overlayCmd, " ")...).Run()
	exit(err)

	// mount proc dir
	procDir := filepath.Join(dst, "mount", "proc")
	// procDir := filepath.Join(root, "proc")
	os.MkdirAll(procDir, 0755)
	err = syscall.Mount("/proc", procDir, "proc", 0, "")
	exit(err)

	etcDir := filepath.Join(dst, "mount", "etc")
	// procDir := filepath.Join(root, "proc")
	os.MkdirAll(etcDir, 0755)
	err = syscall.Mount("/etc", etcDir, "", syscall.MS_BIND, "")
	exit(err)

	homeDir := filepath.Join(dst, "mount", "home")
	// procDir := filepath.Join(root, "proc")
	os.MkdirAll(homeDir, 0755)
	err = syscall.Mount("/home", homeDir, "", syscall.MS_BIND, "")
	exit(err)

	// fmt.Println("proc target", procDir)

	// pivot root
	err = pivotRoot(filepath.Join(dst, "mount"))
	// err = pivotRoot(filepath.Join(root))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = syscall.Sethostname([]byte("r3stos UwU "))
	exit(err)

	fmt.Println(os.Getwd())

	arr, _ := filepath.Glob("/home/*")
	fmt.Println(arr)
	arr, _ = filepath.Glob("/*")
	fmt.Println(arr)

	stat, err := os.Stat("/bin/env")
	fmt.Println(stat, err)

	cmd := exec.Command(cmdStr, args...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	userObj, err := userlib.Lookup(user)
	exit(err)

	uid, err := strconv.Atoi(userObj.Uid)
	exit(err)

	gid, err := strconv.Atoi(userObj.Gid)
	exit(err)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid:         uint32(uid),
			Gid:         uint32(gid),
			NoSetGroups: true,
		},
	}

	// cmd will fail with file does not exist if the hom
	// directory does not exist
	homeDir = filepath.Join("/home", user)
	cmd.Dir = homeDir
	cmd.Env = []string{"HOME=" + homeDir}

	fmt.Println("\n", *cmd)

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
