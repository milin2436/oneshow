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
)

const k320 int64 = 327680
const BLOCK int64 = 10 * k320
const TMP_FILE_FIX = ".one.tmp"

type CurTask struct {
	FullPath    string
	FileName    string
	URL         string
	FileSize    int64
	CurPosition int64
}

func (cli *OneClient) APIGetUploadFileInfo(URL string) (*UploadURLResult, error) {
	header := cli.SetOneDriveAPIToken()
	objs := new(UploadURLResult)
	resp, err := cli.HTTPClient.HttpGet(URL, header, nil)
	err = HandleResponForParseAPI(resp, err, objs)
	if err != nil {
		return nil, err
	}
	return objs, nil
}

//APIListFilesByPath get files by path

func (cli *OneClient) APICreateUploadSession(driveID string, path string) (*UploadURLResult, error) {
	uri := "/drives/%s/root:%s:/createUploadSession"
	URL := cli.APIHost + fmt.Sprintf(uri, driveID, path)
	if path == "/" {
		uri := "/drives/%s/root/createUploadSession"
		URL = cli.APIHost + fmt.Sprintf(uri, driveID)
	}

	bodyTmp := `{
  "item": {
	"@microsoft.graph.conflictBehavior": "rename"
  }
}`
	core.Println("APIListFilesByPath request url = ", URL)
	core.Println("body =", bodyTmp)

	header := cli.SetOneDriveAPIToken()
	objs := new(UploadURLResult)
	resp, err := cli.HTTPClient.HttpPost(URL, header, bodyTmp)
	err = HandleResponForParseAPI(resp, err, objs)
	if err != nil {
		return nil, err
	}
	return objs, nil
}

func (cli *OneClient) apiUploadFilePart(task *CurTask, URL string, file *os.File, st int64, ed int64, fileSize int64, buff *bytes.Buffer) (*UploadURLResult, error) {
	header := map[string]string{}
	objs := new(UploadURLResult)

	len := ed - st + 1
	_, err := io.CopyN(buff, file, len)
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
	t0 := time.Now()
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
	dis := time.Now().Sub(t0)
	v := len / dis.Milliseconds() * 1000
	remainTime := (fileSize - ed - 1) / v
	fmt.Printf("file = %s;%s/s done %s need time %ds filesize:%s\n", task.FileName, ViewHumanShow(v), ViewPercent(ed+1, fileSize), remainTime, ViewHumanShow(fileSize))

	if err != nil {
		return nil, err
	}
	return objs, nil
}
func (cli *OneClient) APIUploadFilePart(task *CurTask, URL string, file *os.File, position int64, fileSize int64) error {
	_, err := file.Seek(position, os.SEEK_SET)
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
		_, err := cli.apiUploadFilePart(task, URL, file, start, end, fileSize, &buff)
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
		_, err = cli.apiUploadFilePart(task, URL, file, start, end, fileSize, &buff)
		if err != nil {
			return err
		}
	}
	core.Println("filesize ", fileSize)
	return nil
}
func parsePositionFromTmp(ret *UploadURLResult) (int64, error) {
	if len(ret.NextExpectedRanges) == 0 {
		return 0, errors.New("NextExpectedRanges is empty")
	}
	arr := ret.NextExpectedRanges[0]
	core.Println("first range =  ", arr)
	arrList := strings.Split(arr, "-")
	startStr := arrList[0]
	core.Println("start position = ", startStr)
	return strconv.ParseInt(startStr, 10, 64)
}

func (cli *OneClient) UploadBigFile(srcFile string, driveID string, path string) error {
	absSrcFile, err := filepath.Abs(srcFile)
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
	core.Println("full path = ", absSrcFile)
	//find tmp file
	parent := filepath.Dir(absSrcFile)
	fileInfo := filepath.Join(parent, info.Name()+TMP_FILE_FIX)
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
		ret, err := cli.APICreateUploadSession(driveID, path)
		if err != nil {
			return err
		}
		uploadURL = ret.UploadURL
		ioutil.WriteFile(fileInfo, []byte(uploadURL), 0661)
	}
	file, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer file.Close()
	cur := new(CurTask)
	cur.CurPosition = position
	cur.FullPath = absSrcFile
	cur.FileName = info.Name()
	cur.URL = uploadURL
	cur.FileSize = info.Size()
	err = cli.APIUploadFilePart(cur, uploadURL, file, position, info.Size())
	if err != nil {
		return err
	}
	os.Remove(fileInfo)
	return nil
}
