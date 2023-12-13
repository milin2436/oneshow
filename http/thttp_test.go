package http

import (
	"log"
	"testing"
)

func Test_ExecOnTime(t *testing.T) {
	mytest()
}

func case1() {
	ps := map[string]string{}
	ps["test"] = "xx"

	header := map[string]string{}
	header["Host"] = "www.baidu.com"
	header["xQ"] = "www.baidu.com"
	//header["Content-Type"] = "json/text"
	reps, err := SimpleHttpClient.HttpFormPost("http://127.0.0.1:4444/req", header, ps)
	html, err := HandleRespon2String(reps, err)
	if err != nil {
		log.Println("err = ", err)
	}
	log.Println(html)
}
func case2() {
	err := SimpleHttpClient.HttpGetDownloadFile("https://www.baidu.com/", "")
	if err != nil {
		log.Printf("err = %s\n", err.Error())
	}
}

func mytest() {
	case2()
}
