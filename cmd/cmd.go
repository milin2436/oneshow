package  cmd
import(
    "fmt"
	"os"
    "errors"
)
type CMD func(*Program);

type Program struct{
    Name string
	Desc string 
    Usage string
    Cmd CMD
	ParamDefMap map[string]*ParamDef
    Target string
}

type ParamGroup struct{
    Name string
    Value string
    NeedValue bool
}

type ParamDef struct{
    Name string // -t 10
    LongName string // --time=10
    NeedValue bool
    Desc string
}
type Context struct{
	Single *Program
	CmdMap map[string]*Program 
	ParamGroupMap map[string]*ParamGroup 
}

var (
	VERSION  = "unknown"
    CmdName = "mon"
	debug = false 
)

func (ct *Context) ShowHelp(){
	fmt.Println("HELP ===========================")
	fmt.Printf("%s version %s \n",CmdName,VERSION)
	fmt.Println("================================\n\n")
	for k,p := range ct.CmdMap {
		fmt.Printf("%-15s %s\n\n",k,p.Desc)
	}
}
func PrintCmdHelp(pro *Program){
	fmt.Println(pro.Usage+"\n")
	fmt.Println(pro.Desc+"\n")
	for _,p := range pro.ParamDefMap {
        fmt.Printf("-%s  %s\n\n",p.Name,p.Desc)
	}
}

func (ct *Context) GetDefCmdMap(){
	ct.CmdMap  = map[string]*Program{}

    pro := new(Program)
    pro.Name = "test"
    pro.Desc = "this is test command"
    pro.Usage = "usage: "+pro.Name+" [OPTION]... [target]"
	pro.Cmd = func(pro *Program){
        if ct.ParamGroupMap["h"] != nil {
            PrintCmdHelp(pro)
            return
        }
        fmt.Println("This is a test function")
        
	}
    pro.ParamDefMap = map[string]*ParamDef{}

    pro.ParamDefMap["p"] = &ParamDef{
            "p",
            "port",
            true,
            "set port of server"}

    pro.ParamDefMap["h"] = &ParamDef{
            "h",
            "help",
            false,
            "print help"}
    pro.ParamDefMap["H"] = &ParamDef{
            "H",
            "host",
            true,
            "set host name of server"}
    pro.ParamDefMap["v"] = &ParamDef{
            "v",
            "verbose",
            false,
            "verbosely list running processed"}


	ct.CmdMap[pro.Name] = pro

    // next program
}
func Debug(v ...interface{}) {
	if debug == true {
        logs := []interface{}{}
        logs = append(logs,"DEBUG@")
        logs = append(logs,v...)
		fmt.Println(logs...)
	}
}

func (ct *Context)ParseArgs(args []string,program *Program) error {
    length := len(args)
    findParam := true
    param := ""
    curParam := (*ParamGroup)(nil)
    for i := 2; i < length ; i++ {
        Debug("inter curParam =>",curParam,"  i=",i)

        param = args[i]

        if findParam {
            if param[0] == '-' {
                Debug("find param = ",param[1:]);
                pf := program.ParamDefMap[param[1:]]
                if pf != nil {
                    curParam = new(ParamGroup)
                    curParam.Name = pf.Name
                    curParam.NeedValue = pf.NeedValue

                    if pf.NeedValue {
                        findParam = false
                    } else {
                        //save to ParamGroup
                        curParam.Value = param
                        ct.ParamGroupMap[curParam.Name] = curParam

                        //reset findParam and curParam
                        findParam = true
                        curParam = nil
                    }
                } else {
                    Debug("ignore this param = ",param)
                }
                continue
            }
            
            //find target value
            Debug("find target = ",param);
        } else {
            if param[0] == '-' {
                Debug("ignore this param = ",param)
                return errors.New("can not get value of "+curParam.Name)
            }

            if curParam == nil {
                return errors.New("curParam must be a non null point for "+param)
            }

            //save to ParamGroup
            curParam.Value = param
            ct.ParamGroupMap[curParam.Name] = curParam

            //reset findParam and curParam
            findParam = true
            curParam = nil
            continue
        }

        if (i+1) == length {
            program.Target = args[i];
        }
    }

    if curParam != nil {
        return errors.New("can not get value of "+curParam.Name)
    }
    return nil
}
func NewContext() *Context{
    ct := new(Context)
	//ct.CmdMap  = map[string]*Program{}
	ct.ParamGroupMap =  map[string]*ParamGroup{}
    return ct
}
func Getpid() int {
	return os.Getpid()
}
func (ct *Context) Run(){
	if ct.Single != nil {
		app := ct.Single
		sargs := []string{"dd"}
		sargs =  append(sargs,os.Args...)
		err := ct.ParseArgs(sargs,app)
		if err != nil {
			fmt.Printf("paramters error : %s \n\n",err.Error())
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
            err := ct.ParseArgs(os.Args,fn)
            if err != nil {
				fmt.Printf("paramters error : %s \n\n",err.Error())
                PrintCmdHelp(fn)
                return
            }
            fn.Cmd(fn)
			return
		}
	}
	ct.ShowHelp()
}
