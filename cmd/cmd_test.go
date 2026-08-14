package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
)

// newTestProgram mirrors the shape of a real command (see main/oneshow.go).
func newTestProgram() *Program {
	return &Program{
		Name: "test",
		ParamDefMap: map[string]*ParamDef{
			"h":  {Name: "h", LongName: "help", NeedValue: false, Desc: "print help"},
			"l":  {Name: "l", LongName: "list", NeedValue: false, Desc: "list detail"},
			"d":  {Name: "d", LongName: "dir", NeedValue: true, Desc: "set dir"},
			"dn": {Name: "dn", LongName: "download", NeedValue: false, Desc: "download"},
		},
	}
}

func parse(args []string) (*Context, *Program, error) {
	ct := NewContext()
	pro := newTestProgram()
	err := ct.ParseArgs(args, pro)
	return ct, pro, err
}

func TestShortFlag(t *testing.T) {
	ct, _, err := parse([]string{"-l"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct.ParamGroupMap["l"] == nil {
		t.Fatal("-l should set ParamGroupMap[l]")
	}
}

func TestMultiCharShortFlag(t *testing.T) {
	ct, _, err := parse([]string{"-dn"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct.ParamGroupMap["dn"] == nil {
		t.Fatal("-dn should set ParamGroupMap[dn]")
	}
}

func TestLongFlag(t *testing.T) {
	ct, _, err := parse([]string{"--help", "--list"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct.ParamGroupMap["h"] == nil {
		t.Fatal("--help should set ParamGroupMap[h]")
	}
	if ct.ParamGroupMap["l"] == nil {
		t.Fatal("--list should set ParamGroupMap[l]")
	}
}

func TestLongFlagCollidingWithShortName(t *testing.T) {
	// --h is not a declared LongName but should still resolve via Name.
	ct, _, err := parse([]string{"--h"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct.ParamGroupMap["h"] == nil {
		t.Fatal("--h should resolve to the h param")
	}
}

func TestFlagValueFromNextArg(t *testing.T) {
	ct, pro, err := parse([]string{"-d", "dir", "file"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct.ParamGroupMap["d"] == nil || ct.ParamGroupMap["d"].Value != "dir" {
		t.Fatalf("-d should hold value dir, got %+v", ct.ParamGroupMap["d"])
	}
	if pro.Target != "file" {
		t.Fatalf("target should be file, got %q", pro.Target)
	}
}

func TestInlineValueForms(t *testing.T) {
	cases := []struct {
		arg  string
		want string
	}{
		{"-d=dir", "dir"},
		{"--dir=dir", "dir"},
		{"--dir=/a/b c", "/a/b c"},
	}
	for _, c := range cases {
		ct, _, err := parse([]string{c.arg})
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.arg, err)
		}
		got := ct.ParamGroupMap["d"].Value
		if got != c.want {
			t.Fatalf("%s: value = %q, want %q", c.arg, got, c.want)
		}
	}
}

func TestUnknownOptionErrors(t *testing.T) {
	for _, arg := range []string{"-x", "--nope", "-vrebose"} {
		_, _, err := parse([]string{arg})
		if err == nil || !strings.Contains(err.Error(), "unknown option") {
			t.Fatalf("%s: expected unknown-option error, got %v", arg, err)
		}
	}
}

func TestMissingValueErrors(t *testing.T) {
	cases := []struct{ args []string }{
		{[]string{"-d"}},
		{[]string{"-d", "-l"}},
	}
	for _, c := range cases {
		_, _, err := parse(c.args)
		if err == nil || !strings.Contains(err.Error(), "can not get value of d") {
			t.Fatalf("%v: expected missing-value error, got %v", c.args, err)
		}
	}
}

func TestEmptyArgDoesNotPanic(t *testing.T) {
	ct, _, err := parse([]string{"", "-l"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct.ParamGroupMap["l"] == nil {
		t.Fatal("flag after empty arg should still be parsed")
	}
}

func TestMultiplePositionals(t *testing.T) {
	ct, pro, err := parse([]string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pro.Args) != 3 || pro.Args[0] != "a" || pro.Args[1] != "b" || pro.Args[2] != "c" {
		t.Fatalf("Args = %v, want [a b c]", pro.Args)
	}
	if pro.Target != "c" {
		t.Fatalf("Target = %q, want c", pro.Target)
	}
	if len(ct.ParamGroupMap) != 0 {
		t.Fatalf("ParamGroupMap should be empty, got %v", ct.ParamGroupMap)
	}
}

func TestDashDashTerminator(t *testing.T) {
	ct, pro, err := parse([]string{"-l", "--", "-d", "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct.ParamGroupMap["l"] == nil {
		t.Fatal("-l before -- should still be parsed")
	}
	if ct.ParamGroupMap["d"] != nil {
		t.Fatal("-d after -- should be treated as positional, not a flag")
	}
	if len(pro.Args) != 2 || pro.Args[0] != "-d" || pro.Args[1] != "x" {
		t.Fatalf("Args = %v, want [-d x]", pro.Args)
	}
}

func TestSingleDashIsPositional(t *testing.T) {
	_, pro, err := parse([]string{"-"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pro.Target != "-" {
		t.Fatalf("Target = %q, want -", pro.Target)
	}
}

func TestAccessHelpers(t *testing.T) {
	ct, _, err := parse([]string{"-l", "-d", "dir", "file"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ct.Has("l") {
		t.Fatal("Has(l) should be true")
	}
	if ct.Has("dn") {
		t.Fatal("Has(dn) should be false")
	}

	if !ct.Need("l", "d") {
		t.Fatal("Need(l, d) should be true")
	}
	if ct.Need("l", "dn") {
		t.Fatal("Need(l, dn) should be false")
	}

	if !ct.Any("dn", "d") {
		t.Fatal("Any(dn, d) should be true")
	}
	if ct.Any("dn", "h") {
		t.Fatal("Any(dn, h) should be false")
	}

	if got := ct.Get("d", "."); got != "dir" {
		t.Fatalf("Get(d) = %q, want dir", got)
	}
	if got := ct.Get("t", "4"); got != "4" {
		t.Fatalf("Get(t) default = %q, want 4", got)
	}
}

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	f()
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestShowHelpOutput(t *testing.T) {
	ct := NewContext()
	ct.Name = "oneshow"
	ct.Version = "v1.0.0"
	ct.CmdMap = map[string]*Program{
		"ls": {Name: "ls", Desc: "list directory contents"},
		"d":  {Name: "d", Desc: "download a file"},
	}
	out := captureStdout(ct.ShowHelp)
	for _, want := range []string{"oneshow v1.0.0", "Usage:", "Commands:", "ls", "d", "<command> [options] [args]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("ShowHelp output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("ShowHelp output contains ANSI codes when not a tty:\n%q", out)
	}
}

func TestPrintCmdHelpOutput(t *testing.T) {
	pro := newTestProgram()
	pro.Name = "test"
	pro.Usage = "usage: test [OPTION] path"
	out := captureStdout(func() { PrintCmdHelp(pro) })
	for _, want := range []string{"test", "Usage:", "test [OPTION] path", "Options:", "-d", "--dir", "<value>", "print help"} {
		if !strings.Contains(out, want) {
			t.Fatalf("PrintCmdHelp output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "usage:") {
		t.Fatalf("PrintCmdHelp should strip the leading 'usage:' token:\n%s", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("PrintCmdHelp output contains ANSI codes when not a tty:\n%q", out)
	}
}

func TestStateClearedBetweenParses(t *testing.T) {
	ct, pro, err := parse([]string{"-d", "x", "file"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pro.Target != "file" {
		t.Fatalf("first parse Target = %q, want file", pro.Target)
	}
	// second parse must not leak flags or target from the first
	if err := ct.ParseArgs([]string{"-l"}, pro); err != nil {
		t.Fatalf("second parse error: %v", err)
	}
	if ct.ParamGroupMap["d"] != nil {
		t.Fatal("flag d from first parse leaked into second")
	}
	if pro.Target != "" {
		t.Fatalf("Target leaked from first parse, got %q", pro.Target)
	}
	if ct.ParamGroupMap["l"] == nil {
		t.Fatal("-l from second parse should be present")
	}
}
