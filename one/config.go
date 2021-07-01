package one

import (
	"encoding/json"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

const configFile = ".od.json"

func findConfigFile() string {
	buff, err := ioutil.ReadFile(configFile)
	if err != nil {
		home := os.Getenv("HOME")
		if home != "" {
			buff, err = ioutil.ReadFile(home + "/" + configFile)
			if err == nil {
				return string(buff)
			}
		}
		return ""
	}
	return string(buff)
}
func getConfigAuthToken() *AuthToken {
	//HOME USER PWD SHELL
	cfg := new(AuthToken)
	content := findConfigFile()
	//fmt.Println(content)
	if content == "" {
		log.Println("can not find config file")
		return nil
	}
	err := json.Unmarshal([]byte(content), cfg)
	if err != nil {
		log.Println("err = ", err)
		return nil
	}
	return cfg
}

//SaveToken2Home home
func SaveToken2Home(token *AuthToken) error {
	home := os.Getenv("HOME")
	pcfg := home + "/" + configFile
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
