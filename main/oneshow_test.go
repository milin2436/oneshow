package main

import (
	"testing"

	"github.com/milin2436/oneshow/cmd"
)

func TestSetFunsRegistersAllCommands(t *testing.T) {
	ct := cmd.NewContext()
	setFuns(ct)

	want := []string{"ls", "rm", "info", "d", "auth", "u", "web", "webdav", "users", "su", "saveUser", "who", "search", "mv"}
	if len(ct.CmdMap) != len(want) {
		t.Fatalf("registered %d commands, want %d", len(ct.CmdMap), len(want))
	}
	for _, name := range want {
		if ct.CmdMap[name] == nil {
			t.Errorf("command %q not registered", name)
		}
		if ct.CmdMap[name].Cmd == nil {
			t.Errorf("command %q has no handler", name)
		}
	}
}

// TestEveryCommandHelpWorks parses -h for each registered command and runs it.
// Every handler returns inside the -h branch, so this never touches the network.
func TestEveryCommandHelpWorks(t *testing.T) {
	ct := cmd.NewContext()
	setFuns(ct)

	for name, pro := range ct.CmdMap {
		if err := ct.ParseArgs([]string{"-h"}, pro); err != nil {
			t.Fatalf("%s: parse -h failed: %v", name, err)
		}
		pro.Cmd(pro)
	}
}
