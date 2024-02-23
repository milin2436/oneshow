package one

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/milin2436/oneshow/core"
	chttp "github.com/milin2436/oneshow/http"
)

type UploadTask interface {
	Parent() string
	Name() string
	Size() int64
	Open() error
	Close() error
	Init() error
	SeekPosition(position int64) error
	// len = end - start + 1
	Read(buff *bytes.Buffer, start int64, end int64, len int64) error
}

type LocalFileUploadTask struct {
	filePath   string
	name       string
	parentPath string
	source     string
	file       *os.File
	fileSize   int64
}

func (task *LocalFileUploadTask) Parent() string {
	if task.parentPath == "" {
		task.parentPath = filepath.Dir(task.filePath)
	}
	return task.parentPath
}
func (task *LocalFileUploadTask) Name() string {
	return task.name
}
func (task *LocalFileUploadTask) Size() int64 {
	return task.fileSize
}
func (task *LocalFileUploadTask) Open() error {
	file, err := os.Open(task.filePath)
	if err != nil {
		return err
	}
	task.file = file
	return nil
}
func (task *LocalFileUploadTask) Close() error {
	return task.file.Close()
}
func (task *LocalFileUploadTask) Init() error {
	absSrcFile, err := filepath.Abs(task.source)
	if err != nil {
		return err
	}
	info, err := os.Stat(absSrcFile)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("file is dir : " + absSrcFile)
	}
	task.fileSize = info.Size()
	task.filePath = absSrcFile
	task.name = info.Name()
	return nil
}
func (task *LocalFileUploadTask) SeekPosition(position int64) error {
	_, err := task.file.Seek(position, os.SEEK_SET)
	return err
}
func (task *LocalFileUploadTask) Read(buff *bytes.Buffer, start int64, end int64, len int64) error {
	_, err := io.CopyN(buff, task.file, len)
	return err
}

type URLUploadTask struct {
	FileName    string
	URL         string
	FileSize    int64
	CurPosition int64
	HTTPClient  *chttp.HttpClient
}

func (task *URLUploadTask) Name() string {
	return task.FileName
}
func (task *URLUploadTask) Size() int64 {
	return task.FileSize
}
func (task *URLUploadTask) Parent() string {
	return "."
}
func (task *URLUploadTask) Open() error {
	return nil
}
func (task *URLUploadTask) Close() error {
	return nil
}

func (task *URLUploadTask) Init() error {
	wk := new(DWorker)
	wk.HTTPCli = task.HTTPClient
	fileName, fileSize, isRange, err := wk.GetDownloadFileInfo(task.URL, "")
	if err != nil {
		return err
	}
	if !isRange {
		return fmt.Errorf("no range for this URL: %s", task.URL)
	}
	task.FileName = fileName
	task.FileSize = fileSize
	return nil
}
func (task *URLUploadTask) SeekPosition(position int64) error {
	return nil
}

func (task *URLUploadTask) getDataBlock(buff *bytes.Buffer, start int64, end int64) error {
	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)
	core.Println("header range :", rangeHeader)
	header := map[string]string{}
	header["RANGE"] = rangeHeader
	resp, err := task.HTTPClient.HttpGet(task.URL, header, nil)
	if err != nil {
		return errors.New(fmt.Sprint("download ", task.URL, " failed", err))
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
	core.Println("response header range :", strConRge)
	core.Println("start :", start)
	_, err = io.Copy(buff, resp.Body)
	return err
}
func (task *URLUploadTask) Read(buff *bytes.Buffer, start int64, end int64, len int64) error {
	return task.getDataBlock(buff, start, end)
}
func (cli *OneClient) checkSourceType(source string) (UploadTask, error) {
	URL := strings.ToLower(source)
	if strings.HasPrefix(URL, "http://") || strings.HasPrefix(URL, "https://") {
		task := new(URLUploadTask)
		task.HTTPClient = cli.HTTPClient
		task.URL = source
		return task, nil
	}
	task := new(LocalFileUploadTask)
	task.source = source
	return task, nil
	//return nil, errors.New("can not find source type for " + source)
}

