package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// CMD is the handler executed when a subcommand is invoked.
type CMD func(*Program)

// Program describes a subcommand and its parameter definitions.
type Program struct {
	Name        string
	Desc        string
	Usage       string
	Cmd         CMD
	ParamDefMap map[string]*ParamDef
	Target      string   // last positional argument
	Args        []string // all positional arguments, in order
}

// ParamGroup is one parsed instance of a parameter.
type ParamGroup struct {
	Name      string
	Value     string
	NeedValue bool
}

// ParamDef declares a parameter of a Program. Name is the short form used as
// the map key (e.g. "h" for -h, "dn" for -dn); LongName enables the --long form.
type ParamDef struct {
	Name      string // -t 10
	LongName  string // --time=10
	NeedValue bool
	Desc      string
}

// Context holds the registered commands, the parsed parameters of the current
// run, and (in single-command mode) the one Program to run.
type Context struct {
	Single        *Program
	CmdMap        map[string]*Program
	ParamGroupMap map[string]*ParamGroup
	Name          string // shown in help header; defaults to CmdName
	Version       string // shown in help header; defaults to VERSION
	Debug         bool   // enables debug logging during parsing
}

// CmdName and VERSION are injected at build time via -ldflags -X (see
// main/Makefile) and copied onto each Context at NewContext, so help output
// does not read package globals directly.
var (
	CmdName = "cmd"
	VERSION = "unknown"
)

// NewContext creates an empty Context. Register commands by filling
// ct.CmdMap, or set ct.Single for single-command mode.
func NewContext() *Context {
	ct := new(Context)
	ct.Name = CmdName
	ct.Version = VERSION
	ct.ParamGroupMap = map[string]*ParamGroup{}
	return ct
}

// Getpid returns the current process id.
func Getpid() int {
	return os.Getpid()
}

// debug prints to stdout when ct.Debug is enabled.
func (ct *Context) debug(v ...interface{}) {
	if ct.Debug {
		fmt.Println(append([]interface{}{"DEBUG@"}, v...)...)
	}
}

var (
	useColor   = isTerminal()
	colorReset = "\x1b[0m"
	colorBold  = "\x1b[1m"
	colorCyan  = "\x1b[36m"
	colorGreen = "\x1b[32m"
)

// isTerminal reports whether stdout is attached to a terminal, so help text
// can use ANSI colors without polluting redirected output.
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// paint wraps s in the ANSI escape code when color output is enabled.
func paint(code, s string) string {
	if !useColor {
		return s
	}
	return code + s + colorReset
}

// pad right-pads s with spaces to the given width.
func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// cleanUsage strips a leading "usage:" token from a developer-authored usage
// string so it reads cleanly under the "Usage:" label.
func cleanUsage(u string) string {
	u = strings.TrimSpace(u)
	if lower := strings.ToLower(u); strings.HasPrefix(lower, "usage:") {
		u = strings.TrimSpace(u[len("usage:"):])
	}
	return u
}

// ShowHelp lists every registered command.
func (ct *Context) ShowHelp() {
	title := ct.Name
	if ct.Version != "" {
		title += " " + ct.Version
	}
	fmt.Println(paint(colorBold, title))
	fmt.Println("")
	fmt.Println(paint(colorCyan, "Usage:"))
	fmt.Printf("  %s <command> [options] [args]\n", ct.Name)

	if len(ct.CmdMap) == 0 {
		fmt.Println("\nno commands registered")
		return
	}

	fmt.Println("\n" + paint(colorCyan, "Commands:"))
	names := make([]string, 0, len(ct.CmdMap))
	for k := range ct.CmdMap {
		names = append(names, k)
	}
	sort.Strings(names)

	width := 0
	for _, k := range names {
		if len(k) > width {
			width = len(k)
		}
	}
	for _, k := range names {
		fmt.Printf("  %s  %s\n", paint(colorGreen, pad(k, width)), ct.CmdMap[k].Desc)
	}
	fmt.Printf("\nRun '%s <command> --help' for details on a command.\n", ct.Name)
}

// PrintCmdHelp prints the usage and parameters of one command.
func PrintCmdHelp(pro *Program) {
	fmt.Println(paint(colorBold, pro.Name))
	if pro.Desc != "" {
		fmt.Println(pro.Desc)
	}
	if usage := cleanUsage(pro.Usage); usage != "" {
		fmt.Println("\n" + paint(colorCyan, "Usage:"))
		fmt.Printf("  %s\n", usage)
	}

	if len(pro.ParamDefMap) == 0 {
		return
	}
	fmt.Println("\n" + paint(colorCyan, "Options:"))
	names := make([]string, 0, len(pro.ParamDefMap))
	for k := range pro.ParamDefMap {
		names = append(names, k)
	}
	sort.Strings(names)

	type opt struct {
		left string
		desc string
	}
	opts := make([]opt, 0, len(names))
	width := 0
	for _, k := range names {
		p := pro.ParamDefMap[k]
		left := "-" + p.Name
		if p.LongName != "" && p.LongName != p.Name {
			left += ", --" + p.LongName
		}
		if p.NeedValue {
			left += " <value>"
		}
		if len(left) > width {
			width = len(left)
		}
		opts = append(opts, opt{left, p.Desc})
	}
	for _, o := range opts {
		fmt.Printf("  %s  %s\n", paint(colorGreen, pad(o.left, width)), o.desc)
	}
}

