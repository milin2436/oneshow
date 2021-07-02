package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/milin2436/oneshow/cmd"
	"github.com/milin2436/oneshow/core"
	"github.com/milin2436/oneshow/one"
)

//OneTask a task for thread
type OneTask struct {
	Path string
	Cli  *one.OneClient
	Desc string
	ID   int
}

var taskQueue chan *OneTask
var exitQueue chan int

//worker thread count
var threadCnt = 4

func getOnedrivePath(dirPath string) string {
	if dirPath == "" {
		dirPath = "/"
	}
	strLen := len(dirPath)
	if strLen > 1 && dirPath[strLen-1] == '/' {
		dirPath = dirPath[:strLen-1]
	}
	return dirPath
}
func taskWorker(goid int) {
	for {
		msg, ok := <-taskQueue
		if ok {
			fmt.Println("upload file = ", msg.Path)
			descFile, err := cli.APIGetFile(cli.CurDriveID, msg.Desc)
			if err == nil && descFile != nil {
				fmt.Println("file existed in onedrive,file = ", msg.Desc)
				continue
			}
			err = msg.Cli.UploadBigFile(msg.Path, msg.Cli.CurDriveID, msg.Desc)
			if err != nil {
				fmt.Println("err = ", err, " in ", msg.Path)
				exitQueue <- 0
				continue
			}
			fmt.Println("done file = ", msg.Path)
		} else {
			exitQueue <- goid
			fmt.Println("will close go thread ,ID = ", goid)
			break
		}
	}
}

func batchUpload(cli *one.OneClient, curDir string, descDir string) {
	fileList, err := ioutil.ReadDir(curDir)
	if err != nil {
		fmt.Println("error in loop dir,err = ", err)
		return
	}
	//taskCnt := 0
	for _, f := range fileList {
		info := f
		if f.IsDir() {
			batchUpload(cli, filepath.Join(curDir, info.Name()), filepath.Join(descDir, info.Name()))
			continue
		}
		path := filepath.Join(curDir, info.Name())
		if strings.HasSuffix(info.Name(), ".one.tmp") {
			fmt.Println("skip ", path)
			continue
		}
		msg := new(OneTask)
		msg.Path = path
		msg.Desc = filepath.Join(descDir, info.Name())
		msg.Cli = cli
		//add upload task to queue
		taskQueue <- msg
	}
}
func batchDownload(cli *one.OneClient, curDir string, descDir string) {
	fileList, err := cli.APIListFilesByPath(cli.CurDriveID, curDir)
	if err != nil {
		fmt.Println("error in loop dir,err = ", err)
		return
	}
	for _, f := range fileList.Value {
		if f.Folder != nil {
			batchDownload(cli, filepath.Join(curDir, f.Name), filepath.Join(descDir, f.Name))
			continue
		}
		path := filepath.Join(curDir, f.Name)
		fmt.Println("donwloading  ", path)
		err = os.MkdirAll(descDir, 0771)
		if err != nil {
			fmt.Println("create dir to failed ", err)
			break
		}
		cli.Download(path, descDir)
	}
}

