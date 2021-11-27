package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/milin2436/oneshow/one"
)

//var cli *one.OneClient

func AutoUpdateToken(cli *one.OneClient) {
	for {
		CheckToken(cli)
		time.Sleep(time.Minute)
	}
}
func CheckToken(cli *one.OneClient) error {
	expires := time.Time(cli.Token.ExpiresTime)

	expires = expires.Truncate(time.Minute)
	if time.Now().After(expires) {
		//fmt.Println("to expries time, update token")
		newToken, err := cli.UpdateToken()
		if err != nil {
			return err
		}
		cli.Token = newToken
	}
	return nil

}

func OutHtml(body string) string {
	html := `
	<html>
		<head>
			<title></title>
		</head>
		<body>
			%s
		</body>
	</html>
	`
	ret := fmt.Sprintf(html, body)
	return ret
}
func humanShow(isize int64) string {
	size := float64(isize)
	if size < 1024 {
		return fmt.Sprintf("%.2fbytes", size)
	}
	tmp := size / 1024.0
	if tmp < 1024 {
		return fmt.Sprintf("%.2fK", tmp)
	}
	tmp = tmp / 1024.0
	if tmp < 1024 {
		return fmt.Sprintf("%.2fM", tmp)
	}
	tmp = tmp / 1024.0
	return fmt.Sprintf("%.2fG", tmp)
}

func CmdLS(dirPath string, cli *one.OneClient) string {
	var buff bytes.Buffer
	ret, err := cli.APIListFilesByPath(cli.CurDriveID, dirPath)
	if dirPath != "/" {
		dirPath = dirPath + "/"
	}
	if err != nil {
		fmt.Println("err = ", err)
		return err.Error()
	}
	for _, v := range ret.Value {
		if v.Folder != nil {
			//<a src="" >$name</a>
			s := fmt.Sprintf(`<div><a href="/vfs?path=%s">%s/</a></div>`, dirPath+v.Name, v.Name)
			buff.WriteString(s)
		} else {
			s := fmt.Sprintf(`<div><a href="%s" target="blank">%s</a> %s <a href="/play?id=%s" target="blank">play</a></div>`, v.DownloadURL, v.Name, humanShow(v.Size), url.QueryEscape(v.DownloadURL))
			buff.WriteString(s)
		}
		//buff.WriteString("<br />")
	}
	return buff.String()
}

func GetQueryParamByKey(r *http.Request, key string) string {

	keys, ok := r.URL.Query()[key]
	if !ok || len(keys[0]) < 1 {
		return ""
	}

	return keys[0]
}
func Serivce(address string, https bool) {
	var err1 error
	cli, err1 := one.NewOneClient()
	if err1 != nil {
		panic(err1.Error())
	}
	http.HandleFunc("/p", func(w http.ResponseWriter, r *http.Request) {
		dd := GetQueryParamByKey(r, "d")
		w.Write([]byte(dd))
	})

	http.HandleFunc("/vfs", func(w http.ResponseWriter, r *http.Request) {
		dirPath := GetQueryParamByKey(r, "path")
		if dirPath == "" {
			dirPath = "/"
		}
		strLen := len(dirPath)
		if strLen > 1 && dirPath[strLen-1] == '/' {
			dirPath = dirPath[:strLen-1]
		}
		err := CheckToken(cli)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		body := CmdLS(dirPath, cli)
		html := OutHtml(body)
		w.Write([]byte(html))
	})
	http.HandleFunc("/play", func(w http.ResponseWriter, r *http.Request) {
		dirPath := GetQueryParamByKey(r, "id")
		bodyTmp := `
		<video width="640" height="480" controls="controls">
			<source src="%s" />
		</video>
		`
		body := fmt.Sprintf(bodyTmp, dirPath)
		html := OutHtml(body)
		w.Write([]byte(html))
	})

	http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		uu := r.URL.String()
		var buff bytes.Buffer
		buff.WriteString("HOST = ")
		buff.WriteString(r.RemoteAddr)
		buff.WriteString("\n")

		buff.WriteString("method = ")
		buff.WriteString(r.Method)
		buff.WriteString("\n")

		buff.WriteString("url = ")
		buff.WriteString(uu)
		buff.WriteString("\n")

		buff.WriteString("header = ")
		for k, v := range r.Header {
			buff.WriteString(k)
			buff.WriteString(":")
			if len(v) == 1 {
				buff.WriteString(v[0])
			} else {
				for _, sv := range v {
					buff.WriteString(sv)
					buff.WriteString(";")
				}
			}
			buff.WriteString("\n")
		}
		buff.WriteString("\n")

		buff.WriteString("body = ")
		if r.Body != nil {
			defer r.Body.Close()
			io.Copy(&buff, r.Body)
		}
		w.Write(buff.Bytes())
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{\"error\":\"ok\"}"))
	})
	var err error
	if https {
		fmt.Println("https server on ", address)
		err = http.ListenAndServeTLS(address, "cacert.pem", "privkey.pem", nil)
	} else {
		fmt.Println("http server on ", address)
		err = http.ListenAndServe(address, nil)
	}
	if err != nil {
		fmt.Println("run thie service to failed on error = ", err)
	}
}
