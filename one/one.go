package one

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/milin2436/oneshow/core"
	chttp "github.com/milin2436/oneshow/http"
	"github.com/milin2436/oneshow/one/utils"
)

var ClientID string = "51d4977e-8740-41c9-956b-bc5fa4f58806"

var ClientSecret string = "jvv9q-o9Yt2bxg.6kRmOLi~5xhQDrN.5._"

var Scope string = "Files.Read Files.ReadWrite Files.Read.All Files.ReadWrite.All offline_access Sites.Read.All User.Read"

var CallbackURL = "http://localhost:4444/result"

// OneClient is context object
type OneClient struct {
	HTTPClient *chttp.HTTPClient

	SSOHost string
	APIHost string

	UserName   string
	ConfigFile string
	Token      *AuthToken
	CurDriveID string
}

// GetAuthCodeURL gen an URL for get auto code from API
func (cli *OneClient) GetAuthCodeURL() string {
	parms := cli.GetOneDriveAppInfo()
	parms["access_type"] = "offline"
	parms["response_type"] = "code"
	parms["state"] = "uid-oneshow"

	uri := "/common/oauth2/v2.0/authorize"
	parmsVal := url.Values{}
	for k, v := range parms {
		parmsVal.Add(k, v)
	}
	URL := cli.SSOHost + uri + "?" + parmsVal.Encode()
	core.DebugPrintln(URL)
	return URL
}

// GetFirstToken get token and refresh token by code
func (cli *OneClient) GetFirstToken(code string) error {
	parms := cli.GetOneDriveAppInfoWithSecret()
	parms["grant_type"] = "authorization_code"
	parms["code"] = code

	uri := "/common/oauth2/v2.0/token"
	URL := cli.SSOHost + uri

	token, err := HandleResponseForParseToken(cli.HTTPClient.HTTPFormPost(URL, nil, parms))
	if err != nil {
		core.DebugPrintln("err=", err)
		return err
	}
	core.DebugPrintln("first refresh token:", token.RefreshToken)
	cli.Token = token
	cli.SaveTokenToHomeDefault(token)
	return nil
}

// UpdateToken update expired token
func (cli *OneClient) UpdateToken() (*AuthToken, error) {
	parms := cli.GetOneDriveAppInfoWithSecret()
	parms["grant_type"] = "refresh_token"
	parms["refresh_token"] = cli.Token.RefreshToken

	uri := "/common/oauth2/v2.0/token"
	URL := cli.SSOHost + uri
	token, err := HandleResponseForParseToken(cli.HTTPClient.HTTPFormPost(URL, nil, parms))
	if err != nil {
		core.DebugPrintln("err=", err)
		return nil, err
	}
	//using old drive ID
	token.DriveID = cli.CurDriveID
	cli.Token = token
	err = cli.SaveTokenToUserConfig(token)
	if err != nil {
		return nil, err
	}
	core.DebugPrintln(token.AccessToken)
	return token, nil
}

// SaveTokenToUserConfig save token to the user's own config
func (cli *OneClient) SaveTokenToUserConfig(token *AuthToken) error {
	exTime := time.Now().Add(time.Second * time.Duration(token.ExpiresIn-60))
	token.ExpiresTime = Timestamp(exTime)
	if token.DriveID == "" {
		dri, err := cli.APIGetMeDrive()
		if err != nil {
			return err
		}
		token.DriveID = dri.ID
	}
	cli.SaveTokenToHome(token)
	return nil
}

// SaveTokenToHomeDefault save token to default config when first login
func (cli *OneClient) SaveTokenToHomeDefault(token *AuthToken) {
	exTime := time.Now().Add(time.Second * time.Duration(token.ExpiresIn-60))
	token.ExpiresTime = Timestamp(exTime)
	dri, err := cli.APIGetMeDrive()
	if err == nil {
		token.DriveID = dri.ID
	}
	SaveTokenToDefaultPath(token)
}

// GetOneDriveAppInfo setup application information
func (cli *OneClient) GetOneDriveAppInfo() map[string]string {
	parms := map[string]string{}
	parms["client_id"] = ClientID
	parms["scope"] = Scope
	parms["redirect_uri"] = CallbackURL
	return parms
}

