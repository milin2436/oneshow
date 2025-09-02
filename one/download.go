package one

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	chttp "github.com/milin2436/oneshow/http"
)

//AuthService add auth for download service
type AuthService interface {
	GetTokenHeader() map[string]string
}

//DownloadInfo show download info
type DownloadInfo struct {
	URL             string
	FileName        string
	CurPosition     int64
	Size            int64
	Desc            string
	Del             int
	Rate            int64
	LastUpdatedTime time.Time
}

//DWorker download class
type DWorker struct {
	HTTPCli     *chttp.HttpClient
	CurDownload *DownloadInfo
	TaskCtl     *ThreadControl
	//0 wait ; 1 downloading ; 2 cancel ; 3 error ; 4 done
	Status      int
	WorkStatus  string
	cancelFlag  bool //cancel this download
	WorkerID    int
	AuthSve     AuthService
	DownloadDir string
	Proxy       bool
	Error       error
	dm          *DownloadManager
}

type ThreadControl struct {
	Cancel   context.Context
	CancelFn context.CancelFunc
}

//DownloadManager manager download task
type DownloadManager struct {
	startID       int
	rootContext   context.Context
	taskQueue     chan *DWorker
	closeFlag     chan int
	dispatchQueue chan int
	max           int
	activeCnt     int
	data          map[string]*DWorker
	dataLock      sync.RWMutex
}

//GetDownloadFileName get name of download source
func GetDownloadFileName(u *url.URL, fileName string, disposition string) string {
	if disposition != "" && strings.Contains(disposition, "filename") {
		_, params, err := mime.ParseMediaType(disposition)
		if err == nil {
			return params["filename"]
		}
	}
	fn := strings.TrimSpace(fileName)
	if fn != "" {
		return fn
	}
	idx := strings.LastIndex(u.Path, "/")
	if idx > -1 {
		if u.Path == "/" {
			return "index.html"
		}
		return u.Path[idx+1:]
	}
	return u.Path
}

