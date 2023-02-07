package aliyun

import (
	"encoding/json"
	"errors"
	"github.com/gngpp/vlog"
	"go-aliyun-webdav/aliyun/model"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	CONFIRMED = "CONFIRMED"
	EXPIRED   = "EXPIRED"
	NEW       = "NEW"
)

const WebLoginTokenApi = "https://passport.aliyundrive.com/newlogin/qrcode/generate.do?appName=aliyun_drive&fromSite=52&appName=aliyun_drive&appEntrance=web&isMobile=false&lang=zh_CN"
const MobileLoginTokenApi = "https://passport.aliyundrive.com/newlogin/qrcode/generate.do?appName=aliyun_drive&isMobile=true"

type Api struct {
	qrCodeCK    *model.QueryQrCodeCKForm
	generateMux sync.Mutex
	queryMux    sync.Mutex
	isMobile    bool
}

func NewApi(isMobile bool) *Api {
	return &Api{
		isMobile: isMobile,
	}
}

func (_this *Api) GetGeneratorQrCodeContent() (*model.QueryQrCodeCKForm, error) {
	var globalErr error
	var resp *http.Response
	if _this.isMobile {
		resp, globalErr = http.Get(MobileLoginTokenApi)
	} else {
		resp, globalErr = http.Get(WebLoginTokenApi)
	}
	if globalErr != nil {
		vlog.Errorf("resp qrcode error: %v", globalErr.Error())
		return nil, globalErr
	}
	body := resp.Body
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			vlog.Errorf("system error: %v\n", err)
		}
	}(body)
	bytes, _ := ioutil.ReadAll(body)
	q := model.GeneratorQrCodeResult{}
	globalErr = json.Unmarshal(bytes, &q)
	if globalErr != nil {
		vlog.Errorf("convert body error: %v\n", globalErr)
		return nil, globalErr
	}

	content := q.Content
	if content.Success {
		result := model.QueryQrCodeCKForm{
			T:           strconv.FormatInt(content.Data.T, 10),
			CodeContent: content.Data.CodeContent,
			CK:          content.Data.Ck,
		}
		_this.generateMux.Lock()
		_this.qrCodeCK = &result
		_this.generateMux.Unlock()

		return _this.qrCodeCK, nil
	}
	vlog.Debug(content.Data.TitleMsg)
	return nil, errors.New(content.Data.TitleMsg)
}

func (_this *Api) GetQrCodeCK() *model.QueryQrCodeCKForm {
	return _this.qrCodeCK
}

func CreateClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives:   false,
			DialContext:         dialer.DialContext,
			IdleConnTimeout:     10 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
}

func (_this *Api) GetQueryQrCodeResult() (*model.QueryQrCodeResult, bool) {
	values := url.Values{}
	_this.generateMux.Lock()
	values.Add("t", _this.qrCodeCK.T)
	values.Add("ck", _this.qrCodeCK.CK)
	_this.generateMux.Unlock()

	ticker := time.NewTicker(time.Second)
	q := &model.QueryQrCodeResult{}
	for {
		<-ticker.C
		// 默认keep-alive
		req, err := http.NewRequest("POST", "https://passport.aliyundrive.com/newlogin/qrcode/query.do?appName=aliyun_drive&fromSite=52&_bx-v=2.2.3", strings.NewReader(values.Encode()))
		req.Header.Add("content-type", "application/x-www-form-urlencoded")
		if err != nil {
			return nil, false
		}
		defer req.Body.Close()
		defaultClient := CreateClient()
		response, err := defaultClient.Do(req)
		if err != nil {
			return nil, false
		}

		if err != nil {
			vlog.Debugf("query qrcode request error:\n%v", err)
			return nil, false
		}
		var globalErr error
		body := response.Body

		bytes, _ := ioutil.ReadAll(body)
		//vlog.Infof("query qrcode row json result:\n%v", string(bytes))
		_ = body.Close()
		_this.queryMux.Lock()
		globalErr = json.Unmarshal(bytes, q)
		_this.queryMux.Unlock()

		if globalErr != nil {
			vlog.Errorf("convert body error:\n%v", globalErr)
			return nil, false
		}
		if q.Content.Success {
			if q.Content.Data.QrCodeStatus == CONFIRMED {
				ticker.Stop()
				break
			}
		}
	}
	return q, true
}
