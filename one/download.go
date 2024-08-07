package one

import (
	"bytes"
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
	"time"

	chttp "github.com/milin2436/oneshow/http"
)

//AuthService add auth for download service
type AuthService interface {
	GetTokenHeader() map[string]string
}

//DownloadInfo show download info
type DownloadInfo struct {
	URL         string
	FileName    string
	CurPosition int64
	Size        int64
	Desc        string
	Del         int
}

//DWorker download class
type DWorker struct {
	HTTPCli     *chttp.HttpClient
	CurDownload *DownloadInfo
	WorkStatus  string
	cancelFlag  bool //cancel this download
	suspendFlag bool //supend this download
	WorkerID    int
	AuthSve     AuthService
	DownloadDir string
	Proxy       bool
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

func parseRangeCookie(strConRge string) (error, int64) {
	//Content-Range:[bytes 0-1/707017362]
	idx := strings.Index(strConRge, "/")
	if idx == -1 {
		return errors.New("format error in " + strConRge), 0
	}
	strSize := strConRge[idx+1:]
	ret, err := strconv.ParseInt(strSize, 10, 64)
	return err, ret
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
func readFilePosion(f *os.File) (error, int64) {
	_, err := f.Seek(0, os.SEEK_SET)
	if err != nil {
		return err, 0
	}
	part := make([]byte, 8)
	_, err = io.ReadFull(f, part)
	if err != nil {
		return err, 0
	}
	var ret, tmp int64
	for i := 0; i < 8; i++ {
		tmp = int64(part[i]) << uint((7-i)*8)
		ret = tmp + ret
	}
	return nil, ret
}
func lookSize(rdFile string) (error, int64) {
	tfile, err := os.OpenFile(rdFile, os.O_RDWR, 0700)
	if err != nil {
		return err, 0
	}
	defer tfile.Close()
	return readFilePosion(tfile)
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

	_, err = io.Copy(df, resp.Body)
	return err
}
func (wk *DWorker) Download(url string) error {
	if wk.HTTPCli == nil {
		return errors.New("pls http client for this worker")
	}
	if wk.Proxy {
		url = getAcceleratedURL(url)
	}
	wk.cancelFlag = false
	wk.WorkStatus = fmt.Sprintf("downloading id = %d for %s\n", wk.WorkerID, url)

	fileName, fileSize, isRange, err := wk.GetDownloadFileInfo(url, "")
	if !isRange {
		return wk.downloadNoRange(url, fileName)
	}
	if err != nil {
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
		err, curPosion = readFilePosion(tfile)
		if err != nil {
			return err
		}
		log.Println(fmt.Sprintf("downloaded %d bytes (%s) for %s", curPosion, ViewHumanShow(curPosion), fileName))
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
	cnt := 0
	wk.CurDownload.URL = url
	wk.CurDownload.FileName = fileName
	wk.CurDownload.CurPosition = curPosion
	wk.CurDownload.Size = fileSize
	log.Println("start ==>", url)
	log.Println("File name ==>", fileName)
	for {
		err = wk.goonDownloadFile(url, curPosion, fileSize, dfile, tfile)
		if err == nil && wk.cancelFlag {
			wk.cancelFlag = false
			break
		}

		if wk.suspendFlag {
			for {
				if wk.suspendFlag {
					time.Sleep(1000 * time.Millisecond)
					wk.WorkStatus = fmt.Sprintf(" suspend on id = %d for %s", wk.WorkerID, url)
					log.Println("loop on suspend")
				} else {
					wk.WorkStatus = fmt.Sprintf("downloading id = %d for %s", wk.WorkerID, url)
					err = errors.New("suspend error")
					break
				}
			}
			log.Println("go on downloading url = ", url)
		}
		if err == nil {
			finish = true
			break
		}
		cnt++
		log.Println("a error = ", err, " start new http connect....,try again ", cnt)
		err, curPosion = readFilePosion(tfile)
		if err != nil {
			log.Println("read position to failed,finish this task")
			return err
		}
		time.Sleep(time.Second * time.Duration(cnt))
		if cnt >= 10 {
			cnt = 0
		}
		log.Println("reset position ", curPosion, " start new http connct...")
	}
	//release resource
	if tfile != nil {
		tfile.Close()
	}
	//remove position file
	if finish {
		os.Remove(rdFile)
	}
	log.Println("done ==>", fileName)
	return nil
}
func (wk *DWorker) addAutoHTTPHeader(header map[string]string) {
	if wk.AuthSve != nil {
		authHTTPHeader := wk.AuthSve.GetTokenHeader()
		if authHTTPHeader != nil {
			for k, v := range authHTTPHeader {
				header[k] = v
			}
		}
	}
}
func (wk *DWorker) GetDownloadFileInfo(uurl string, fileName string) (string, int64, bool, error) {
	//RANGE: bytes=100000-
	header := map[string]string{}
	header["RANGE"] = "bytes=0-1"
	//wk.addAutoHTTPHeader(header)
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
	err, fileSize := parseRangeCookie(strConRge)
	log.Println("response header ", strConRge, " fileSize = ", ViewHumanShow(fileSize))
	if err != nil {
		return "", 0, true, err
	}
	return realName, fileSize, true, nil
}
func (wk *DWorker) goonDownloadFile(uurl string, position int64, fileSize int64, dfile *os.File, tfile *os.File) error {
	rangeHeader := fmt.Sprintf("bytes=%d-", position)
	log.Println("header range :", rangeHeader)
	header := map[string]string{}
	header["RANGE"] = rangeHeader
	//wk.addAutoHTTPHeader(header)
	resp, err := wk.HTTPCli.HttpGet(uurl, header, nil)
	if err != nil {
		return errors.New(fmt.Sprint("download ", uurl, " failed", err))
	}
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
	t0 := time.Now()
	for {
		count, err := resp.Body.Read(buff)
		if err != nil && err != io.EOF {
			return err
		}
		position = position + int64(count)
		if err == io.EOF {
			log.Println("pv & tatol ", position, " & ", fileSize)
		}
		_, err = dfile.Write(buff[0:count])
		if err != nil {
			return err
		}
		if position == fileSize {
			break
		}
		if wk.cancelFlag || wk.suspendFlag {
			break
		}
		readCnt++
		if readCnt > 512 {
			err = recordFilePosion(tfile, position)
			if err != nil {
				log.Println("write postion to failed, err = ", err)
			}
			t1 := time.Now()
			dis := t1.Sub(t0)
			addData := position - wk.CurDownload.CurPosition
			v := addData / dis.Milliseconds() * 1000

			slog := fmt.Sprintf("download rate = %s/s,finish %s %s", ViewHumanShow(v), ViewHumanShow(position), ViewPercent(position, fileSize))
			wk.CurDownload.CurPosition = position
			wk.CurDownload.Desc = slog
			log.Println(slog)
			readCnt = 0
			t0 = t1
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

func webdavGetFileCotent(cli *chttp.HttpClient, buff *bytes.Buffer, uurl string, position int64, fileSize int64) (int64, error) {
	fmt.Println("fileSize :", fileSize)
	maxNeedLen := fileSize - position
	needLen := int64(buff.Cap() - buff.Len())
	if needLen > maxNeedLen {
		needLen = maxNeedLen
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", position, position+needLen-1)
	fmt.Println("header range :", rangeHeader)
	header := map[string]string{}
	header["RANGE"] = rangeHeader
	resp, err := cli.HttpGet(uurl, header, nil)
	if err != nil {
		return 0, errors.New(fmt.Sprint("download ", uurl, " failed", err))
	}
	defer resp.Body.Close()
	strConRge := resp.Header.Get("Content-Range")
	if strConRge == "" {
		return 0, errors.New("no support range")
	}
	sc := resp.StatusCode / 100
	if sc != 2 {
		return 0, errors.New("request errors,status code = " + strconv.Itoa(sc) + "," + resp.Status)
	}
	fmt.Println("response header range :", strConRge)
	fmt.Println("seed to :", position)
	//1M 1k*1024
	count, err := io.CopyN(buff, resp.Body, needLen)
	fmt.Println("read to buff :", count)
	return count, err
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