// Run parses os.Args and dispatches. In single-command mode (ct.Single set)
// it parses os.Args[1:]; otherwise it looks up the command in os.Args[1] and
// parses os.Args[2:]. Unknown commands and parse errors print help.
func (ct *Context) Run() {
	if ct.Single != nil {
		app := ct.Single
		if err := ct.ParseArgs(os.Args[1:], app); err != nil {
			fmt.Printf("parameters error : %s \n\n", err.Error())
			PrintCmdHelp(app)
			return
		}
		app.Cmd(app)
		return
	}
	if ct.CmdMap == nil {
		fmt.Println("CmdMap can not be empty")
		return
	}
	if len(os.Args) > 1 {
		fn := ct.CmdMap[os.Args[1]]
		if fn != nil && fn.Cmd != nil {
			if err := ct.ParseArgs(os.Args[2:], fn); err != nil {
				fmt.Printf("parameters error : %s \n\n", err.Error())
				PrintCmdHelp(fn)
				return
			}
			fn.Cmd(fn)
			return
		}
	}
	ct.ShowHelp()
}

// ParseArgs parses args (the arguments after the command name) against the
// program's parameter definitions. Results are stored in ct.ParamGroupMap;
// positional arguments are stored in program.Args, with the last one also in
// program.Target. Supported forms:
//
//	-h          boolean flag, short form
//	--help      boolean flag, long form (matched by LongName)
//	-d <dir>    flag with a value taken from the next argument
//	-d=<dir>    flag with an inline value
//	--dir=<dir> long flag with an inline value
//	--          everything after is treated as a positional argument
//
// Unknown options and flags missing their value return an error. State left
// over from a previous parse is cleared before parsing.
func (ct *Context) ParseArgs(args []string, program *Program) error {
	// reset any state left over from a previous parse
	ct.ParamGroupMap = map[string]*ParamGroup{}
	program.Target = ""
	program.Args = nil

	var curParam *ParamGroup
	positionals := []string{}

	for i := 0; i < len(args); i++ {
		param := args[i]
		ct.debug("parse arg =", param)

		if curParam != nil {
			// a previous flag is waiting for its value
			if param == "" || param[0] == '-' {
				return fmt.Errorf("can not get value of %s", curParam.Name)
			}
			curParam.Value = param
			ct.ParamGroupMap[curParam.Name] = curParam
			curParam = nil
			continue
		}

		if param == "" {
			continue
		}
		if param == "-" {
			positionals = append(positionals, param)
			continue
		}
		if param[0] != '-' {
			positionals = append(positionals, param)
			continue
		}
		if param == "--" {
			// everything after "--" is a positional argument
			positionals = append(positionals, args[i+1:]...)
			break
		}

		pf := resolveParamDef(program.ParamDefMap, param)
		if pf == nil {
			return fmt.Errorf("unknown option: %s", param)
		}
		pg := &ParamGroup{Name: pf.Name, NeedValue: pf.NeedValue}
		if pf.NeedValue {
			if v, ok := flagValue(param); ok {
				pg.Value = v
				ct.ParamGroupMap[pg.Name] = pg
			} else {
				curParam = pg // value comes from the next argument
			}
		} else {
			pg.Value = param
			ct.ParamGroupMap[pg.Name] = pg
		}
	}

	if curParam != nil {
		return fmt.Errorf("can not get value of %s", curParam.Name)
	}
	if len(positionals) > 0 {
		program.Args = positionals
		program.Target = positionals[len(positionals)-1]
	}
	return nil
}

// Has reports whether the parameter name was given on the command line.
// name is the ParamDef key (the short form, e.g. "l" for -l / --list).
func (ct *Context) Has(name string) bool {
	return ct.ParamGroupMap[name] != nil
}

// Need reports whether every one of the named parameters was given.
func (ct *Context) Need(names ...string) bool {
	for _, n := range names {
		if !ct.Has(n) {
			return false
		}
	}
	return true
}

// Any reports whether at least one of the named parameters was given.
func (ct *Context) Any(names ...string) bool {
	for _, n := range names {
		if ct.Has(n) {
			return true
		}
	}
	return false
}

// Get returns the value of the named parameter, or def when it was not
// given. An explicitly empty value is returned as-is.
func (ct *Context) Get(name, def string) string {
	if p := ct.ParamGroupMap[name]; p != nil {
		return p.Value
	}
	return def
}

// resolveParamDef finds the ParamDef a flag token refers to. Short Name is
// tried first, then LongName, so both -h and --help resolve to the "h" def.
func resolveParamDef(defs map[string]*ParamDef, token string) *ParamDef {
	name := strings.TrimLeft(token, "-")
	if i := strings.Index(name, "="); i >= 0 {
		name = name[:i]
	}
	if name == "" {
		return nil
	}
	if pd := defs[name]; pd != nil {
		return pd
	}
	for _, pd := range defs {
		if pd.LongName == name {
			return pd
		}
	}
	return nil
}

// flagValue returns the inline value of a token like "-d=dir" or "--dir=dir".
func flagValue(token string) (string, bool) {
	if i := strings.Index(token, "="); i >= 0 {
		return token[i+1:], true
	}
	return "", false
}