// GetOneDriveAppInfoWithSecret with secret info
func (cli *OneClient) GetOneDriveAppInfoWithSecret() map[string]string {
	p := cli.GetOneDriveAppInfo()
	p["client_secret"] = ClientSecret
	return p
}

// SetOneDriveAPIToken for http request setup a token
func (cli *OneClient) SetOneDriveAPIToken() map[string]string {
	header := map[string]string{}
	header["Content-Type"] = "application/json"
	header["Authorization"] = "Bearer " + cli.Token.AccessToken
	return header
}

// APIGetMeDrive get onedrive infomation
func (cli *OneClient) APIGetMeDrive() (*Drive, error) {
	uri := "/me/drive"
	URL := cli.APIHost + uri
	header := cli.SetOneDriveAPIToken()
	dri := new(Drive)
	resp, err := cli.HTTPClient.HTTPGet(URL, header, nil)
	err = HandleResponseForParseAPI(resp, err, dri)
	if err != nil {
		fmt.Println("err=", err)
		return nil, err
	}
	core.DebugPrintln("id=", dri.ID)
	return dri, nil
}

func (cli *OneClient) apiListFilesByPath(url string) (*ListChildrenResponse, error) {
	core.DebugPrintln("APIListFilesByPath request url = ", url)
	header := cli.SetOneDriveAPIToken()
	objs := new(ListChildrenResponse)
	resp, err := cli.HTTPClient.HTTPGet(url, header, nil)
	err = HandleResponseForParseAPI(resp, err, objs)
	if err != nil {
		return nil, err
	}
	return objs, nil
}

// APIListFilesByPath get files by path, following pagination links
func (cli *OneClient) APIListFilesByPath(driveID string, path string) (*ListChildrenResponse, error) {
	uri := "/drives/%s/root:%s:/children"
	URL := cli.APIHost + fmt.Sprintf(uri, driveID, path)
	if path == "/" {
		uri := "/drives/%s/root/children"
		URL = cli.APIHost + fmt.Sprintf(uri, driveID)
	}
	ret := []Item{}
	var resp *ListChildrenResponse
	var err error
	for {
		resp, err = cli.apiListFilesByPath(URL)
		if err != nil {
			return resp, err
		}
		ret = append(ret, resp.Value...)
		if resp.NextLink == "" {
			break
		}
		URL = resp.NextLink
	}
	resp.Value = ret
	return resp, err
}

// APISearchByKey search files by Key
func (cli *OneClient) APISearchByKey(driveID string, key string) (*ListChildrenResponse, error) {
	uri := "/drives/%s/root/search(q='%s')"
	URL := cli.APIHost + fmt.Sprintf(uri, driveID, key)
	core.DebugPrintln("APISearchByKey request url = ", URL)
	header := cli.SetOneDriveAPIToken()
	objs := new(ListChildrenResponse)
	resp, err := cli.HTTPClient.HTTPGet(URL, header, nil)
	err = HandleResponseForParseAPI(resp, err, objs)
	if err != nil {
		return nil, err
	}
	return objs, nil
}

// APIGetFile get a file by file path
func (cli *OneClient) APIGetFile(driveID string, path string) (*Item, error) {
	URL := ""
	if path == "/" {
		uri := "/drives/%s/root"
		URL = cli.APIHost + fmt.Sprintf(uri, driveID)
	} else {
		uri := "/drives/%s/root:%s"
		URL = cli.APIHost + fmt.Sprintf(uri, driveID, path)
	}
	core.DebugPrintln("URI = ", URL)
	header := cli.SetOneDriveAPIToken()
	objs := new(Item)
	resp, err := cli.HTTPClient.HTTPGet(URL, header, nil)
	err = HandleResponseForParseAPI(resp, err, objs)
	if err != nil {
		return nil, err
	}
	return objs, nil
}

// APIGetFileByID get a file by item ID
func (cli *OneClient) APIGetFileByID(driveID string, ID string) (*Item, error) {
	uri := "/drives/%s/items/%s"
	URL := cli.APIHost + fmt.Sprintf(uri, driveID, ID)
	header := cli.SetOneDriveAPIToken()
	objs := new(Item)
	resp, err := cli.HTTPClient.HTTPGet(URL, header, nil)
	err = HandleResponseForParseAPI(resp, err, objs)
	if err != nil {
		return nil, err
	}
	return objs, nil
}

