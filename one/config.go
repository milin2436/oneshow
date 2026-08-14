package one

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/milin2436/oneshow/core"
)

// CurUser who is the current user
const CurUser string = ".od_cur_user.id"

// ConfigFileDefault default user when login
const ConfigFileDefault string = ".od.json"

// AppConfigDir config dir
const AppConfigDir = ".config/oneshow"

// OneShowConfigFile config file name
const OneShowConfigFile string = ".oneshow.json"

// AppConfig holds the loaded .oneshow.json application config.
var AppConfig *OneShowConfig

type ConfigManager struct {
}

// GetConfigDir app config dir
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

// InitOneShowConfig load oneshow config information
func InitOneShowConfig() {
	//HOME USER PWD SHELL
	AppConfig = new(OneShowConfig)
	home := GetConfigDir()
	if home != "" {
		fullPath := filepath.Join(home, OneShowConfigFile)
		buff, err := os.ReadFile(fullPath)
		if err != nil {
			return
		}
		err = json.Unmarshal(buff, AppConfig)
		if err != nil {
			fmt.Println("err = ", err)
			return
		}
		//set application config
		setupOneShowConfig()
	}
}

func setupOneShowConfig() {
	cfg := AppConfig
	if cfg.ClientID != "" && cfg.ClientSecret != "" {
		//fmt.Println("using a third-party client :", cfg.ClientID)
		ClientID = cfg.ClientID
		ClientSecret = cfg.ClientSecret
		if cfg.Scope != "" {
			Scope = cfg.Scope
		}
		if cfg.RedirectURL != "" {
			CallbackURL = cfg.RedirectURL
		}
	}
}
func (u *OneClient) getConfigAuthToken() *AuthToken {
	//HOME USER PWD SHELL
	cfg := new(AuthToken)
	content, _ := u.findConfigFile()
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

// SaveTokenToHome home
func (u *OneClient) SaveTokenToHome(token *AuthToken) error {
	home := GetConfigDir()
	pcfg := ""
	if home != "" {
		pcfg = filepath.Join(home, u.ConfigFile)
	} else {
		return errors.New("can not found home dir")
	}
	return SaveTokenToConfig(token, pcfg)
}

// SaveTokenToDefaultPath save config when first login
func SaveTokenToDefaultPath(token *AuthToken) error {
	home := GetConfigDir()
	pcfg := ""
	if home != "" {
		pcfg = filepath.Join(home, ConfigFileDefault)
	} else {
		return errors.New("can not found home dir")
	}
	return SaveTokenToConfig(token, pcfg)
}

// SaveTokenToConfig save to configure file
func SaveTokenToConfig(token *AuthToken, configFile string) error {
	buff, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, buff, 0660)
}

const configFile string = ConfigFileDefault
const configUserFile string = configFile + "."

func (cm *ConfigManager) ListUsers() ([]string, error) {
	home := GetConfigDir()
	return cm.loopDir(home)
}
func (cm *ConfigManager) SaveUser(user string) error {
	home := GetConfigDir()
	userDec := filepath.Join(home, configUserFile+user)
	userSrc := filepath.Join(home, configFile)
	return cm.copyUser(userSrc, userDec)
}
func (cm *ConfigManager) SwitchUser(user string) error {
	home := GetConfigDir()

	userDecFilepath := filepath.Join(home, configUserFile+user)
	if core.ExistFile(userDecFilepath) {
		decFile := filepath.Join(home, CurUser)
		return os.WriteFile(decFile, []byte(user), 0660)
	}
	return fmt.Errorf("the configuration file for the %s user does not exist", user)
}
func (cm *ConfigManager) copyUser(userSrc string, userDec string) error {
	src, err := os.Open(userSrc)
	if err != nil {
		return err
	}
	defer src.Close()
	curFile, err := os.OpenFile(userDec, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0660)
	if err != nil {
		return err
	}
	defer curFile.Close()
	_, err = io.Copy(curFile, src)
	return err
}
func (cm *ConfigManager) Who() (string, error) {
	home := GetConfigDir()
	decFile := filepath.Join(home, CurUser)
	buff, err := os.ReadFile(decFile)
	return string(buff), err
}
func (cm *ConfigManager) loopDir(dirName string) ([]string, error) {
	li := []string{}
	fileList, err := os.ReadDir(dirName)
	if err != nil {
		return nil, err
	}
	for _, f := range fileList {
		info := f
		if f.IsDir() {
			continue
		}
		path := filepath.Join(dirName, info.Name())
		lname := strings.ToLower(info.Name())
		if strings.HasPrefix(lname, configUserFile) {
			li = append(li, path)
		}
	}
	return li, nil
}
