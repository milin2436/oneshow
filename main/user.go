package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/milin2436/oneshow/one"
)

const configFile string = one.ConfigFileDefault
const configUserFile string = configFile + "."

func ListUsers() ([]string, error) {
	home := one.GetConfigDir()
	return loopDir(home)
}
func SaveUser(user string) error {
	home := one.GetConfigDir()
	userDec := filepath.Join(home, configUserFile+user)
	userSrc := filepath.Join(home, configFile)
	return copyUser(userSrc, userDec)
}
func SwitchUser(user string) error {
	home := one.GetConfigDir()
	decFile := filepath.Join(home, one.CurUser)
	return os.WriteFile(decFile, []byte(user), 0660)
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
	home := one.GetConfigDir()
	decFile := filepath.Join(home, one.CurUser)
	buff, err := os.ReadFile(decFile)
	return string(buff), err
}
func loopDir(dirName string) ([]string, error) {
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