// APIUpdateFileByItemID update a file's name and parent directory
func (cli *OneClient) APIUpdateFileByItemID(driveID string, itemID string, newName string, newPathID string) (bool, error) {
	uri := "/drives/%s/items/%s"
	URL := cli.APIHost + fmt.Sprintf(uri, driveID, itemID)
	header := cli.SetOneDriveAPIToken()

	bodyTmp := `{
  "parentReference": {
    "id": "%s"
  },
  "name": "%s"
}`
	bodyTmp = fmt.Sprintf(bodyTmp, newPathID, newName)
	core.DebugPrintln("body =", bodyTmp)
	resp, err := cli.HTTPClient.HTTPRequest("PATCH", URL, header, bodyTmp)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == 200 {
		return true, nil
	} else {
		return false, HandleResponseForParseAPI(resp, nil, nil)
	}
}

// APIDelFileByItemID delete file by item ID
func (cli *OneClient) APIDelFileByItemID(driveID string, itemID string) (bool, error) {
	uri := "/drives/%s/items/%s"
	URL := cli.APIHost + fmt.Sprintf(uri, driveID, itemID)
	header := cli.SetOneDriveAPIToken()

	resp, err := cli.HTTPClient.HTTPRequest("DELETE", URL, header, "")
	if err != nil {
		return false, err
	}
	if resp.StatusCode == 204 {
		return true, nil
	} else {
		return false, HandleResponseForParseAPI(resp, nil, nil)
	}
}

// APIDelFile delete file by file path
func (cli *OneClient) APIDelFile(driveID string, filePath string) (bool, error) {
	uri := "/drives/%s/root:%s"
	URL := cli.APIHost + fmt.Sprintf(uri, driveID, filePath)
	header := cli.SetOneDriveAPIToken()

	resp, err := cli.HTTPClient.HTTPRequest("DELETE", URL, header, "")
	if err != nil {
		return false, err
	}
	if resp.StatusCode == 204 {
		return true, nil
	} else {
		return false, HandleResponseForParseAPI(resp, nil, nil)
	}
}

// APImkdir create a dir
func (cli *OneClient) APImkdir(driveID string, path string, dirName string) (*Item, error) {
	uri := "/drives/%s/root:%s:/children"
	URL := cli.APIHost + fmt.Sprintf(uri, driveID, path)
	if path == "/" {
		uri := "/drives/%s/root/children"
		URL = cli.APIHost + fmt.Sprintf(uri, driveID)
	}
	core.DebugPrintln(URL)
	header := cli.SetOneDriveAPIToken()
	objs := new(Item)

	bodyTpl := `{
  "name": "%s",
  "folder": { },
  "@microsoft.graph.conflictBehavior": "rename"
}`
	body := fmt.Sprintf(bodyTpl, dirName)
	resp, err := cli.HTTPClient.HTTPPost(URL, header, body)
	err = HandleResponseForParseAPI(resp, err, objs)
	if err != nil {
		return nil, err
	}
	return objs, nil
}

// HandleResponseForParseToken parse token
func HandleResponseForParseToken(resp *http.Response, err error) (*AuthToken, error) {
	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	buff, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	core.DebugPrintln("token = ", string(buff))
	core.DebugPrintln("statuscode = ", resp.StatusCode)
	if resp.StatusCode == 200 {
		token := new(AuthToken)
		perr := json.Unmarshal(buff, token)
		if perr != nil {
			return nil, perr
		}
		return token, nil
	} else {
		apiErr := new(AuthError)
		perr := json.Unmarshal(buff, apiErr)
		if perr != nil {
			return nil, perr
		}
		return nil, errors.New(apiErr.ErrorDescription)
	}
}

// HandleResponseForParseAPI parse api
func HandleResponseForParseAPI(resp *http.Response, err error, objs interface{}) error {
	if resp == nil {
		return err
	}
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	buff, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	core.DebugPrintln(string(buff))
	core.DebugPrintln("code,", resp.StatusCode)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		perr := json.Unmarshal(buff, objs)
		if perr != nil {
			return perr
		}
	} else {
		apiErr := new(Error)
		perr := json.Unmarshal(buff, apiErr)
		if perr != nil {
			return perr
		}
		return errors.New(apiErr.ErrorInfo.Message)
	}
	return nil
}

