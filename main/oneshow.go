package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/milin2436/oneshow/v2/cmd"
	"github.com/milin2436/oneshow/v2/core"
	"github.com/milin2436/oneshow/v2/one"
)

func Download(cli *one.OneClient, downloadDir string, dirPath string, a bool) {
	dirPath = one.GetOnedrivePath(dirPath)
	info, err := cli.APIGetFile(cli.CurDriveID, dirPath)
	if err != nil {
		fmt.Println("err =", err)
		return
	}
	go AutoUpdateToken(cli)
	if info.Folder != nil {
		cli.BatchDownload(dirPath, downloadDir, a)
	} else {
		cli.Download(dirPath, downloadDir, a)
	}
}

func setFuns(ct *cmd.Context) {
	ct.CmdMap = map[string]*cmd.Program{}

	//#ls
	pro := new(cmd.Program)
	pro.Name = "ls"
	pro.Desc = "list onedrive directory contents"
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
	//#rm
	pro = new(cmd.Program)
	pro.Name = "rm"
	pro.Desc = "move a file or directory to the trash"
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
		if ret {
			fmt.Printf("removed %s \n", path)
		}
	}

	//print onedrive information
	//#info
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
	//#d
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
	pro.ParamDefMap["a"] = &cmd.ParamDef{
		Name:      "a",
		LongName:  "acceleration",
		NeedValue: false,
		Desc:      "Speed in downloads through CDN"}

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
		a := false
		if ct.ParamGroupMap["a"] != nil {
			a = true
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
			wk.Proxy = a
			err := wk.Download(dirPath)
			if err != nil {
				fmt.Println("err = ", err)
			}
			return
		}
		Download(cli, downloadDir, dirPath, a)
	}

	//next add new user
	//#auth
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
	//#u
	pro.Name = "u"
	pro.Desc = "upload a file or directory to OneDrive"
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
		Desc:      "copy to OneDrive directory, such as: /root/path/to"}
	pro.ParamDefMap["s"] = &cmd.ParamDef{
		Name:      "s",
		LongName:  "src",
		NeedValue: true,
		Desc:      "source file,local file or directory."}
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
		var threadCnt = 4
		if fileInfo.IsDir() {
			tSize := ct.ParamGroupMap["t"]
			if tSize != nil {
				threadCnt, err = strconv.Atoi(tSize.Value)
				if err != nil {
					threadCnt = 4
				}
			}
			cli.BatchUpload(threadCnt, srcFile.Value, fn.Value)
			fmt.Println("done all.")
		} else {
			cli.UploadSourceTryAgain(srcFile.Value, cli.CurDriveID, fn.Value, 100)
		}
	}
	pro = new(cmd.Program)
	//#web
	pro.Name = "web"
	pro.Desc = "run this http super serivce (beta version)"
	pro.Usage = "usage: " + pro.Name + " [OPTION]"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		Name:      "h",
		LongName:  "help",
		NeedValue: false,
		Desc:      "print help"}
	pro.ParamDefMap["s"] = &cmd.ParamDef{
		Name:      "s",
		LongName:  "https",
		NeedValue: false,
		Desc:      "enable https service ,need cacert.pem ,privkey.pem on current dir"}
	pro.ParamDefMap["u"] = &cmd.ParamDef{
		Name:      "u",
		LongName:  "url",
		NeedValue: true,
		Desc:      "setup service address for this service,as -u :5555"}

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
		StartWebSerivce(address, https)
	}
	pro = new(cmd.Program)
	//#webdav
	pro.Name = "webdav"
	pro.Desc = "run webdav service for onedirve (only read)"
	pro.Usage = "usage: " + pro.Name + " [OPTION]"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		Name:      "h",
		LongName:  "help",
		NeedValue: false,
		Desc:      "print help"}
	pro.ParamDefMap["u"] = &cmd.ParamDef{
		Name:      "u",
		LongName:  "url",
		NeedValue: true,
		Desc:      "setup listen address for this service,as -u :5555"}
	pro.ParamDefMap["user"] = &cmd.ParamDef{
		Name:      "user",
		LongName:  "user",
		NeedValue: true,
		Desc:      "setup webdav user"}
	pro.ParamDefMap["passwd"] = &cmd.ParamDef{
		Name:      "passwd",
		LongName:  "password",
		NeedValue: true,
		Desc:      "setup webdav password"}
	pro.ParamDefMap["c"] = &cmd.ParamDef{
		Name:      "c",
		LongName:  "cert",
		NeedValue: true,
		Desc:      "setup https cert file"}
	pro.ParamDefMap["k"] = &cmd.ParamDef{
		Name:      "k",
		LongName:  "key",
		NeedValue: true,
		Desc:      "setup webdav key file"}
	pro.ParamDefMap["ss"] = &cmd.ParamDef{
		Name:      "ss",
		LongName:  "serverlist",
		NeedValue: true,
		Desc:      "server list as 0all;all1"}
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
		fmt.Println("sources : ", ss)
		StartWebdavService(address, user, passwd, cert, key, ss)
	}
	pro = new(cmd.Program)
	//#users
	pro.Name = "users"
	pro.Desc = "list of logged-in users"
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
		cm := new(one.ConfigManager)
		li, err := cm.ListUsers()
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
	//#su
	pro = new(cmd.Program)
	pro.Name = "su"
	pro.Desc = "switch to another logged-in user"
	pro.Usage = "usage: " + pro.Name + " [OPTION]... [UserName]"
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
		user := pro.Target
		if user == "" {
			fmt.Println("user name cannot be empty")
			return
		}
		cm := new(one.ConfigManager)
		err := cm.SwitchUser(user)
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
		user := pro.Target
		if user == "" {
			fmt.Println("user name cannot be empty")
			return
		}
		cm := new(one.ConfigManager)
		err := cm.SaveUser(user)
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
		cm := new(one.ConfigManager)
		userName, err := cm.Who()
		if err != nil {
			fmt.Println("who command call failed, err = ", err)
		} else {
			fmt.Println("current user:", userName)
		}
	}

	//next program
	pro = new(cmd.Program)
	//#search
	pro.Name = "search"
	pro.Desc = "search for files by keywords"
	pro.Usage = "usage: " + pro.Name + " [OPTION]... key"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		Name:      "h",
		LongName:  "help",
		NeedValue: false,
		Desc:      "print help"}
	pro.ParamDefMap["d"] = &cmd.ParamDef{
		Name:      "d",
		LongName:  "detail",
		NeedValue: false,
		Desc:      "show full path of file"}
	pro.ParamDefMap["dn"] = &cmd.ParamDef{
		Name:      "dn",
		LongName:  "download",
		NeedValue: false,
		Desc:      "download file for search result,default save files to search-dn directory,and depend -d flag"}

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
		isCreateDefaultDir := false
		defaultDirName := "search-dn"
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
				if !isCreateDefaultDir {
					os.MkdirAll(defaultDirName, 0770)
					isCreateDefaultDir = true
				}
				pindex := strings.Index(OName, "/root:/")
				if pindex > -1 {
					desc := OName[pindex+6:]
					Download(cli, defaultDirName, desc, false)
				}
			}
		}
	}
	//next program
	pro = new(cmd.Program)
	//#mv
	pro.Name = "mv"
	pro.Desc = "move file to other directory"
	pro.Usage = "usage: " + pro.Name + " [OPTION]... directory"
	pro.ParamDefMap = map[string]*cmd.ParamDef{}

	pro.ParamDefMap["h"] = &cmd.ParamDef{
		Name:      "h",
		LongName:  "help",
		NeedValue: false,
		Desc:      "print help"}
	pro.ParamDefMap["f"] = &cmd.ParamDef{
		Name:      "f",
		LongName:  "file",
		NeedValue: true,
		Desc:      "will move file"}

	pro.ParamDefMap["n"] = &cmd.ParamDef{
		Name:      "n",
		LongName:  "newName",
		NeedValue: true,
		Desc:      "new name"}

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
	isDebug := os.Getenv("oneshowdebug")
	isDebug = strings.TrimSpace(isDebug)
	if isDebug == "true" {
		core.Debug = true
	} else {
		core.Debug = false
	}
	one.InitOneShowConfig()
	ct := cmd.NewContext()
	setFuns(ct)
	ct.Run()
}
