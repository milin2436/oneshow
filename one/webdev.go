package one

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/net/webdav"
)

type OneFileSystem struct {
	Cache  map[string]*OneFile
	Client *OneClient
}
type OneFile struct {
	Client   *OneClient
	Fs       *OneFileSystem
	FullPath string
	item     *Item
	Position int64
	Buff     *bytes.Buffer
}

func (fs *OneFileSystem) newOneFileByItem(i *Item, fullPath string) *OneFile {
	of := new(OneFile)
	of.item = i
	of.Client = fs.Client
	of.Fs = fs
	of.FullPath = fullPath
	return of
}

func (fs *OneFileSystem) CacheItem(name string, item *Item) *OneFile {
	fmt.Println("cache item :", name)
	of := fs.newOneFileByItem(item, name)
	fs.Cache[name] = of
	return of
}
func (fs *OneFileSystem) Copy(cache *OneFile) *OneFile {
	ret := new(OneFile)
	ret.item = cache.item
	ret.Client = cache.Client
	ret.Fs = cache.Fs
	ret.FullPath = cache.FullPath
	return ret
}
func (fs *OneFileSystem) CacheItemCheckExist(name string, item *Item) *OneFile {
	of := fs.Cache[name]
	if of == nil {
		return fs.CacheItem(name, item)
	}
	return of
}
func (fs *OneFileSystem) getFileFromCache(name string) (*OneFile, error) {
	fmt.Println("getFileFromCache :", name)
	of := fs.Cache[name]
	if of == nil {
		info, err := fs.Client.APIGetFile(fs.Client.CurDriveID, name)
		fmt.Println("err = ", err)
		if err != nil {
			return nil, err
		}
		//cache
		of = fs.CacheItem(name, info)
	}
	if of != nil {
		//TODO copy a clone
		of = fs.Copy(of)
	}
	return of, nil
}

func (fs *OneFileSystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return errors.New("No support Mkdir")
}

func (fs *OneFileSystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	fmt.Println("open file", name)
	dirPath := getOnedrivePath(name)
	of, err := fs.getFileFromCache(dirPath)
	if of != nil {
		of.Position = 0
	}
	return of, err
}

func (fs *OneFileSystem) RemoveAll(ctx context.Context, name string) error {
	return errors.New("No support RemoveAll")
}

func (fs *OneFileSystem) Rename(ctx context.Context, oldName, newName string) error {
	return errors.New("No support Rename")
}

func (fs *OneFileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	fmt.Println("stat file", name)
	dirPath := getOnedrivePath(name)
	return fs.getFileFromCache(dirPath)
}

func (of *OneFile) Write(p []byte) (n int, err error) {
	return 0, errors.New("no support write")
}

func (of *OneFile) Close() error {
	fmt.Println("close", of.Name())
	//close this File
	//release resources
	of.Position = -1
	of.Client = nil
	of.Fs = nil
	if of.Buff != nil {
		of.Buff = nil
	}

	if of.IsDir() {
		return nil
	}
	return nil
}

func (of *OneFile) Read(p []byte) (n int, err error) {
	fmt.Println("read", of.Name(), " position = ", of.Position)
	fmt.Println("framework buff len:", len(p))
	if of.IsDir() {
		fmt.Println("can not read dir", of.Name())
		return 0, errors.New("no support write")
	}
	if of.Position >= of.Size() {
		fmt.Println("first check, read file done. name = ", of.Name())
		return 0, io.EOF
	}

	of.InitBuff()

	//check Buff
	if of.Buff.Len() == 0 {
		qkURL := acceleratedURL(of.item.DownloadURL)
		fmt.Println("qkURL = >", qkURL)
		_, err = webdavGetFileCotent(of.Client.HTTPClient, of.Buff, qkURL, of.Position, of.Size())
		//TODO only print
		if err != nil {
			fmt.Println("call webdavGetFileCotent err = ", err)
		}
	}

	//read data from buff
	size, err := of.Buff.Read(p)
	if err == nil {
		of.Position = of.Position + int64(size)
		if of.Position >= of.Size() {
			fmt.Println("last check, read file done. name = ", of.Name())
			return size, io.EOF
		}
	}
	return size, err
}

func (of *OneFile) Seek(offset int64, whence int) (int64, error) {
	fmt.Println("seek", of.Name())
	fmt.Println("offset", offset)
	fmt.Println("whence:", whence)
	if os.SEEK_SET == whence {
		of.Position = offset
	}
	if os.SEEK_CUR == whence {
		of.Position = of.Position + offset
	}
	if os.SEEK_END == whence {
		of.Position = of.Size() - offset
	}
	//TODO check postion
	of.ResetBuff()
	return of.Position, nil
}

func (of *OneFile) Readdir(count int) ([]os.FileInfo, error) {
	fmt.Println("call readdir:", of.Name())
	if of.IsDir() {
		ret, err := of.Client.APIListFilesByPath(of.Client.CurDriveID, of.FullPath)
		if err != nil {
			fmt.Println("call APIListFilesByPath err = ", err)
			return nil, err
		}
		li := []os.FileInfo{}
		lLen := len(ret.Value)
		for i := 0; i < lLen; i++ {
			pitem := &ret.Value[i]
			sfp := filepath.Join(of.FullPath, pitem.Name)
			subOf := of.Fs.CacheItem(sfp, pitem)
			li = append(li, subOf)
		}
		return li, nil

	}
	return nil, errors.New("this is file :" + of.Name())
}

func (of *OneFile) Stat() (os.FileInfo, error) {
	return of, nil
}

//os.FileInfo
func (of *OneFile) Name() string {
	return of.item.Name
}
func (of *OneFile) Size() int64 {
	return of.item.Size
}
func (of *OneFile) Mode() os.FileMode {
	return 0777
}
func (of *OneFile) ModTime() time.Time {
	mdTime := time.Time(of.item.LastModifiedDateTime)
	//dsTime := mdTime.Local()
	return mdTime
}
func (of *OneFile) IsDir() bool {
	fmt.Println("isDir", of.Name())
	if of.item.Folder != nil {
		return true
	}
	return false
}
func (of *OneFile) Sys() interface{} {
	return nil
}
func (of *OneFile) InitBuff() {
	if of.Buff == nil {
		of.Buff = new(bytes.Buffer)
		//4M
		of.Buff.Grow(4194304)
	}
}

//when position change
func (of *OneFile) ResetBuff() {
	if of.Buff != nil {
		of.Buff.Reset()
	}
}

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

func acceleratedURL(hurl string) string {
	if ONE_SHOW_CONFIG.AcceleratedAPI == "" {
		return hurl
	}
	p := url.QueryEscape(hurl)
	return ONE_SHOW_CONFIG.AcceleratedAPI + p
}