// NewBaseOneClient for new user
func NewBaseOneClient() *OneClient {
	cli := new(OneClient)
	httpCli := chttp.NewHTTPClient()
	cli.HTTPClient = httpCli
	cli.APIHost = "https://graph.microsoft.com/v1.0"
	cli.SSOHost = "https://login.microsoftonline.com"
	return cli
}

// NewOneClient instance a OneClient
func NewOneClient() (*OneClient, error) {
	u := getCurUser()
	return NewOneClientUser(u)
}

// NewOneClientUser instance a OneClient for a specific user
func NewOneClientUser(user string) (*OneClient, error) {
	cli := NewBaseOneClient()
	cli.setUserInfo(user)
	tk := cli.getConfigAuthToken()
	if tk == nil {
		return nil, errors.New("pls config a new user")
	}
	cli.Token = tk
	expires := time.Time(tk.ExpiresTime)
	if time.Now().After(expires) {
		core.DebugPrintln("token expired, updating")
		newToken, err := cli.UpdateToken()
		if err != nil {
			return nil, err
		}
		cli.Token = newToken
	}
	if tk != nil {
		cli.CurDriveID = tk.DriveID
	}
	return cli, nil
}

// GetTokenHeader for other client
func (cli *OneClient) GetTokenHeader() map[string]string {
	return cli.SetOneDriveAPIToken()
}

// Download file from api
func (cli *OneClient) Download(file string, downloadDir string, a bool) {
	dri, err := cli.APIGetFile(cli.CurDriveID, file)
	if err != nil {
		fmt.Println("err = ", err)
		return
	}
	wk := NewDWorker()
	wk.HTTPClient = cli.HTTPClient
	wk.AuthService = cli
	wk.DownloadDir = downloadDir
	wk.Proxy = a
	err = wk.Download(dri.DownloadURL)
	if err != nil {
		fmt.Println("failed on ", err, " for ", file)
	}
}

func openBrowser(cmd string, URL ...string) error {
	mycmd := exec.Command(cmd, URL...)
	err := mycmd.Start()
	go func() {
		err = mycmd.Wait()
		if err != nil {
			fmt.Printf("Command finished with error: %v", err)
		}
	}()
	return err
}

// DoAutoForNewUser config a new user
func (cli *OneClient) DoAutoForNewUser() {
	//open browser
	go func() {
		time.Sleep(time.Second * 2)
		autoURL := cli.GetAuthCodeURL()
		if runtime.GOOS == "linux" {
			openBrowser("xdg-open", autoURL)
		} else {
			autoURL = strings.ReplaceAll(autoURL, "&", "^&")
			openBrowser("cmd", "/C", "start", autoURL)
		}
	}()
	respURL := cli.GetOneDriveAppInfo()["redirect_uri"]
	u, _ := url.Parse(respURL)
	sm := http.NewServeMux()
	server := http.Server{Addr: u.Host, Handler: sm}
	sm.HandleFunc(u.Path, func(w http.ResponseWriter, r *http.Request) {
		dd := utils.GetQueryParamByKey(r, "code")
		fmt.Println("code=", dd)
		if dd == "" {
			return
		}
		err := cli.GetFirstToken(dd)
		if err != nil {
			ss := fmt.Sprintf("Token saved to failed err = %s", err.Error())
			w.Write([]byte(ss))
		} else {
			w.Write([]byte("Token saved to local successfully."))
		}
		go func() {
			time.Sleep(time.Second * 3)
			server.Shutdown(context.Background())
		}()
	})
	err := server.ListenAndServe()
	if err != nil && http.ErrServerClosed == err {
		fmt.Println("Token saved successfully, temporary HTTP server shut down and exited.")
	} else {
		fmt.Println("HTTP server start to failed,err = ", err)
	}
}

// VerifyAndUpdateToken refresh the token before it expires
func (cli *OneClient) VerifyAndUpdateToken() error {
	expires := time.Time(cli.Token.ExpiresTime)
	expires = expires.Truncate(time.Minute)
	if time.Now().After(expires) {
		newToken, err := cli.UpdateToken()
		if err != nil {
			return err
		}
		cli.Token = newToken
	}
	return nil
}
