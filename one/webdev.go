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
	"strings"
	"sync"
	"time"

	"golang.org/x/net/webdav"
)

//OneFileSystem onedrive file system for webdav
type OneFileSystem struct {
	Cache  map[string]*OneFile
	Client *OneClient
}

//OneFile onedrive file for webdav
type OneFile struct {
	Client   *OneClient
	Fs       *OneFileSystem
	FullPath string
	item     *Item
	Position int64
	//Buff       *bytes.Buffer
	RemoteIn   io.ReadCloser
	createTime *time.Time
}

var cacheMutex *sync.RWMutex = &sync.RWMutex{}

//MB byte unit
const MB int = 1048576

//KB byte unit
const KB int = 1024

//DefaultBuffSize default buffer size
const DefaultBuffSize int = 100 * KB

func (fs *OneFileSystem) newOneFileByItem(i *Item, fullPath string) *OneFile {
	of := new(OneFile)
	of.item = i
	of.Client = fs.Client
	of.Fs = fs
	of.FullPath = fullPath
	return of
}

//CacheItem cache item Object
func (fs *OneFileSystem) CacheItem(name string, item *Item) *OneFile {
	fmt.Println("cache item :", name)
	of := fs.newOneFileByItem(item, name)
	cur := time.Now()
	of.createTime = &cur
	cacheMutex.Lock()
	fs.Cache[name] = of
	cacheMutex.Unlock()
	return of
}
func (fs *OneFileSystem) deleteItem(name string) {
	cacheMutex.Lock()
	delete(fs.Cache, name)
	cacheMutex.Unlock()
}

//Copy clone
func (fs *OneFileSystem) Copy(cache *OneFile) *OneFile {
	ret := new(OneFile)
	ret.item = cache.item
	ret.Client = cache.Client
	ret.Fs = cache.Fs
	ret.FullPath = cache.FullPath
	return ret
}
func (fs *OneFileSystem) cacheItemCheckExist(name string, item *Item) *OneFile {
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
	} else {
		//Update the cache more than 40 minutes to ensure that the download address is valid
		if time.Now().Sub(*of.createTime).Minutes() > 40 {
			fmt.Println("update cache, name = ", name)
			fs.deleteItem(name)
			return fs.getFileFromCache(name)
		}
	}
	if of != nil {
		//copy a clone,make position correct
		of = fs.Copy(of)
	}
	return of, nil
}

func isIncludeOp(op int, flag int) bool {
	if (flag & op) == op {
		return true
	}
	return false
}

//Mkdir create a directory
func (fs *OneFileSystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	fmt.Println("mkdir name ", name)
	name = filepath.Clean(name)
	parent := filepath.Dir(name)
	dirName := filepath.Base(name)
	fmt.Println(parent, " = ", dirName)
	_, err := fs.Client.APImkdir(fs.Client.CurDriveID, parent, dirName)
	return err
}

//OpenFile create or write or read file
func (fs *OneFileSystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	fmt.Println("open file", name)
	if isIncludeOp(os.O_CREATE, flag) {
		fmt.Println("flag :create")
		//create
	} else if isIncludeOp(os.O_RDWR, flag) || isIncludeOp(os.O_WRONLY, flag) {
		fmt.Println("flag :write")
		//write
	} else {
		fmt.Println("flag :read")
		//read
		dirPath := getOnedrivePath(name)
		of, err := fs.getFileFromCache(dirPath)
		if of != nil {
			of.Position = 0
		}
		return of, err
	}
	return nil, errors.New("No support")
}

//RemoveAll Move files and directories to the recycle bin
func (fs *OneFileSystem) RemoveAll(ctx context.Context, name string) error {
	fmt.Println("rm name :", name)
	_, err := fs.Client.APIDelFile(fs.Client.CurDriveID, name)
	return err
}

//Rename rename file
func (fs *OneFileSystem) Rename(ctx context.Context, oldName, newName string) error {
	return errors.New("No support Rename")
}

