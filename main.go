package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gngpp/vlog"
	"github.com/pelletier/go-toml"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"go-aliyun-webdav/aliyun"
	"go-aliyun-webdav/aliyun/cache"
	"go-aliyun-webdav/aliyun/model"
	"go-aliyun-webdav/webdav"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

func init() {
	cache.Init()
}

var Version = "v1.1.2"

type Task struct {
	Id string `json:"id"`
}

type config struct {
	Connection struct {
		Port         string
		Path         string
		User         string
		Password     string
		RefreshToken string
	}
	Options struct {
		ShowDebugLog bool
		IsMobile     bool
	}
}

func getRefreshToken(isMobile bool) string {
	time.Sleep(time.Second * 1)
	api := aliyun.NewApi(isMobile)
	content, err := api.GetGeneratorQrCodeContent()
	if err != nil {
		return "null"
	}

	q := aliyun.NewQrCode(content.CodeContent, true)
	q.Print()
	vlog.Infof("请使用手机扫描二维码登录")
	qrCodeResult, b := api.GetQueryQrCodeResult()
	if b {
		bytess, err := base64.StdEncoding.DecodeString(qrCodeResult.Content.Data.BizExt)
		if err != nil {
			return "null"
		}
		result := &model.LoginResult{}
		err = json.Unmarshal(bytess, result)
		if err != nil {
			return "null"
		}
		return result.PdsLoginResult.RefreshToken
	}
	return "null"
}

func UpdateRefreshToken(new string) {
	cig, err := readConfig()
	if err != nil {
		return
	}
	cig.Connection.RefreshToken = new
	data, err := toml.Marshal(cig)
	if err != nil {
		return
	}
	if err := os.WriteFile("config.toml", data, 0644); err != nil {
		return
	}
}

func main() {
	oldcig, err := readConfig()
	webtoken := aliyun.RefreshToken(oldcig.Connection.RefreshToken)
	//if reflect.DeepEqual(refreshResult, model.RefreshTokenModel{}) {
	if webtoken == nil {
		vlog.Infof("refreshToken已过期,请重新扫描二维码！")
		refreshToken := getRefreshToken(oldcig.Options.IsMobile)
		webtoken = aliyun.RefreshToken(refreshToken)
		UpdateRefreshToken(refreshToken)
	}

	newcig, err := readConfig()

	//GetDb()
	var port string
	var path string
	var user string
	var pwd string
	var showlog bool

	if err != nil {
		return
	}
	port = newcig.Connection.Port
	path = newcig.Connection.Path
	user = newcig.Connection.User
	pwd = newcig.Connection.Password
	showlog = newcig.Options.ShowDebugLog

	//check = flag.String("crt", "", "检查refreshToken是否过期")

	var address string
	if runtime.GOOS == "windows" {
		address = ":" + port
	} else {
		address = "0.0.0.0:" + port
	}
	NewToken := aliyun.RefreshToken(newcig.Connection.RefreshToken)
	panClient := aliyunpan.NewPanClient(*NewToken, aliyunpan.AppLoginToken{})
	ui, _ := panClient.GetUserInfo()
	configs := model.Config{
		RefreshToken: NewToken.RefreshToken,
		Token:        NewToken.AccessToken,
		DriveId:      ui.FileDriveId,
		ExpireTime:   time.Now().Unix() + int64(NewToken.ExpiresIn),
	}

	fs := &webdav.Handler{
		Prefix:     "/",
		FileSystem: webdav.Dir(path),
		LockSystem: webdav.NewMemLS(),
		Config:     configs,
	}

	vlog.Infof("当前用户：%s UserId: %s", ui.UserName, ui.UserId)
	vlog.Infof("WebDAV服务已启动，地址：http://%s", address)
	vlog.Infof("用户名：%s", user)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		// 获取用户名/密码
		username, password, ok := req.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		//	 验证用户名/密码
		if username != user || password != pwd {
			http.Error(w, "WebDAV: need authorized!", http.StatusUnauthorized)
			return
		}

		// Add CORS headers before any operation so even on a 401 unauthorized status, CORS will work.

		w.Header().Set("Access-Control-Allow-Origin", "*")

		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE")

		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if req.Method == "GET" && strings.HasPrefix(req.URL.Path, fs.Prefix) {
			info, err := fs.FileSystem.Stat(context.TODO(), strings.TrimPrefix(req.URL.Path, fs.Prefix))
			if err == nil && info.IsDir() {
				req.Method = "PROPFIND"

				if req.Header.Get("Depth") == "" {
					req.Header.Add("Depth", "1")
				}
			}
		}
		if showlog {
			fmt.Println(req.URL)
			fmt.Println(req.Method)
		}
		fs.ServeHTTP(w, req)
		UpdateRefreshToken(fs.Config.RefreshToken)
	})
	http.ListenAndServe(address, nil)
}

func readConfig() (config, error) {
	c := config{}
	c.Connection.Port = "8085"
	c.Connection.Path = "./"
	c.Connection.User = "admin"
	c.Connection.Password = "123456"
	c.Connection.RefreshToken = "refresh_token"
	c.Options.ShowDebugLog = false
	c.Options.IsMobile = true
	if _, err := os.Stat("config.toml"); os.IsNotExist(err) {
		data, err := toml.Marshal(c)
		if err != nil {
			return c, fmt.Errorf("failed encoding default config: %v", err)
		}
		if err := os.WriteFile("config.toml", data, 0644); err != nil {
			return c, fmt.Errorf("failed creating config: %v", err)
		}
		return c, nil
	}
	data, err := os.ReadFile("config.toml")
	if err != nil {
		return c, fmt.Errorf("error reading config: %v", err)
	}
	if err := toml.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("error decoding config: %v", err)
	}
	return c, nil
}
