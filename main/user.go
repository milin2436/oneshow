package main

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/milin2436/oneshow/one"
)

const config string = ".od.json."
const configFile string = ".od.json"

func ListUsers() ([]string, error) {
	home, _ := os.UserHomeDir()
	return loopDir(home)
}
func SaveUser(user string) error {
	home, _ := os.UserHomeDir()
	userDec := filepath.Join(home, config+user)
	userSrc := filepath.Join(home, configFile)
	return copyUser(userSrc, userDec)
}
func SwitchUser(user string) error {
	home, _ := os.UserHomeDir()
	decFile := filepath.Join(home, one.CurUser)
	return ioutil.WriteFile(decFile, []byte(user), 0660)
}
func copyUser(userSrc string, userDec string) error {
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
func Who() (string, error) {
	home, _ := os.UserHomeDir()
	decFile := filepath.Join(home, one.CurUser)
	buff, err := ioutil.ReadFile(decFile)
	return string(buff), err
}
func loopDir(dirName string) ([]string, error) {
	li := []string{}
	fileList, err := ioutil.ReadDir(dirName)
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
		if strings.HasPrefix(lname, config) {
			li = append(li, path)
		}
	}
	return li, nil
}