func setFuns(ct *cmd.Context) {
	ct.CmdMap = map[string]*cmd.Program{}

	pro := new(cmd.Program)
	pro.Name = "ls"
	pro.Desc = "list onedrive path"
	pro.Usage = "usage: " + pro.Name + " [OPTION] path"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		Name:      "h",
		LongName:  "help",
		NeedValue: false,
		Desc:      "print help"}

	pro.ParamDefMap["l"] = &cmd.ParamDef{
		Name:      "l",
		LongName:  "list",
		NeedValue: false,
		Desc:      "list files detail"}

	ct.CmdMap[pro.Name] = pro
	pro.Cmd = func(pro *cmd.Program) {
		if ct.ParamGroupMap["h"] != nil {
			cmd.PrintCmdHelp(pro)
			return
		}
		dirPath := pro.Target
		if dirPath == "" {
			dirPath = "/"
		}
		strLen := len(dirPath)
		if strLen > 1 && dirPath[strLen-1] == '/' {
			dirPath = dirPath[:strLen-1]
		}
		cli := one.NewOneClient()
		ret, err := cli.APIListFilesByPath(cli.CurDriveID, dirPath)
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
		if ct.ParamGroupMap["l"] != nil {
			for _, v := range ret.Value {
				//name size owner
				mdTime := time.Time(v.LastModifiedDateTime)
				dsTime := mdTime.Local().Format(time.RFC3339)
				Name := v.Name
				if v.Folder != nil {
					Name = v.Name + "/"
				}
				fmt.Printf("%-10s%-16s%-28s%-100s\n", humanShow(v.Size), v.CreatedBy.User.DisplayName, dsTime, Name)
			}
		} else {
			for _, v := range ret.Value {
				Name := v.Name
				if v.Folder != nil {
					Name = v.Name + "/"
				}
				fmt.Printf("%s\n", Name)
			}

		}
	}

	//next remove command
	pro = new(cmd.Program)
	pro.Name = "rm"
	pro.Desc = "remove a file or dir to trash"
	pro.Usage = "usage: " + pro.Name + " [OPTION]  [file|dir]"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		Name:      "h",
		LongName:  "help",
		NeedValue: false,
		Desc:      "print help"}

	ct.CmdMap[pro.Name] = pro
	pro.Cmd = func(pro *cmd.Program) {
		if ct.ParamGroupMap["h"] != nil {
			cmd.PrintCmdHelp(pro)
			return
		}
		path := pro.Target
		if path == "" {
			fmt.Println("file path can not be empty")
			return
		}
		cli := one.NewOneClient()
		ret, err := cli.APIDelFile(cli.CurDriveID, path)
		if err != nil {
			fmt.Println("err = ", err, " ret = ", ret)
			return
		}
		fmt.Println("result = ", ret)
	}

	//print onedrive information
	pro = new(cmd.Program)
	pro.Name = "info"
	pro.Desc = "show onedrive info"
	pro.Usage = "usage: " + pro.Name + " [OPTION]  file"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		Name:      "h",
		LongName:  "help",
		NeedValue: false,
		Desc:      "print help"}

	ct.CmdMap[pro.Name] = pro
	pro.Cmd = func(pro *cmd.Program) {
		if ct.ParamGroupMap["h"] != nil {
			cmd.PrintCmdHelp(pro)
			return
		}
		cli := one.NewOneClient()
		drive, err := cli.APIGetMeDrive()
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
		fmt.Printf("%-20s%s\n", "drive type", drive.DriveType)
		fmt.Printf("%-20s%s\n", "state", drive.Quota.State)
		fmt.Printf("%-20s%s\n", "ower", drive.Owner.User.DisplayName)
		fmt.Printf("%-20s%s\n", "total", humanShow(drive.Quota.Total))
		fmt.Printf("%-20s%s\n", "used", humanShow(drive.Quota.Used))
		fmt.Printf("%-20s%s\n", "Remaing", humanShow(drive.Quota.Remaining))
		fmt.Printf("%-20s%s\n", "trash", humanShow(drive.Quota.Deleted))
	}

	//next download
	pro = new(cmd.Program)
	pro.Name = "d"
	pro.Desc = "download a file or dir or URL to local"
	pro.Usage = "usage: " + pro.Name + " [OPTION]  [file | dir | URL]"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		Name:      "h",
		LongName:  "help",
		NeedValue: false,
		Desc:      "print help"}
	pro.ParamDefMap["d"] = &cmd.ParamDef{
		Name:      "d",
		LongName:  "downloadDir",
		NeedValue: true,
		Desc:      "download dir,default current dir"}

	ct.CmdMap[pro.Name] = pro
	pro.Cmd = func(pro *cmd.Program) {
		if ct.ParamGroupMap["h"] != nil {
			cmd.PrintCmdHelp(pro)
			return
		}
		dirPath := pro.Target
		if dirPath == "" {
			fmt.Println("file or dir can not be empty")
			return
		}
		dirObj := ct.ParamGroupMap["d"]
		downloadDir := "."
		if dirObj != nil {
			downloadDir = dirObj.Value
		}
		cli = one.NewOneClient()
		//suport donwload by URL
		URL := strings.ToLower(dirPath)
		if strings.HasPrefix(URL, "http://") || strings.HasPrefix(URL, "https://") {
			wk := one.NewDWorker()
			wk.HTTPCli = cli.HTTPClient
			wk.AuthSve = cli
			wk.DownloadDir = downloadDir
			err := wk.Download(dirPath)
			if err != nil {
				fmt.Println("err = ", err)
			}
			return
		}
		dirPath = getOnedrivePath(dirPath)
		info, err := cli.APIGetFile(cli.CurDriveID, dirPath)
		if err != nil {
			fmt.Println("err =", err)
			return
		}
		go AutoUpdateToken()
		if info.Folder != nil {
			batchDownload(cli, dirPath, downloadDir)
		} else {
			cli.Download(dirPath, downloadDir)
		}
	}

	//next add new user
	pro = new(cmd.Program)
	pro.Name = "auth"
	pro.Desc = "get a auth for new user"
	pro.Usage = "usage: " + pro.Name + " [OPTION]"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		Name:      "h",
		LongName:  "help",
		NeedValue: false,
		Desc:      "print help"}

	ct.CmdMap[pro.Name] = pro
	pro.Cmd = func(pro *cmd.Program) {
		if ct.ParamGroupMap["h"] != nil {
			cmd.PrintCmdHelp(pro)
			return
		}
		cli := one.NewBaseOneClient()
		cli.DoAutoForNewUser()
	}

	//next upload little text file
    /*
	pro = new(cmd.Program)
	pro.Name = "upload"
	pro.Desc = "upload a little text file to onedrive"
	pro.Usage = "usage: " + pro.Name + " [OPTION]"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}
	pro.ParamDefMap["h"] = &cmd.ParamDef{
		Name:      "h",
		LongName:  "help",
		NeedValue: false,
		Desc:      "print help"}
	pro.ParamDefMap["f"] = &cmd.ParamDef{
		Name:      "f",
		LongName:  "fileName",
		NeedValue: true,
		Desc:      "fileName in onedrive,need full path, such as: /root/a.txt"}
	pro.ParamDefMap["c"] = &cmd.ParamDef{
		Name:      "c",
		LongName:  "content",
		NeedValue: true,
		Desc:      "file content"}

	ct.CmdMap[pro.Name] = pro
	pro.Cmd = func(pro *cmd.Program) {
		if ct.ParamGroupMap["h"] != nil {
			cmd.PrintCmdHelp(pro)
			return
		}
		fn := ct.ParamGroupMap["f"]
		content := ct.ParamGroupMap["c"]
		if fn == nil || fn.Value == "" {
			fmt.Println("file name can not be empty")
			return
		}
		if content == nil || content.Value == "" {
			fmt.Println("content can not be empty")
			return
		}
		cli := one.NewOneClient()
		_, err := cli.APIUploadText(cli.CurDriveID, fn.Value, content.Value)
		if err != nil {
			fmt.Println("upload file to failed")
		}
	}
    */

	//next upload local file or dir
	pro = new(cmd.Program)
	pro.Name = "u"
	pro.Desc = "upload a file or dir to onedrive"
	pro.Usage = "usage: " + pro.Name + " [OPTION]"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}
	pro.ParamDefMap["h"] = &cmd.ParamDef{
		Name:      "h",
		LongName:  "help",
		NeedValue: false,
		Desc:      "print help"}
	pro.ParamDefMap["f"] = &cmd.ParamDef{
		Name:      "f",
		LongName:  "fileName",
		NeedValue: true,
		Desc:      "copy to onedrive dir, such as: /root/path/to"}
	pro.ParamDefMap["s"] = &cmd.ParamDef{
		Name:      "s",
		LongName:  "src",
		NeedValue: true,
		Desc:      "source file,local file or dir."}
	pro.ParamDefMap["t"] = &cmd.ParamDef{
		Name:      "t",
		LongName:  "thread",
		NeedValue: true,
		Desc:      "setup uploaded thread count,default 4."}

	ct.CmdMap[pro.Name] = pro
	pro.Cmd = func(pro *cmd.Program) {
		if ct.ParamGroupMap["h"] != nil {
			cmd.PrintCmdHelp(pro)
			return
		}
		fn := ct.ParamGroupMap["f"]
		srcFile := ct.ParamGroupMap["s"]
		if fn == nil || fn.Value == "" {
			fmt.Println("onedrive path can not be empty")
			return
		}
		if srcFile == nil || srcFile.Value == "" {
			fmt.Println("source file can not be empty")
			return
		}
		cli = one.NewOneClient()
		fileInfo, err := os.Stat(srcFile.Value)
		if err != nil {
			fmt.Println("file does not exit  : ", srcFile.Value)
			return
		}
		go AutoUpdateToken()
		if fileInfo.IsDir() {
			tSize := ct.ParamGroupMap["t"]
			if tSize != nil {
				threadCnt, err = strconv.Atoi(tSize.Value)
				if err != nil {
					threadCnt = 4
				}
			}
			taskQueue = make(chan *OneTask, 5)
			exitQueue = make(chan int)
			for i := 0; i < threadCnt; i++ {
				go taskWorker(i)
			}
			batchUpload(cli, srcFile.Value, fn.Value)
			close(taskQueue)
			for j := 0; j < threadCnt; j++ {
				goid := <-exitQueue
				fmt.Println("got closed thread. goid = ", goid)
			}
			fmt.Println("done all.")
		} else {
			err := cli.UploadBigFile(srcFile.Value, cli.CurDriveID, filepath.Join(fn.Value, fileInfo.Name()))
			if err != nil {
				fmt.Println("upload file to failed")
			}
		}
	}
	pro = new(cmd.Program)
	pro.Name = "web"
	pro.Desc = "run this http super serivce (beta version)"
	pro.Usage = "usage: " + pro.Name + " [OPTION]"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		"h",
		"help",
		false,
		"print help"}
	pro.ParamDefMap["s"] = &cmd.ParamDef{
		"s",
		"https",
		false,
		"enable https service ,need cacert.pem ,privkey.pem on current dir"}
	pro.ParamDefMap["u"] = &cmd.ParamDef{
		"u",
		"url",
		true,
		"setup service address for this service,as -u :5555"}

	ct.CmdMap[pro.Name] = pro
	pro.Cmd = func(pro *cmd.Program) {

		if ct.ParamGroupMap["h"] != nil {
			cmd.PrintCmdHelp(pro)
			return
		}
		address := ":8080"
		upp := ct.ParamGroupMap["u"]
		if upp != nil {
			address = upp.Value
		}
		https := false
		if ct.ParamGroupMap["s"] != nil {
			https = true
		}
		Serivce(address, https)
	}
	pro = new(cmd.Program)
	pro.Name = "users"
	pro.Desc = "list login users"
	pro.Usage = "usage: " + pro.Name + " [OPTION]"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		"h",
		"help",
		false,
		"print help"}
	ct.CmdMap[pro.Name] = pro
	pro.Cmd = func(pro *cmd.Program) {
		if ct.ParamGroupMap["h"] != nil {
			cmd.PrintCmdHelp(pro)
			return
		}
		li, err := ListUsers()
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
		if len(li) == 0 {
			fmt.Println("pls call saveUser command for save a session")
			return
		}
		for _, user := range li {
			fmt.Println(user)
		}
	}
	//swich to other session
	pro = new(cmd.Program)
	pro.Name = "su"
	pro.Desc = "swich to other logined user"
	pro.Usage = "usage: " + pro.Name + " [OPTION]... [UserName]"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		"h",
		"help",
		false,
		"print help"}

	ct.CmdMap[pro.Name] = pro
	pro.Cmd = func(pro *cmd.Program) {
		if ct.ParamGroupMap["h"] != nil {
			cmd.PrintCmdHelp(pro)
			return
		}
		user := pro.Target
		if user == "" {
			fmt.Println("user name cannot be empty")
			return
		}
		err := SwitchUser(user)
		if err != nil {
			fmt.Println("err = ", err)
		} else {
			fmt.Println("switch to ", user)
		}
	}

	//next program
	pro = new(cmd.Program)
	pro.Name = "saveUser"
	pro.Desc = "save current user to name"
	pro.Usage = "usage: " + pro.Name + " [OPTION]... [UserName]"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		"h",
		"help",
		false,
		"print help"}

	ct.CmdMap[pro.Name] = pro
	pro.Cmd = func(pro *cmd.Program) {
		if ct.ParamGroupMap["h"] != nil {
			cmd.PrintCmdHelp(pro)
			return
		}
		user := pro.Target
		if user == "" {
			fmt.Println("user name cannot be empty")
			return
		}
		err := SaveUser(user)
		if err != nil {
			fmt.Println("err = ", err)
		} else {
			fmt.Println("save to ", user)
		}
	}

	//next program
	pro = new(cmd.Program)
	pro.Name = "search"
	pro.Desc = "search files by key"
	pro.Usage = "usage: " + pro.Name + " [OPTION]... key"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		"h",
		"help",
		false,
		"print help"}
	pro.ParamDefMap["d"] = &cmd.ParamDef{
		"d",
		"detail",
		false,
		"show full path of file"}

	ct.CmdMap[pro.Name] = pro
	pro.Cmd = func(pro *cmd.Program) {
		if ct.ParamGroupMap["h"] != nil {
			cmd.PrintCmdHelp(pro)
			return
		}
		key := pro.Target
		if key == "" {
			fmt.Println("Key text cannot be empty")
			return
		}
		cli = one.NewOneClient()
		ret, err := cli.APISearchByKey(cli.CurDriveID, key)
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
		if len(ret.Value) == 0 {
			fmt.Println("no results")
			return
		}
		detail := false
		if ct.ParamGroupMap["d"] != nil {
			detail = true
		}
		for _, v := range ret.Value {
			pre := ""
			if detail {
				fullV, err := cli.APIGetFileByID(cli.CurDriveID, v.ID)
				if err != nil {
					fmt.Println("err = ", err)
					continue
				}
				pre = fullV.ParentReference.Path + "/"
			}
			Name := pre + v.Name
			if v.Folder != nil {
				Name = v.Name + "/"
			}
			fmt.Printf("%s\n", Name)
		}
	}
	//next program
	pro = new(cmd.Program)
	pro.Name = "mv"
	pro.Desc = "move file to other dir"
	pro.Usage = "usage: " + pro.Name + " [OPTION]... dir"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		"h",
		"help",
		false,
		"print help"}
	pro.ParamDefMap["f"] = &cmd.ParamDef{
		"f",
		"file",
		true,
		"will move file"}

	pro.ParamDefMap["n"] = &cmd.ParamDef{
		"n",
		"newName",
		true,
		"new name"}

	ct.CmdMap[pro.Name] = pro
	pro.Cmd = func(pro *cmd.Program) {
		if ct.ParamGroupMap["h"] != nil {
			cmd.PrintCmdHelp(pro)
			return
		}
		dir := pro.Target
		if dir == "" {
			fmt.Println("dir cannot be empty")
			return
		}
		fp := ct.ParamGroupMap["f"]
		if fp == nil {
			fmt.Println("file cannot be empty")
			return
		}
		newName := ""
		np := ct.ParamGroupMap["n"]
		if np != nil {
			newName = np.Value
		}
		cli = one.NewOneClient()
		file := fp.Value
		dir = getOnedrivePath(dir)
		ifile, err := cli.APIGetFile(cli.CurDriveID, file)
		if err != nil {
			fmt.Println("file is wrong,err = ", err)
			return
		}
		idir, err := cli.APIGetFile(cli.CurDriveID, dir)
		if err != nil {
			fmt.Println("dir is wrong,err = ", err)
			return
		}
		if idir.Folder == nil {
			fmt.Println("path is not dir .path = ", dir)
			return
		}
		if newName == "" {
			newName = ifile.Name
		}
		f, err := cli.APIUpdateFileByItemID(cli.CurDriveID, ifile.ID, newName, idir.ID)
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
		fmt.Println(f)
	}
}
func main() {
	core.Debug = false
	ct := cmd.NewContext()
	setFuns(ct)
	ct.Run()
}
