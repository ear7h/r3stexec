package esp

import (
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
)

const (
	StdinClose = -1
)

type Process struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	// ipc pipes
	stdin  *os.File
	stdout *os.File
	stderr *os.File

	// running program
	done     chan error
	exitCode int
	once     sync.Once

	conn *MasterConn
}

func (proc *Process) Run() (int, error) {
	proc.Start()
	return proc.Wait()
}

func (proc *Process) Start() error {
	fmt.Println("starting")
	proc.done = make(chan error)

	go func() {
		var err error
		for proc.done != nil {
			if proc.Stdin != nil {
				_, err = io.Copy(proc.stdin, proc.Stdin)
				if err != nil {
					fmt.Println("stdin: ", err)
					break
				}
			}
		}

		if err != nil {
			fmt.Fprintln(os.Stdout, err.Error())
			proc.once.Do(func() {
				proc.done <- err
			})
		}
		proc.EOF()
	}()

	go func() {
		var err error
		for proc.done != nil {
			_, err = io.Copy(proc.Stdout, proc.stdout)
			fmt.Println("stdout", err)
			if err != nil {
				fmt.Println("break")
				break
			}
		}

		if err != nil {
			fmt.Fprintln(os.Stdout, err.Error())
			proc.once.Do(func() {
				proc.done <- err
			})
		}
	}()

	go func() {
		var err error
		for proc.done != nil {
			_, err = io.Copy(proc.Stderr, proc.stderr)
			if err != nil {
				break
			}
		}

		if err != nil {
			fmt.Fprintln(os.Stdout, err.Error())
			proc.once.Do(func() {
				proc.done <- err
			})
		}
	}()

	go func() {
		i, err := readInt32(proc.conn)

		proc.once.Do(func() {
			fmt.Println("read int")
			proc.exitCode = int(i)
			proc.done <- err
		})
	}()

	return nil
}

func (proc *Process) Wait() (int, error) {
	fmt.Println("waiting")

	err := <-proc.done
	close(proc.done)
	proc.done = nil

	fmt.Println("done")

	var err1 error
	if proc.stdin != nil {
		err1 = proc.stdin.Close()
		if err != nil {
			return 0, err1
		}
	}

	err1 = proc.stdout.Close()
	if err != nil {
		return 0, err1
	}

	err1 = proc.stderr.Close()
	if err != nil {
		return 0, err1
	}

	return proc.exitCode, err
}

func (proc *Process) Kill() error {
	return proc.conn.Close()
}

func (proc *Process) Signal(sig syscall.Signal) error {
	return writeInt32(proc.conn, int32(sig))
}

// EOF sends eof to the slaves standard in. This is necessary bc calling close on the master's fd does not effect the slave's fd.
func (proc *Process) EOF() error {
	proc.stdin.Close()
	proc.stdin = nil
	fmt.Println("sending eof to stdin")
	return writeInt32(proc.conn, StdinClose)
}
