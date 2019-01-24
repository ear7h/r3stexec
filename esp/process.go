package esp

import (
	"os"
	"syscall"
)

const (
	StdinClose = -1
)

type Process struct {
	Stdin  *os.File
	Stdout *os.File
	Stderr *os.File

	conn *MasterConn
}

func (proc *Process) Kill() error {
	return proc.conn.Close()
}

func (proc *Process) Signal(sig syscall.Signal) error {
	return writeInt32(proc.conn, int32(sig))
}

func (proc *Process) Wait() (int, error) {
	i, err := readInt32(proc.conn)
	return int(i), err
}

// EOF sends eof to the slaves standard in. This is necessary bc calling close on the master's fd does not effect the slave's fd.
func (proc *Process) EOF() error {
	return writeInt32(proc.conn, StdinClose)
}
