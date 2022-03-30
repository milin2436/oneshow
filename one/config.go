package one

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

//CurUser who is the current user
const CurUser string = ".od_cur_user.id"

//ConfigFileDefault default user when login
const ConfigFileDefault string = ".od.json"

const one_show_config_file string = ".oneshow.json"

var ONE_SHOW_CONFIG *OneShowConfig

var configFile string = ".od.json"

func setCurUser() {
	suser := os.Getenv("oneshowuser")
	home, _ := os.UserHomeDir()
	buff, err := ioutil.ReadFile(filepath.Join(home, CurUser))
	if err != nil {
		configFile = ConfigFileDefault
	} else {
		userName := string(buff)
		userName = strings.TrimSpace(userName)
		configFile = ConfigFileDefault + "." + userName
	}
	suser = strings.TrimSpace(suser)
	if suser != "" {
		configFile = ConfigFileDefault + "." + suser
		fmt.Println("using config = ", configFile)
	}
	//fmt.Println("using config = ", configFile)
}
func findConfigFile() string {
	buff, err := ioutil.ReadFile(configFile)
	if err != nil {
		home, _ := os.UserHomeDir()
		if home != "" {
			fullPath := filepath.Join(home, configFile)
			buff, err = ioutil.ReadFile(fullPath)
			if err == nil {
				return string(buff)
			}
		}
		return ""
	}
	return string(buff)
}

//InitOneShowConfig load oneshow config information
func InitOneShowConfig() {
	//HOME USER PWD SHELL
	ONE_SHOW_CONFIG = new(OneShowConfig)
	home, _ := os.UserHomeDir()
	if home != "" {
		fullPath := filepath.Join(home, one_show_config_file)
		buff, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return
		}
		err = json.Unmarshal(buff, ONE_SHOW_CONFIG)
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
		//set application config
		setupOneShowConfig()
	}
}

func setupOneShowConfig() {
	cfg := ONE_SHOW_CONFIG
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
func getConfigAuthToken() *AuthToken {
	//HOME USER PWD SHELL
	cfg := new(AuthToken)
	content := findConfigFile()
	//fmt.Println(content)
	if content == "" {
		fmt.Println("can not find config file")
		return nil
	}
	err := json.Unmarshal([]byte(content), cfg)
	if err != nil {
		fmt.Println("err = ", err)
		return nil
	}
	return cfg
}

//SaveToken2Home home
func SaveToken2Home(token *AuthToken) error {
	home, _ := os.UserHomeDir()
	pcfg := ""
	if home != "" {
		pcfg = filepath.Join(home, configFile)
	} else {
		return errors.New("can not found home dir")
	}
	return SaveToken2Config(token, pcfg)
}

//SaveToken2DefaultPath save config when first login
func SaveToken2DefaultPath(token *AuthToken) error {
	home, _ := os.UserHomeDir()
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
	return ioutil.WriteFile(configFile, buff, 0660)
}