func (cli *OneClient) apiUploadSourcePart(task UploadTask, URL string, st int64, ed int64, fileSize int64, buff *bytes.Buffer) (*UploadURLResult, error) {
	t0 := time.Now()
	header := map[string]string{}
	objs := new(UploadURLResult)

	len := ed - st + 1
	err := task.Read(buff, st, ed, len)
	//_, err := io.CopyN(buff, file, len)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("PUT", URL, buff)
	if err != nil {
		return nil, err
	}
	//Content-Length: 26
	//Content-Range: bytes 0-25/128
	start := strconv.FormatInt(st, 10)
	end := strconv.FormatInt(ed, 10)
	fileSizeStr := strconv.FormatInt(fileSize, 10)
	bytes := "bytes " + start + "-" + end + "/" + fileSizeStr
	header["Content-Range"] = bytes
	core.Println("bytes = ", bytes)
	//add header
	for k, v := range header {
		req.Header.Add(k, v)
	}
	//req.ContentLength = len
	resp, err := cli.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 201 {
		err = HandleResponForParseAPI(resp, err, objs)
		if err == nil {
			return nil, nil
		}
	} else {
		err = HandleResponForParseAPI(resp, err, objs)
	}
	//fix issue 4
	if err != nil {
		return nil, err
	}
	dis := time.Now().Sub(t0)
	v := len / dis.Milliseconds() * 1000
	remainTime := (fileSize - ed - 1) / v
	fmt.Printf("file = %s;%s/s done %s need time %ds filesize:%s\n", task.Name(), ViewHumanShow(v), ViewPercent(ed+1, fileSize), remainTime, ViewHumanShow(fileSize))

	return objs, nil
}
func (cli *OneClient) APIUploadSourcePart(task UploadTask, URL string, position int64, fileSize int64) error {
	err := task.SeekPosition(position)
	if err != nil {
		return err
	}
	var buff bytes.Buffer
	buff.Grow(int(BLOCK))
	remain := fileSize - position
	blist := remain / BLOCK
	for i := int64(0); i < blist; i++ {
		start := position + i*BLOCK
		end := start + BLOCK - 1
		core.Println("start = ", start, "  end = ", end)
		_, err := cli.apiUploadSourcePart(task, URL, start, end, fileSize, &buff)
		buff.Reset()
		if err != nil {
			return err
		}
	}
	last := remain % BLOCK
	if last != 0 {
		start := fileSize - last
		end := fileSize - 1
		core.Println("start = ", start, "  end = ", end)
		_, err = cli.apiUploadSourcePart(task, URL, start, end, fileSize, &buff)
		if err != nil {
			return err
		}
	}
	core.Println("filesize ", fileSize)
	return nil
}
func (cli *OneClient) UploadSourceTryAgain(source string, driveID string, oneDriveParentPath string, tryLimit int) error {
	var err error
	for ti := 1; ti <= tryLimit; ti++ {
		err = cli.UploadSource(source, driveID, oneDriveParentPath)
		if err != nil {
			fmt.Println("err = ", err, " in ", source)
			fmt.Printf("try again for the %dth time\n", ti)
			//exitQueue <- 0
		} else {
			fmt.Println("done file = ", source)
			return nil
		}
	}
	return err
}

func (cli *OneClient) UploadSource(source string, driveID string, oneDriveParentPath string) error {
	task, err := cli.checkSourceType(source)
	if err != nil {
		return err
	}
	err = task.Init()
	if err != nil {
		return err
	}
	oneDrivePath := filepath.Join(oneDriveParentPath, task.Name())
	//find tmp file
	parent := task.Parent()
	fileInfo := filepath.Join(parent, task.Name()+TMP_FILE_FIX)
	infoTmp, err := os.Stat(fileInfo)

	position := int64(0)
	uploadURL := ""
	if err == nil && !infoTmp.IsDir() {
		//continue upload
		text, err := ioutil.ReadFile(fileInfo)
		if err != nil {
			return err
		}
		uploadURL = string(text)
		uploadURL = strings.TrimSpace(uploadURL)
		core.Println("URL === ", uploadURL)
		ret, err := cli.APIGetUploadFileInfo(uploadURL)
		if err != nil {
			return err
		}
		position, err = parsePositionFromTmp(ret)
		if err != nil {
			return err
		}
	} else {
		//new a upload
		ret, err := cli.APICreateUploadSession(driveID, oneDrivePath)
		if err != nil {
			return err
		}
		uploadURL = ret.UploadURL
		ioutil.WriteFile(fileInfo, []byte(uploadURL), 0661)
	}
	err = task.Open()
	if err != nil {
		return err
	}
	defer task.Close()

	//fix issue 5. The transfer upload support is not very good and there are many problems.
	//The upload and transfer function is turned off by default.

	//uploadURL = AcceleratedURL(uploadURL)
	core.Println("upload url =", uploadURL)

	err = cli.APIUploadSourcePart(task, uploadURL, position, task.Size())
	if err != nil {
		return err
	}
	os.Remove(fileInfo)
	return nil
}
