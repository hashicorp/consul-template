package child

import (
	"io/ioutil"
	"os"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/hashicorp/go-gatedio"
)

func testChild(t *testing.T) *Child {
	c, err := New(&NewInput{
		Stdout:       ioutil.Discard,
		Stderr:       ioutil.Discard,
		Command:      "echo",
		Args:         []string{"hello", "world"},
		ReloadSignal: os.Interrupt,
		KillSignal:   os.Kill,
		KillTimeout:  10 * time.Millisecond,
		Splay:        0 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNew(t *testing.T) {
	t.Parallel()

	stdin := gatedio.NewByteBuffer()
	stdout := gatedio.NewByteBuffer()
	stderr := gatedio.NewByteBuffer()
	command := "echo"
	args := []string{"hello", "world"}
	reloadSignal := os.Interrupt
	killSignal := os.Kill
	killTimeout := 2 * time.Second
	splay := 2 * time.Second

	c, err := New(&NewInput{
		Stdin:        stdin,
		Stdout:       stdout,
		Stderr:       stderr,
		Command:      command,
		Args:         args,
		ReloadSignal: reloadSignal,
		KillSignal:   killSignal,
		KillTimeout:  killTimeout,
		Splay:        splay,
	})
	if err != nil {
		t.Fatal(err)
	}

	if c.stdin != stdin {
		t.Errorf("expected %q to be %q", c.stdin, stdin)
	}

	if c.stdout != stdout {
		t.Errorf("expected %q to be %q", c.stdout, stdout)
	}

	if c.stderr != stderr {
		t.Errorf("expected %q to be %q", c.stderr, stderr)
	}

	if c.command != command {
		t.Errorf("expected %q to be %q", c.command, command)
	}

	if !reflect.DeepEqual(c.args, args) {
		t.Errorf("expected %q to be %q", c.args, args)
	}

	if c.reloadSignal != reloadSignal {
		t.Errorf("expected %q to be %q", c.reloadSignal, reloadSignal)
	}

	if c.killSignal != killSignal {
		t.Errorf("expected %q to be %q", c.killSignal, killSignal)
	}

	if c.killTimeout != killTimeout {
		t.Errorf("expected %q to be %q", c.killTimeout, killTimeout)
	}

	if c.splay != splay {
		t.Errorf("expected %q to be %q", c.splay, splay)
	}

	if c.stopCh == nil {
		t.Errorf("expected %#v to be", c.stopCh)
	}
}

func TestNew_errMissingCommand(t *testing.T) {
	t.Parallel()

	_, err := New(nil)
	if err == nil {
		t.Fatal("expected error")
	}

	if err != ErrMissingCommand {
		t.Errorf("expected %q to be %q", err, ErrMissingCommand)
	}
}

func TestExitCh_noProcess(t *testing.T) {
	t.Parallel()

	c := testChild(t)
	ch := c.ExitCh()
	if ch != nil {
		t.Errorf("expected %#v to be nil", ch)
	}
}

func TestExitCh(t *testing.T) {
	t.Parallel()

	c := testChild(t)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Stop()

	ch := c.ExitCh()
	if ch == nil {
		t.Error("expected ch to exist")
	}
}

func TestPid_noProcess(t *testing.T) {
	t.Parallel()

	c := testChild(t)
	pid := c.Pid()
	if pid != 0 {
		t.Errorf("expected %q to be 0", pid)
	}
}

func TestPid(t *testing.T) {
	t.Parallel()

	c := testChild(t)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Stop()

	pid := c.Pid()
	if pid == 0 {
		t.Error("expected pid to not be 0")
	}
}

func TestStart(t *testing.T) {
	t.Parallel()

	c := testChild(t)

	// Set our own reader and writer so we can verify they are wired to the child.
	stdin := gatedio.NewByteBuffer()
	stdout := gatedio.NewByteBuffer()
	stderr := gatedio.NewByteBuffer()
	c.stdin = stdin
	c.stdout = stdout
	c.stderr = stderr

	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Stop()

	select {
	case <-c.ExitCh():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("process should have exited")
	}

	expected := "hello world\n"
	if stdout.String() != expected {
		t.Errorf("expected %q to be %q", stdout.String(), expected)
	}
}

func TestSignal(t *testing.T) {
	t.Parallel()

	c := testChild(t)
	c.command = "bash"
	c.args = []string{"-c", "trap 'echo one' SIGUSR1; while true; do :; done"}

	out := gatedio.NewByteBuffer()
	c.stdout, c.stderr = out, out

	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Stop()

	// For some reason bash doesn't start immediately
	time.Sleep(2 * time.Second)

	if err := c.Signal(syscall.SIGUSR1); err != nil {
		t.Fatal(err)
	}

	// Give time for the file to flush
	time.Sleep(2 * time.Second)

	// Stop the child now
	c.Stop()

	expected := "one\n"
	if out.String() != expected {
		t.Errorf("expected %q to be %q", out.String(), expected)
	}
}

func TestSignal_noProcess(t *testing.T) {
	t.Parallel()

	c := testChild(t)
	if err := c.Signal(syscall.SIGUSR1); err != nil {
		// Just assert there is no error
		t.Fatal(err)
	}
}

func TestReload_signal(t *testing.T) {
	t.Parallel()

	c := testChild(t)
	c.command = "bash"
	c.args = []string{"-c", "trap 'echo one' SIGUSR1; while true; do :; done"}
	c.reloadSignal = syscall.SIGUSR1

	out := gatedio.NewByteBuffer()
	c.stdout, c.stderr = out, out

	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Stop()

	// For some reason bash doesn't start immediately
	time.Sleep(2 * time.Second)

	if err := c.Reload(); err != nil {
		t.Fatal(err)
	}

	// Give time for the file to flush
	time.Sleep(2 * time.Second)

	// Stop the child now
	c.Stop()

	expected := "one\n"
	if out.String() != expected {
		t.Errorf("expected %q to be %q", out.String(), expected)
	}
}

func TestReload_noSignal(t *testing.T) {
	t.Parallel()

	c := testChild(t)
	c.command = "bash"
	c.args = []string{"-c", "while true; do :; done"}
	c.reloadSignal = nil

	out := gatedio.NewByteBuffer()
	c.stdout, c.stderr = out, out

	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Stop()

	// For some reason bash doesn't start immediately
	time.Sleep(2 * time.Second)

	// Grab the original pid
	opid := c.cmd.Process.Pid

	if err := c.Reload(); err != nil {
		t.Fatal(err)
	}

	// Give time for the file to flush
	time.Sleep(2 * time.Second)

	// Get the new pid
	npid := c.cmd.Process.Pid

	// Stop the child now
	c.Stop()

	if opid == npid {
		t.Error("expected new process to restart")
	}
}

func TestReload_noProcess(t *testing.T) {
	t.Parallel()

	c := testChild(t)
	c.reloadSignal = syscall.SIGUSR1
	if err := c.Reload(); err != nil {
		t.Fatal(err)
	}
}

func TestKill_signal(t *testing.T) {
	t.Parallel()

	c := testChild(t)
	c.command = "bash"
	c.args = []string{"-c", "trap 'echo one' SIGUSR1; while true; do :; done"}
	c.killSignal = syscall.SIGUSR1

	out := gatedio.NewByteBuffer()
	c.stdout, c.stderr = out, out

	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Stop()

	// For some reason bash doesn't start immediately
	time.Sleep(500 * time.Millisecond)

	c.Kill()

	// Give time for the file to flush
	time.Sleep(500 * time.Millisecond)

	// Stop the child now
	c.Stop()

	expected := "one\n"
	if out.String() != expected {
		t.Errorf("expected %q to be %q", out.String(), expected)
	}
}

func TestKill_noSignal(t *testing.T) {
	t.Parallel()

	c := testChild(t)
	c.command = "bash"
	c.args = []string{"-c", "while true; do :; done"}
	c.killSignal = nil

	out := gatedio.NewByteBuffer()
	c.stdout, c.stderr = out, out

	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer c.Stop()

	// For some reason bash doesn't start immediately
	time.Sleep(2 * time.Second)

	c.Kill()

	// Give time for the file to flush
	time.Sleep(2 * time.Second)

	if c.cmd != nil {
		t.Errorf("expected cmd to be nil")
	}
}

func TestKill_noProcess(t *testing.T) {
	t.Parallel()

	c := testChild(t)
	c.killSignal = syscall.SIGUSR1
	c.Kill()
}
