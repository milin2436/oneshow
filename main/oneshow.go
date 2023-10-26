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

func taskWorker(goid int, cli *one.OneClient) {
	for {
		msg, ok := <-taskQueue
		if ok {
			fmt.Println("upload file = ", msg.Path)
			descFile, err := cli.APIGetFile(cli.CurDriveID, msg.Desc)
			if err == nil && descFile != nil {
				fmt.Println("file existed in onedrive,file = ", msg.Desc)
				continue
			}
			tryLimit := 500
			for ti := 1; ti <= tryLimit; ti++ {
				err = msg.Cli.UploadBigFile(msg.Path, msg.Cli.CurDriveID, msg.Desc)
				if err != nil {
					fmt.Println("err = ", err, " in ", msg.Path)
					fmt.Printf("try again for the %dth time\n", ti)
					//exitQueue <- 0
				} else {
					fmt.Println("done file = ", msg.Path)
					break
				}
			}
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
		localFilePath := filepath.Join(descDir, f.Name)
		localFilePathTmp := filepath.Join(descDir, f.Name+".finfo")
		if one.PathExists(localFilePath) && !one.PathExists(localFilePathTmp) {
			fmt.Println("The file exists,skip it : ", localFilePath)
			continue
		}
		cli.Download(path, descDir)
	}
}
func Download(cli *one.OneClient, downloadDir string, dirPath string) {
	dirPath = one.GetOnedrivePath(dirPath)
	info, err := cli.APIGetFile(cli.CurDriveID, dirPath)
	if err != nil {
		fmt.Println("err =", err)
		return
	}
	go AutoUpdateToken(cli)
	if info.Folder != nil {
		batchDownload(cli, dirPath, downloadDir)
	} else {
		cli.Download(dirPath, downloadDir)
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
	pro.ParamDefMap["d"] = &cmd.ParamDef{
		Name:      "d",
		LongName:  "direct_url",
		NeedValue: false,
		Desc:      "list files direct url"}

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
		cli, err := one.NewOneClient()
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
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
				fmt.Printf("%-10s%-16s%-28s%-100s\n", one.ViewHumanShow(v.Size), v.CreatedBy.User.DisplayName, dsTime, Name)
			}
		} else if ct.ParamGroupMap["d"] != nil {
			for _, v := range ret.Value {
				fmt.Printf("[%s]%-200s\n\n", v.Name, v.DownloadURL)
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
		cli, err := one.NewOneClient()
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
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
		cli, err := one.NewOneClient()
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
		drive, err := cli.APIGetMeDrive()
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
		fmt.Printf("%-20s%s\n", "drive type", drive.DriveType)
		fmt.Printf("%-20s%s\n", "state", drive.Quota.State)
		fmt.Printf("%-20s%s\n", "ower", drive.Owner.User.DisplayName)
		fmt.Printf("%-20s%s\n", "total", one.ViewHumanShow(drive.Quota.Total))
		fmt.Printf("%-20s%s\n", "used", one.ViewHumanShow(drive.Quota.Used))
		fmt.Printf("%-20s%s\n", "Remaing", one.ViewHumanShow(drive.Quota.Remaining))
		fmt.Printf("%-20s%s\n", "trash", one.ViewHumanShow(drive.Quota.Deleted))
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
		cli, err := one.NewOneClient()
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
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
		Download(cli, downloadDir, dirPath)
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
		cli, err := one.NewOneClient()
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
		go AutoUpdateToken(cli)
		//handle http source
		if strings.HasPrefix(srcFile.Value, "http://") || strings.HasPrefix(srcFile.Value, "https://") {
			cli.UploadSourceTryAgain(srcFile.Value, cli.CurDriveID, fn.Value, 100)
			return
		}
		fileInfo, err := os.Stat(srcFile.Value)
		if err != nil {
			fmt.Println("file does not exit  : ", srcFile.Value)
			return
		}
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
				go taskWorker(i, cli)
			}
			batchUpload(cli, srcFile.Value, fn.Value)
			close(taskQueue)
			for j := 0; j < threadCnt; j++ {
				goid := <-exitQueue
				fmt.Println("got closed thread. goid = ", goid)
			}
			fmt.Println("done all.")
		} else {
			/*
				err := cli.UploadBigFile(srcFile.Value, cli.CurDriveID, filepath.Join(fn.Value, fileInfo.Name()))
				if err != nil {
					fmt.Println("upload file to failed")
				}
			*/
			cli.UploadSourceTryAgain(srcFile.Value, cli.CurDriveID, fn.Value, 100)
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
	pro.Name = "webdav"
	pro.Desc = "run webdav service for onedirve (only read)(beta version)"
	pro.Usage = "usage: " + pro.Name + " [OPTION]"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		"h",
		"help",
		false,
		"print help"}
	pro.ParamDefMap["u"] = &cmd.ParamDef{
		"u",
		"url",
		true,
		"setup listen address for this service,as -u :5555"}
	pro.ParamDefMap["user"] = &cmd.ParamDef{
		"user",
		"user",
		true,
		"setup webdav user"}
	pro.ParamDefMap["passwd"] = &cmd.ParamDef{
		"passwd",
		"password",
		true,
		"setup webdav password"}
	pro.ParamDefMap["c"] = &cmd.ParamDef{
		"c",
		"cert",
		true,
		"setup https cert file"}
	pro.ParamDefMap["k"] = &cmd.ParamDef{
		"k",
		"key",
		true,
		"setup webdav key file"}
	pro.ParamDefMap["ss"] = &cmd.ParamDef{
		"ss",
		"serverlist",
		true,
		"server list as 0all;all1"}
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
		userp := ct.ParamGroupMap["user"]
		user := ""
		if userp != nil {
			user = userp.Value
		}
		pwdp := ct.ParamGroupMap["passwd"]
		passwd := ""
		if pwdp != nil {
			passwd = pwdp.Value
		}

		cp := ct.ParamGroupMap["c"]
		cert := ""
		if cp != nil {
			cert = cp.Value
		}
		kp := ct.ParamGroupMap["k"]
		key := ""
		if kp != nil {
			key = kp.Value
		}
		ssp := ct.ParamGroupMap["ss"]
		ss := ""
		if ssp != nil {
			ss = ssp.Value
		}
		fmt.Println("sss = ", ss)
		Webdav(address, user, passwd, cert, key, ss)
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
	pro.Name = "who"
	pro.Desc = "show current user name"
	pro.Usage = "usage: " + pro.Name
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
		userName, err := Who()
		if err != nil {
			fmt.Println("who command call failed, err = ", err)
		} else {
			fmt.Println("current user:", userName)
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
	pro.ParamDefMap["dn"] = &cmd.ParamDef{
		"dn",
		"download",
		false,
		"download file for search result,default save files to search-dn directory,and depend -d flag"}

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
		cli, err := one.NewOneClient()
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
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
		isDownload := false
		if ct.ParamGroupMap["d"] != nil {
			detail = true
		}
		if ct.ParamGroupMap["dn"] != nil {
			isDownload = true
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
			OName := pre + v.Name
			Name := OName
			if v.Folder != nil {
				Name = Name + "/"
			}
			fmt.Printf("%s\n", Name)
			if detail && isDownload {
				pindex := strings.Index(OName, "/root:/")
				if pindex > -1 {
					desc := OName[pindex+6:]
					fmt.Println("will download ", desc)
					Download(cli, "search-dn", desc)
				}
			}
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
		cli, err := one.NewOneClient()
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
		file := fp.Value
		dir = one.GetOnedrivePath(dir)
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
	one.InitOneShowConfig()
	ct := cmd.NewContext()
	setFuns(ct)
	ct.Run()
}
