package polecat

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/steveyegge/gastown/internal/testutil"
	"github.com/steveyegge/gastown/internal/tmux"
)

func TestMain(m *testing.M) {
	socket := fmt.Sprintf("gt-polecat-test-%d", os.Getpid())
	tmux.SetDefaultSocket(socket)
	code := m.Run()
	_ = exec.Command("tmux", "-L", socket, "kill-server").Run()
	tmux.SetDefaultSocket("")
	testutil.TerminateDoltContainer()
	os.Exit(code)
}