func parseRangeCookie(strConRge string) (int64, error) {
	//Content-Range:[bytes 0-1/707017362]
	idx := strings.Index(strConRge, "/")
	if idx == -1 {
		return 0, errors.New("format error in " + strConRge)
	}
	strSize := strConRge[idx+1:]
	ret, err := strconv.ParseInt(strSize, 10, 64)
	return ret, err
}
func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}
func recordFilePosion(f *os.File, position int64) error {
	_, err := f.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}
	//8*8
	part := make([]byte, 8)
	for i := 0; i < 8; i++ {
		part[i] = byte(position >> uint((7-i)*8) & 0xFF)
	}
	_, err = f.Write(part)
	return err
}
func readFilePosion(f *os.File) (int64, error) {
	_, err := f.Seek(0, os.SEEK_SET)
	if err != nil {
		return 0, err
	}
	part := make([]byte, 8)
	_, err = io.ReadFull(f, part)
	if err != nil {
		return 0, err
	}
	var ret, tmp int64
	for i := 0; i < 8; i++ {
		tmp = int64(part[i]) << uint((7-i)*8)
		ret = tmp + ret
	}
	return ret, nil
}
func (wk *DWorker) downloadNoRange(url string, fileName string) error {
	//set full path
	fileFullPath := filepath.Join(wk.DownloadDir, fileName)
	headers := map[string]string{}
	wk.addAutoHTTPHeader(headers)
	resp, err := wk.HTTPCli.HttpGet(url, headers, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	df, err := os.OpenFile(fileFullPath, os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	defer df.Close()
	var len int
	buffer := make([]byte, 1024)
	for {
		if wk.TaskCtl != nil {
			select {
			case <-wk.TaskCtl.Cancel.Done():
				wk.Status = 2
				return nil
			default:
			}
		}
		len, err = resp.Body.Read(buffer)
		if err != nil {
			if err == io.EOF {
				return nil
			} else {
				return err
			}
		} else {
			_, err = df.Write(buffer[:len])
			if err != nil {
				return err
			}
		}
	}
}
func (wk *DWorker) doDownload4NoRange(url string, fileName string) error {
	err := wk.downloadNoRange(url, fileName)
	if err != nil {
		wk.Error = err
		wk.Status = 3
		return err
	}
	if wk.Status == 1 {
		wk.Status = 4
	}
	return nil
}
func (wk *DWorker) Download(url string) error {
	if wk.dm != nil {
		defer wk.dm.DispatchNotify(wk.WorkerID)
	}
	if wk.HTTPCli == nil {
		return errors.New("pls http client for this worker")
	}
	if wk.Proxy {
		url = getAcceleratedURL(url)
	}
	wk.cancelFlag = false
	wk.WorkStatus = fmt.Sprintf("downloading id = %d for %s\n", wk.WorkerID, url)

	fmt.Println("url = > ", url)
	fileName, fileSize, isRange, err := wk.GetDownloadFileInfo(url, "")
	fmt.Println("fn", fileName, " val = ", isRange)
	if !isRange {
		return wk.doDownload4NoRange(url, fileName)
	}
	if err != nil {
		wk.Error = err
		wk.Status = 3
		return err
	}
	//set full path
	fileName = filepath.Join(wk.DownloadDir, fileName)
	//TODO

	rdFile := fileName + ".finfo"
	curPosion := int64(0)
	finish := false
	var dfile, tfile *os.File
	if PathExists(fileName) && PathExists(rdFile) {
		log.Println("go on downloadfile file ", fileName)
		dfile, err = os.OpenFile(fileName, os.O_RDWR, 0660)
		if err != nil {
			return err
		}
		defer dfile.Close()
		tfile, err = os.OpenFile(rdFile, os.O_RDWR, 0700)
		if err != nil {
			return err
		}
		curPosion, err = readFilePosion(tfile)
		if err != nil {
			return err
		}
		fmt.Printf("downloaded %d bytes (%s) for %s", curPosion, ViewHumanShow(curPosion), fileName)
	} else if PathExists(fileName) && (!PathExists(rdFile)) {
		//TODO nothing
		log.Println("nothing for " + fileName)
		return nil
	} else {
		//a new download task
		log.Println("start a new download task")
		dfile, err = os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0660)
		if err != nil {
			return err
		}
		if err := dfile.Truncate(fileSize); err != nil {
			return err
		}
		defer dfile.Close()

		tfile, err = os.OpenFile(rdFile, os.O_CREATE|os.O_RDWR, 0777)
		if err != nil {
			return err
		}
		err = recordFilePosion(tfile, curPosion)
		if err != nil {
			return err
		}
	}
	cnt := 1
	maxTryCount := 10
	wk.CurDownload.URL = url
	wk.CurDownload.FileName = fileName
	wk.CurDownload.CurPosition = curPosion
	wk.CurDownload.Size = fileSize
	log.Println("start ==>", url)
	log.Println("File name ==>", fileName)
	for {
		err = wk.execResumableDownload(url, curPosion, fileSize, dfile, tfile)
		if err == nil && wk.cancelFlag {
			finish = false
			wk.Status = 2
			break
		}

		if err == nil {
			finish = true
			wk.Status = 4
			break
		}
		log.Println("a error = ", err, " start new http connect....,try again ", cnt)
		realCurPosion, err := readFilePosion(tfile)
		if err != nil {
			log.Println("read position to failed,finish this task")
			return err
		}
		if realCurPosion > curPosion {
			cnt = 1
		} else {
			//No new data was downloaded, indicating consecutive error.
			cnt++
			if cnt >= maxTryCount {
				wk.Error = err
				wk.Status = 3
				break
			}
		}
		curPosion = realCurPosion
		time.Sleep(time.Second * time.Duration(cnt))
		log.Println("reset position ", curPosion, " start new http connct...")
	}
	//release resource
	if tfile != nil {
		tfile.Close()
	}
	//remove position file
	if finish {
		os.Remove(rdFile)
		log.Println("done ==>", fileName)
	}
	return nil
}
func (wk *DWorker) addAutoHTTPHeader(header map[string]string) {
	if wk.AuthSve != nil {
		authHTTPHeader := wk.AuthSve.GetTokenHeader()
		for k, v := range authHTTPHeader {
			header[k] = v
		}
	}
}
func (wk *DWorker) GetDownloadFileInfo(uurl string, fileName string) (string, int64, bool, error) {
	//RANGE: bytes=0-1

	header := map[string]string{}
	header["RANGE"] = "bytes=0-1"
	header["Accept"] = "*/*"

	resp, err := wk.HTTPCli.HttpGet(uurl, header, nil)
	if err != nil {
		return "", 0, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", 0, false, fmt.Errorf("status code error : %d", resp.StatusCode)
	}
	realName := GetDownloadFileName(resp.Request.URL, fileName, resp.Header.Get("Content-Disposition"))

	strConRge := resp.Header.Get("Content-Range")
	if strConRge == "" {
		return realName, 0, false, nil
	}
	fileSize, err := parseRangeCookie(strConRge)
	log.Println("response header ", strConRge, " fileSize = ", ViewHumanShow(fileSize))
	if err != nil {
		return "", 0, true, err
	}
	return realName, fileSize, true, nil
}
func (wk *DWorker) execResumableDownload(durl string, position int64, fileSize int64, dfile *os.File, tfile *os.File) error {
	rangeHeader := fmt.Sprintf("bytes=%d-", position)
	log.Println("header range :", rangeHeader)
	header := map[string]string{}
	header["RANGE"] = rangeHeader
	//Remove authentication information from the request headers,
	//as it is already included in the URL,
	//to resolve the 401 error for OneDrive personal accounts

	//wk.addAutoHTTPHeader(header)
	resp, err := wk.HTTPCli.HttpGet(durl, header, nil)
	if err != nil {
		return errors.New(fmt.Sprint("download ", durl, " failed", err))
	}
	//Release HTTP resources
	defer resp.Body.Close()
	strConRge := resp.Header.Get("Content-Range")
	if strConRge == "" {
		return errors.New("no support range")
	}
	sc := resp.StatusCode / 100
	if sc != 2 {
		return errors.New("request errors,status code = " + strconv.Itoa(sc) + "," + resp.Status)
	}
	log.Println("response header range :", strConRge)
	log.Println("seed to :", position)
	_, err = dfile.Seek(position, os.SEEK_SET)
	if err != nil {
		return errors.New("seek error in " + err.Error())
	}
	buff := make([]byte, 102400)
	//1M 1k*1024
	readCnt := 0
	wk.CurDownload.LastUpdatedTime = time.Now()
	for {
		if wk.TaskCtl != nil {
			select {
			case <-wk.TaskCtl.Cancel.Done():
				fmt.Println("download task exiting...")
				wk.cancelFlag = true
				return nil
			default:
			}
		}
		count, err := resp.Body.Read(buff)
		if err != nil && err != io.EOF {
			//An I/O error occurred from http server
			return err
		}
		position = position + int64(count)

		_, wriErr := dfile.Write(buff[0:count])
		if wriErr != nil {
			//An I/O error occurred for save data to local file.
			return wriErr
		}
		if err == io.EOF || position >= fileSize {
			if position == fileSize {
				log.Println("Finish file download. File size ", fileSize)
			} else {
				log.Printf("The downloaded file size(%d) does not match the actual size(%d)\n", position, fileSize)
			}
			break
		}

		readCnt++
		if readCnt > 5 {
			t1 := time.Now()
			dis := t1.Sub(wk.CurDownload.LastUpdatedTime)
			readCnt = 0
			if dis.Seconds() >= 1 {
				err = recordFilePosion(tfile, position)
				if err != nil {
					log.Println("write postion to failed, err = ", err)
				}
				//dis := t1.Sub(t0)
				addData := position - wk.CurDownload.CurPosition
				v := addData / dis.Milliseconds() * 1000

				slog := fmt.Sprintf("download rate = %s/s,finish %s %s $ in %f s", ViewHumanShow(v), ViewHumanShow(position), ViewPercent(position, fileSize), dis.Seconds())
				wk.CurDownload.CurPosition = position
				wk.CurDownload.Desc = slog
				wk.CurDownload.Rate = v
				wk.CurDownload.LastUpdatedTime = t1
				log.Println(slog)
			}
		}
	}
	return nil
}

//NewDWorker create a download worker
func NewDWorker() *DWorker {
	wk := new(DWorker)
	wk.CurDownload = new(DownloadInfo)
	return wk
}

func webdavGetFileFromPosition(cli *chttp.HttpClient, uurl string, position int64, fileSize int64) (io.ReadCloser, error) {
	rangeHeader := fmt.Sprintf("bytes=%d-", position)
	fmt.Println("header range :", rangeHeader)
	header := map[string]string{}
	header["RANGE"] = rangeHeader
	resp, err := cli.HttpGet(uurl, header, nil)
	if err != nil {
		return nil, errors.New(fmt.Sprint("download ", uurl, " failed", err))
	}
	//defer resp.Body.Close()
	strConRge := resp.Header.Get("Content-Range")
	if strConRge == "" {
		return nil, errors.New("no support range")
	}
	sc := resp.StatusCode / 100
	if sc != 2 {
		return nil, errors.New("request errors,status code = " + strconv.Itoa(sc) + "," + resp.Status)
	}
	fmt.Println("response header range :", strConRge)
	fmt.Println("seed to :", position)
	//1M 1k*1024
	return resp.Body, nil
}

func (cli *OneClient) BatchDownload(curDir string, descDir string, a bool) {
	fileList, err := cli.APIListFilesByPath(cli.CurDriveID, curDir)
	if err != nil {
		fmt.Println("error in loop dir,err = ", err)
		return
	}
	for _, f := range fileList.Value {
		if f.Folder != nil {
			cli.BatchDownload(filepath.Join(curDir, f.Name), filepath.Join(descDir, f.Name), a)
			continue
		}
		path := filepath.Join(curDir, f.Name)
		fmt.Println("donwloading  ", path)
		err = os.MkdirAll(descDir, 0771)
		if err != nil {
			fmt.Println("create dir to failed ", err)
			break
		}
		localFilePath := filepath.Join(descDir, f.Name)
		localFilePathTmp := filepath.Join(descDir, f.Name+".finfo")
		if PathExists(localFilePath) && !PathExists(localFilePathTmp) {
			fmt.Println("The file exists,skip it : ", localFilePath)
			continue
		}
		cli.Download(path, descDir, a)
	}
}

func NewDM() *DownloadManager {
	dm := new(DownloadManager)
	dm.rootContext = context.Background()
	dm.startID = 0
	dm.taskQueue = make(chan *DWorker, 1024)
	dm.closeFlag = make(chan int)
	dm.dispatchQueue = make(chan int, 20)
	dm.data = map[string]*DWorker{}

	dm.max = 4
	return dm
}

func (dm *DownloadManager) AddTask(httpCli *chttp.HttpClient, durl string, downloadDir string, a bool) {
	wk := NewDWorker()
	wk.HTTPCli = httpCli
	wk.DownloadDir = downloadDir
	wk.Proxy = a

	wk.CurDownload.URL = durl
	dm.taskQueue <- wk
}

func (dm *DownloadManager) GetAllTask() []*DWorker {
	li := []*DWorker{}
	dm.dataLock.RLock()
	for _, t := range dm.data {
		li = append(li, t)
	}
	dm.dataLock.RUnlock()
	return li
}

func (dm *DownloadManager) CandelTask(id int) {
	var task *DWorker = nil
	dm.dataLock.RLock()
	for _, t := range dm.data {
		if t.WorkerID == id {
			task = t
			break
		}
	}
	dm.dataLock.RUnlock()

	//TODO task lock
	if task != nil {
		if task.Status == 0 {
			task.Status = 2
		} else if task.Status == 1 {
			//running
			task.TaskCtl.CancelFn()
		}
	}
	dm.DispatchNotify(-1)
}

func (dm *DownloadManager) RestartTask(id int) {
	var task *DWorker = nil
	dm.dataLock.RLock()
	for _, t := range dm.data {
		if t.WorkerID == id {
			task = t
			break
		}
	}
	dm.dataLock.RUnlock()

	//TODO task lock
	if task != nil {
		if task.Status == 2 {
			c, fn := context.WithCancel(dm.rootContext)
			task.TaskCtl.Cancel = c
			task.TaskCtl.CancelFn = fn
			task.Status = 0
		}
	}
	dm.DispatchNotify(-1)
}

func (dm *DownloadManager) Close() {
	dm.closeFlag <- 0
}

func (dm *DownloadManager) DispatchNotify(id int) {
	dm.dispatchQueue <- id
}
func (dm *DownloadManager) release(flag int) {
	fmt.Println("close DM ...")
	if flag > 0 {
		dm.dataLock.RLock()
		for _, t := range dm.data {
			if t.Status == 1 {
				t.TaskCtl.CancelFn()
			}
		}
		dm.dataLock.RUnlock()
	}
}

func (dm *DownloadManager) addTask2Datamap(task *DWorker) {
	fmt.Println("add task", task)
	dm.startID++
	task.WorkerID = dm.startID
	c, fn := context.WithCancel(dm.rootContext)
	task.TaskCtl = new(ThreadControl)
	task.TaskCtl.Cancel = c
	task.TaskCtl.CancelFn = fn
	task.dm = dm

	dm.dataLock.Lock()
	dm.data[task.CurDownload.URL] = task
	dm.dataLock.Unlock()
}

func (dm *DownloadManager) getNextTask() *DWorker {
	if len(dm.data) == 0 {
		return nil
	}
	for _, t := range dm.data {
		if t.Status == 0 {
			return t
		}
	}
	return nil
}

// workerID is finished ID of task
func (dm *DownloadManager) dispatch(workerID int) {
	if workerID > -1 {
		fmt.Println("task over =>", workerID)
		dm.activeCnt--
	}
	if dm.activeCnt < dm.max {
		task := dm.getNextTask()
		if task == nil {
			fmt.Println("no waitting download task")
			return
		}
		task.Status = 1
		go task.Download(task.CurDownload.URL)
		dm.activeCnt++
	} else {
		fmt.Println("no free worker thread")
	}
}

func (dm *DownloadManager) Start() {
	for {
		select {
		case task := <-dm.taskQueue:
			dm.addTask2Datamap(task)
			dm.dispatch(-1)
		case val := <-dm.closeFlag:
			dm.release(val)
			return
		case workID := <-dm.dispatchQueue:
			dm.dispatch(workID)
		}
	}
}
