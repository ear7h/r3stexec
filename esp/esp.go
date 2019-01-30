package esp

/*
Execution by socket protocol
*/

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

func Listen(filename string) (*Slave, error) {
	addr, err := net.ResolveUnixAddr("unix", filename)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, err
	}

	return &Slave{*conn}, nil
}

type Slave struct {
	net.UnixListener
}

func (s *Slave) Accept() (net.Conn, error) {
	c, err := s.AcceptUnix()
	if err != nil {
		return nil, err
	}

	return &SlaveConn{*c}, nil
}

func (s *Slave) AcceptSlave() (*SlaveConn, error) {
	c, err := s.AcceptUnix()
	if err != nil {
		return nil, err
	}

	return &SlaveConn{*c}, nil
}

type SlaveConn struct {
	net.UnixConn
}

// closes connection
func (c *SlaveConn) HandleExec() error {
	defer func() {
		fmt.Println("closing conn")
		c.Close()
	}()

	cmdStr, args, err := readInitMsg(c)
	if err != nil {
		return err
	}

	fmt.Println("command: ", cmdStr, args)
	cmd := exec.Command(cmdStr, args...)
	inr, inw, err := os.Pipe()
	if err != nil {
		return err
	}
	defer inr.Close()
	defer inw.Close()
	outr, outw, err := os.Pipe()
	if err != nil {
		return err
	}
	defer outr.Close()
	defer outw.Close()
	errr, errw, err := os.Pipe()
	if err != nil {
		return err
	}
	defer errr.Close()
	defer errw.Close()

	cmd.Stdin = inr
	cmd.Stdout = outw
	cmd.Stderr = errw

	err = c.sendFiles(inw, outr, errr)
	if err != nil {
		return err
	}

	fmt.Println("sent files")

	err = cmd.Start()
	if err != nil {
		fmt.Println("error starting: ", err)
		return err
	}
	fmt.Println("started")

	var once sync.Once
	var exitCode int
	errc := make(chan error)

	go func() {
		err := cmd.Wait()
		fmt.Println("cmd done")

		once.Do(func() {
			errc <- err
		})
	}()

	go func() {
		var err error
		var n int32
		for {
			n, err = readInt32(c)
			if err != nil {
				break
			}

			if n < 0 {
				switch n {
				case StdinClose:
					fmt.Println("closing stdin")
					inr.Close()
				}
			} else {
				sig := syscall.Signal(n)
				cmd.Process.Signal(sig)
				fmt.Println("signal", sig)
			}
		}

		once.Do(func() {
			errc <- err
		})
	}()

	err = <-errc
	close(errc)

	if err == io.EOF {
		cmd.Process.Kill()
		err = fmt.Errorf("socket closed, proc killed")
	} else if err != nil {
		fmt.Println("error: ", err.Error())
		errw.Write([]byte(err.Error()))
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		}
		writeInt32(c, int32(exitCode))
	} else {
		writeInt32(c, 0)
	}

	return err
}

func readInt32(c net.Conn) (int32, error) {
	b := make([]byte, 4)
	i := 0
	var err error
	for i < 4 && err == nil {
		var n int
		n, err = c.Read(b)
		i += n
	}
	return int32(binary.LittleEndian.Uint32(b)), err
}

func writeInt32(c net.Conn, num int32) error {
	b := make([]byte, 4)
	i := 0
	var err error

	binary.LittleEndian.PutUint32(b, uint32(num))

	for i < 4 && err == nil {
		var n int
		n, err = c.Write(b)
		i += n
	}
	return err
}

// taken from:
// https://github.com/ftrvxmtrx/fd/blob/c6d800382fff6dc1412f34269f71b7f83bd059ad/fd.go
func (c *SlaveConn) sendFiles(files ...*os.File) error {

	if len(files) == 0 {
		return nil
	}

	// get the fd for the socket
	cf, err := c.File()
	if err != nil {
		return err
	}
	cfd := int(cf.Fd())
	defer cf.Close()

	fds := make([]int, len(files))
	for k, v := range files {
		fds[k] = int(v.Fd())
	}

	rights := syscall.UnixRights(fds...)
	return syscall.Sendmsg(cfd, nil, rights, nil, 0)

}

type MasterConn struct {
	net.UnixConn
}

func dial(filename string) (*MasterConn, error) {
	addr, err := net.ResolveUnixAddr("unix", filename)
	if err != nil {
		return nil, err
	}
	c, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return nil, err
	}

	return &MasterConn{*c}, nil
}

// taken from:
// https://github.com/ftrvxmtrx/fd/blob/c6d800382fff6dc1412f34269f71b7f83bd059ad/fd.go
func (l *MasterConn) getFiles(files ...string) ([]*os.File, error) {

	if len(files) == 0 {
		return []*os.File{}, nil
	}

	cf, err := l.File()
	if err != nil {
		return nil, err
	}
	cfd := int(cf.Fd())
	defer cf.Close()

	buf := make([]byte, syscall.CmsgSpace(len(files)*4))
	_, _, _, _, err = syscall.Recvmsg(cfd, nil, buf, 0)
	if err != nil {
		return nil, err
	}

	//var msgs []syscall.SocketControlMessage
	msgs, err := syscall.ParseSocketControlMessage(buf)
	if err != nil {
		return nil, err
	}

	fds := []int{}
	for i := 0; i < len(msgs) && err == nil; i++ {
		var nfds []int
		nfds, err = syscall.ParseUnixRights(&msgs[i])
		fds = append(fds, nfds...)
	}
	if err != nil {
		return nil, err
	}
	fmt.Println(fds)

	res := make([]*os.File, len(fds))
	for k, v := range fds {
		res[k] = os.NewFile(uintptr(v), files[k])
	}

	return res, nil
}

func Exec(addr, cmd string, args ...string) (*Process, error) {
	conn, err := dial(addr)
	if err != nil {
		return nil, err
	}

	err = writeInitMsg(conn, cmd, args...)
	if err != nil {
		return nil, err
	}

	files, err := conn.getFiles("stdin", "stdout", "stderr")
	if err != nil {
		return nil, err
	}

	return &Process{
		stdin:  files[0],
		stdout: files[1],
		stderr: files[2],
		conn:   conn,
	}, nil
}
