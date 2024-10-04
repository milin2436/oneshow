package one

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/milin2436/oneshow/core"
)

//CurUser who is the current user
const CurUser string = ".od_cur_user.id"

//ConfigFileDefault default user when login
const ConfigFileDefault string = ".od.json"

//AppConfigDir config dir
const AppConfigDir = ".config/oneshow"

//OneshowConfigFile config file name
const OneshowConfigFile string = ".oneshow.json"

//OneshowConfig load .oneshow.json config file
var OneshowConfig *OneShowConfig

//GetConfigDir app config dir
func GetConfigDir() string {
	home, _ := os.UserHomeDir()
	configDir := AppConfigDir

	ret := filepath.Join(home, configDir)
	if core.ExistFile(ret) {
		return ret
	}
	os.MkdirAll(ret, os.ModePerm)
	return ret
}

func getCurUser() string {
	envUser := os.Getenv("oneshowuser")
	envUser = strings.TrimSpace(envUser)
	home := GetConfigDir()
	user := ""
	if envUser != "" {
		user = envUser
		fmt.Println("user envUser :", user)
	} else {
		buff, err := os.ReadFile(filepath.Join(home, CurUser))
		if err != nil {
			user = ""
		} else {
			userName := string(buff)
			userName = strings.TrimSpace(userName)
			user = userName
		}
	}
	//fmt.Println("using config = ", user)
	return user
}
func (u *OneClient) setUserInfo(name string) {
	u.UserName = name
	if name == "" {
		u.ConfigFile = ConfigFileDefault
	} else {
		u.ConfigFile = ConfigFileDefault + "." + name
	}
}
func (u *OneClient) findConfigFile() (string, error) {
	home := GetConfigDir()
	buff, err := os.ReadFile(filepath.Join(home, u.ConfigFile))
	if err != nil {
		return "", err
	}
	return string(buff), nil
}

//InitOneShowConfig load oneshow config information
func InitOneShowConfig() {
	//HOME USER PWD SHELL
	OneshowConfig = new(OneShowConfig)
	home := GetConfigDir()
	if home != "" {
		fullPath := filepath.Join(home, OneshowConfigFile)
		buff, err := os.ReadFile(fullPath)
		if err != nil {
			return
		}
		err = json.Unmarshal(buff, OneshowConfig)
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
		//set application config
		setupOneShowConfig()
	}
}

func setupOneShowConfig() {
	cfg := OneshowConfig
	if cfg.Client_ID != "" && cfg.ClientSecret != "" {
		//fmt.Println("using a third-party client :", cfg.Client_ID)
		CLIENT_ID = cfg.Client_ID
		CLIENT_SECRET = cfg.ClientSecret
		if cfg.Scope != "" {
			SCOPE = cfg.Scope
		}
		if cfg.RedirectURL != "" {
			CALLBACK_URL = cfg.RedirectURL
		}
	}
}
func (u *OneClient) getConfigAuthToken() *AuthToken {
	//HOME USER PWD SHELL
	cfg := new(AuthToken)
	content, err := u.findConfigFile()
	//fmt.Println(content)
	if content == "" {
		fmt.Println("can not find config file")
		return nil
	}
	err = json.Unmarshal([]byte(content), cfg)
	if err != nil {
		fmt.Println("err = ", err)
		return nil
	}
	return cfg
}

//SaveToken2Home home
func (u *OneClient) SaveToken2Home(token *AuthToken) error {
	home := GetConfigDir()
	pcfg := ""
	if home != "" {
		pcfg = filepath.Join(home, u.ConfigFile)
	} else {
		return errors.New("can not found home dir")
	}
	return SaveToken2Config(token, pcfg)
}

//SaveToken2DefaultPath save config when first login
func SaveToken2DefaultPath(token *AuthToken) error {
	home := GetConfigDir()
	pcfg := ""
	if home != "" {
		pcfg = filepath.Join(home, ConfigFileDefault)
	} else {
		return errors.New("can not found home dir")
	}
	return SaveToken2Config(token, pcfg)
}

//SaveToken2Config save to configure file
func SaveToken2Config(token *AuthToken, configFile string) error {
	buff, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, buff, 0660)
}
