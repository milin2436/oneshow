package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/milin2436/oneshow/cmd"
	"github.com/milin2436/oneshow/core"
	"github.com/milin2436/oneshow/one"
)

func Download(cli *one.OneClient, downloadDir string, dirPath string, a bool) {
	dirPath = one.GetOneDrivePath(dirPath)
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
	pro.Desc = "list OneDrive directory contents"
	pro.Usage = pro.Name + " [OPTION] path"
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
		Desc:      "list files in detail"}
	pro.ParamDefMap["d"] = &cmd.ParamDef{
		Name:      "d",
		LongName:  "direct_url",
		NeedValue: false,
		Desc:      "print a wget command for each file to download it"}

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
				fmt.Printf("%-10s%-16s%-28s%-100s\n", one.FormatSize(v.Size), v.CreatedBy.User.DisplayName, dsTime, Name)
			}
		} else if ct.ParamGroupMap["d"] != nil {
			for _, v := range ret.Value {
				if v.Folder != nil {
					fmt.Printf("%s/\n", v.Name)
					continue
				}
				if v.DownloadURL == "" {
					fmt.Printf("%s\n", v.Name)
					continue
				}
				fmt.Printf("wget -O %q %q\n", v.Name, v.DownloadURL)
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
	pro.Usage = pro.Name + " [OPTION] [file|dir]"
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
	pro.Desc = "show OneDrive account information"
	pro.Usage = pro.Name + " [OPTION] file"
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
		fmt.Printf("%-20s%s\n", "owner", drive.Owner.User.DisplayName)
		fmt.Printf("%-20s%s\n", "total", one.FormatSize(drive.Quota.Total))
		fmt.Printf("%-20s%s\n", "used", one.FormatSize(drive.Quota.Used))
		fmt.Printf("%-20s%s\n", "remaining", one.FormatSize(drive.Quota.Remaining))
		fmt.Printf("%-20s%s\n", "trash", one.FormatSize(drive.Quota.Deleted))
	}

	//next download
	//#d
	pro = new(cmd.Program)
	pro.Name = "d"
	pro.Desc = "download a file, directory, or URL to the local machine"
	pro.Usage = pro.Name + " [OPTION] [file | dir | URL]"
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
		Desc:      "download directory; default is the current directory"}
	pro.ParamDefMap["a"] = &cmd.ParamDef{
		Name:      "a",
		LongName:  "acceleration",
		NeedValue: false,
		Desc:      "speed up downloads via CDN"}

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
			wk.HTTPClient = cli.HTTPClient
			wk.AuthService = cli
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
	pro.Desc = "authorize a new user"
	pro.Usage = pro.Name + " [OPTION]"
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

	//next upload
	pro = new(cmd.Program)
	//#u
	pro.Name = "u"
	pro.Desc = "upload a file or directory to OneDrive"
	pro.Usage = pro.Name + " [OPTION]"
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
		Desc:      "OneDrive destination directory, e.g. /root/path/to"}
	pro.ParamDefMap["s"] = &cmd.ParamDef{
		Name:      "s",
		LongName:  "src",
		NeedValue: true,
		Desc:      "source file or directory on the local machine"}
	pro.ParamDefMap["t"] = &cmd.ParamDef{
		Name:      "t",
		LongName:  "thread",
		NeedValue: true,
		Desc:      "number of upload threads; default 4"}

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
	pro.Desc = "run the HTTP service (beta)"
	pro.Usage = pro.Name + " [OPTION]"
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
		Desc:      "enable HTTPS; requires cacert.pem and privkey.pem in the current directory"}
	pro.ParamDefMap["u"] = &cmd.ParamDef{
		Name:      "u",
		LongName:  "url",
		NeedValue: true,
		Desc:      "listen address for the service, e.g. -u :5555"}

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
		StartWebService(address, https)
	}
	pro = new(cmd.Program)
	//#webdav
	pro.Name = "webdav"
	pro.Desc = "run a WebDAV service for OneDrive (read-only)"
	pro.Usage = pro.Name + " [OPTION]"
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
		Desc:      "listen address, e.g. -u :5555"}
	pro.ParamDefMap["user"] = &cmd.ParamDef{
		Name:      "user",
		LongName:  "user",
		NeedValue: true,
		Desc:      "WebDAV username"}
	pro.ParamDefMap["passwd"] = &cmd.ParamDef{
		Name:      "passwd",
		LongName:  "password",
		NeedValue: true,
		Desc:      "WebDAV password"}
	pro.ParamDefMap["c"] = &cmd.ParamDef{
		Name:      "c",
		LongName:  "cert",
		NeedValue: true,
		Desc:      "HTTPS certificate file"}
	pro.ParamDefMap["k"] = &cmd.ParamDef{
		Name:      "k",
		LongName:  "key",
		NeedValue: true,
		Desc:      "WebDAV private key file"}
	pro.ParamDefMap["ss"] = &cmd.ParamDef{
		Name:      "ss",
		LongName:  "serverlist",
		NeedValue: true,
		Desc:      "semicolon-separated server list, e.g. 0all;all1"}
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
		StartWebDAVService(address, user, passwd, cert, key, ss)
	}
	pro = new(cmd.Program)
	//#users
	pro.Name = "users"
	pro.Desc = "list logged-in users"
	pro.Usage = pro.Name + " [OPTION]"
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
			fmt.Println("run 'saveUser' first to save a session")
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
	pro.Usage = pro.Name + " [OPTION]... [username]"
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
	pro.Desc = "save the current user under a name"
	pro.Usage = pro.Name + " [OPTION]... [username]"
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
	pro.Desc = "show the current username"
	pro.Usage = pro.Name
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
	pro.Desc = "search OneDrive for files by keyword"
	pro.Usage = pro.Name + " [OPTION]... [keyword]"
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
		Desc:      "show the full path of each result"}
	pro.ParamDefMap["dn"] = &cmd.ParamDef{
		Name:      "dn",
		LongName:  "download",
		NeedValue: false,
		Desc:      "download matching files to the search-dn directory (requires -d)"}

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
					if err := os.MkdirAll(defaultDirName, 0770); err != nil {
						fmt.Println("create download directory failed: ", err)
						return
					}
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
	pro.Desc = "move a file to another directory"
	pro.Usage = pro.Name + " [OPTION]... [directory]"
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
		Desc:      "file to move"}

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
		dir = one.GetOneDrivePath(dir)
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
