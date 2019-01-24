package esp

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// th
func readInitMsg(c *SlaveConn) (cmd string, args []string, err error) {
	buf := make([]byte, 8)
	i := int64(0)

	for i < 4 && err == nil {
		var n int
		n, err = c.Read(buf[i:4])
		i += int64(n)
	}
	if err != nil {
		return "", nil, err
	}

	argc := int(binary.LittleEndian.Uint32(buf[:4]))
	fmt.Println("argc: ", argc)
	args = make([]string, argc+1)
	arg := &strings.Builder{}

	for argi := 0; argi <= argc; argi++ {
		fmt.Println(args)
		i = 0
		for i < 8 && err == nil {
			var n int
			n, err = c.Read(buf[i:8])
			i += int64(n)
		}
		if err != nil {
			return "", nil, err
		}

		argl := int64(binary.LittleEndian.Uint64(buf))
		fmt.Printf("argl(%d) : %d\n", argi, argl)
		i = 0
		arg.Reset()
		_, err := io.CopyN(arg, c, argl)
		fmt.Println("done reading: ", err)
		if err != nil {
			return "", nil, err
		}

		args[argi] = arg.String()
		fmt.Println(args)
	}

	return args[0], args[1:], nil
}

// The init msg is encoded as a length-prefixed (sort of) linked list.
// The first 4 bytes are little endian encoded int, represents
// the number of arguments argc. The following bytes are argc+1
// number of terminated strings prefixed with 8 byte ints. The
// first string is the command to run
func writeInitMsg(c *MasterConn, cmd string, args ...string) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf[:4], uint32(len(args)))

	var err error
	i := int64(0)
	for i < 4 && err == nil {
		var n int
		n, err = c.Write(buf[i:4])
		i += int64(n)
	}
	if err != nil {
		return err
	}

	for _, v := range append([]string{cmd}, args...) {
		// send length
		binary.LittleEndian.PutUint64(buf, uint64(len(v)))
		i = 0
		for i < 8 && err == nil {
			var n int
			n, err = c.Write(buf[i:8]) // 8 not really needed
			i += int64(n)
		}
		// send string
		i = 0
		for i < int64(len(v)) && err == nil {
			fmt.Println("sending: ", []byte(v)[i:])
			var n int
			n, err = c.Write([]byte(v)[i:])
			i += int64(n)
		}
		if err != nil {
			return err
		}
	}

	return nil
}
