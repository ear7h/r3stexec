// go test .
package esp

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"
)

const (
	SockPath = "/tmp/esptest.sock"
	Secret   = "asdasdasdasdasdasdasdasdasdasdasd\n"
)

var mode = flag.String("mode", "", "[slave | master]")

func TestEsp(t *testing.T) {

	switch *mode {
	case "slave":
		testSlave(t)
		return
	case "master":
		testMaster(t)
		return
	case "":
		break
	default:
		t.Fatalf("mode %s unknown", *mode)
	}

	slave := exec.Command("go", "test", "-v", ".", "-args", "--mode=slave")

	err := slave.Start()
	if err != nil {
		t.Fatal(err)
	}

	// crucial
	time.Sleep(1 * time.Second)

	testMaster(t)

	err = slave.Wait()
	if err != nil {
		t.Fatal(err)
	}
}

func testSlave(t *testing.T) {
	os.Remove(SockPath)
	l, err := Listen(SockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	defer os.Remove(SockPath)

	conn, err := l.AcceptSlave()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	err = conn.handleExec()
	if err != nil {
		t.Fatal(err)
	}
}

func testMaster(t *testing.T) {
	proc, err := Exec(SockPath, "cat")
	if err != nil {
		t.Fatal(err)
	}

	proc.Stdin.Write([]byte(Secret))
	proc.Stdin.Close()

	proc.EOF()

	byt, err := ioutil.ReadAll(proc.Stdout)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Print("====== output start ======\n")
	fmt.Print(string(byt))
	fmt.Print("====== output end ======\n")

	if string(byt) != Secret {
		t.Fatalf("%s != %s", string(byt), Secret)
	}
}