//Stat return information of name
func (fs *OneFileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	fmt.Println("stat file", name)
	dirPath := getOnedrivePath(name)
	return fs.getFileFromCache(dirPath)
}

//write Now no support write
func (of *OneFile) Write(p []byte) (n int, err error) {
	return 0, errors.New("no support write")
}

//Close release resources
func (of *OneFile) Close() error {
	fmt.Println("close", of.Name())
	//close this File
	//release resources
	of.Position = -1
	of.Client = nil
	of.Fs = nil

	of.closeRemoteStream()
	of.RemoteIn = nil
	return nil
}

//Read read content of this file
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
	if of.RemoteIn == nil {
		qkURL := AcceleratedURL(of.item.DownloadURL)
		fmt.Println("qkURL = >", qkURL)
		of.RemoteIn, err = webdavGetFileFromPosition(of.Client.HTTPClient, qkURL, of.Position, of.Size())
		if err != nil {
			fmt.Println("get body err = ", err)
			of.closeRemoteStream()
			return 0, err
		}
	}
	size, err := of.RemoteIn.Read(p)
	if err == nil {
		of.Position = of.Position + int64(size)
		if of.Position >= of.Size() {
			fmt.Println("last check, read file done. name = ", of.Name())
			return size, io.EOF
		}
	}
	return size, err
}

//Seek setup position of this file
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

	of.closeRemoteStream()
	of.RemoteIn = nil
	return of.Position, nil
}

//Readdir Returns all files and directories under the directory
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

//Stat infomation of file
func (of *OneFile) Stat() (os.FileInfo, error) {
	return of, nil
}

//Name name of file
func (of *OneFile) Name() string {
	return of.item.Name
}

//Size file size
func (of *OneFile) Size() int64 {
	return of.item.Size
}

//Mode default for everyone
func (of *OneFile) Mode() os.FileMode {
	return 0777
}

//ModTime return modified time
func (of *OneFile) ModTime() time.Time {
	mdTime := time.Time(of.item.LastModifiedDateTime)
	//dsTime := mdTime.Local()
	return mdTime
}

//IsDir whether this oneFile is a directory
func (of *OneFile) IsDir() bool {
	fmt.Println("isDir", of.Name())
	if of.item.Folder != nil {
		return true
	}
	return false
}

//Sys return nil one onedrive
func (of *OneFile) Sys() interface{} {
	return nil
}

func (of *OneFile) getRightBufferSize() int {
	lname := strings.ToLower(of.Name())
	//video file
	if strings.HasSuffix(lname, ".mp4") ||
		strings.HasSuffix(lname, ".mkv") ||
		strings.HasSuffix(lname, ".wmv") ||
		strings.HasSuffix(lname, ".webm") ||
		strings.HasSuffix(lname, ".avi") ||
		strings.HasSuffix(lname, ".rmvb") ||
		strings.HasSuffix(lname, ".rm") {
		return 25 * MB
	}
	// 1G+
	if int64(1024*MB) < of.Size() {
		return 25 * MB
	}

	if of.Size() > int64(200*MB) && of.Size() <= int64(1024*MB) {
		return 10 * MB
	}
	return MB
}
func (of *OneFile) getBuff(size int) *bytes.Buffer {
	buff := new(bytes.Buffer)
	buff.Grow(size)
	return buff
}

func (of *OneFile) closeRemoteStream() {
	if of.RemoteIn != nil {
		fmt.Println("close Remote inputsteam,name = ", of.Name())
		of.RemoteIn.Close()
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

func AcceleratedURL(hurl string) string {
	if !ONE_SHOW_CONFIG.Acceleration {
		return hurl
	}
	if ONE_SHOW_CONFIG.AcceleratedAPI == "" {
		return hurl
	}
	p := url.QueryEscape(hurl)
	return ONE_SHOW_CONFIG.AcceleratedAPI + p
}
