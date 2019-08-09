package easyhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// NewGetSender 创建get请求发送器
func NewGetSender(
	url string, headers map[string]string, getParams map[string]string,
	receiver interface{}, logger Logger) (sender *GetSender) {
	sender = new(GetSender)
	sender.url = url
	sender.headers = headers
	sender.getParams = getParams
	sender.receiver = receiver
	sender.logger = logger

	return
}

// NewPostSender 创建post请求发送器
func NewPostSender(
	url string, headers map[string]string, postData interface{},
	receiver interface{}, logger Logger) (sender *PostSender) {
	sender = new(PostSender)
	sender.url = url
	sender.headers = headers
	sender.postData = postData
	sender.receiver = receiver
	sender.logger = logger

	return
}

// AddHeader 在http请求中添加header
func (gs *GetSender) AddHeader(k, v string) {
	if gs.headers == nil {
		gs.headers = make(map[string]string)
	}

	gs.headers[k] = v
}

// Request 发送get请求
func (gs *GetSender) Request() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf(
				"使用 GetSender 发送请求的时候失败 <-- %s", err.Error())

			if gs.logger != nil {
				gs.logger.Error(err.Error())
			}
		}
	}()

	req, err := gs.fillRequest()
	if err != nil {
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return
	}

	bodyContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return gs.resolveResp(bodyContent)
}

// Request 发送post请求
func (ps *PostSender) Request() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf(
				"使用PostSender发送请求的时候失败 <-- %s", err.Error())

			if ps.logger != nil {
				ps.logger.Error(err.Error())
			}
		}
	}()

	req, err := ps.fillRequest()
	if err != nil {
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return
	}

	bodyContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return ps.resolveResp(bodyContent)
}

func (gs *GetSender) fillRequest() (fatReq *http.Request, err error) {
	if gs.getParams != nil {
		queryParams := url.Values{}
		for k, v := range gs.getParams {
			queryParams.Add(k, v)
		}

		gs.url = fmt.Sprintf("%s?%s", gs.url, queryParams.Encode())
	}

	headerBytes, _ := json.Marshal(gs.headers)
	if gs.logger != nil {
		gs.logger.Infof("Ready get: '%s' with header: '%s'",
			gs.url, string(headerBytes))
	}

	fatReq, err = http.NewRequest("GET", gs.url, nil)
	if err != nil {
		return
	}

	fatReq.Header.Set("Accept-Encoding", "")
	for k, v := range gs.headers {
		fatReq.Header.Set(k, v)
	}

	return

}

func (ps *PostSender) fillRequest() (fatReq *http.Request, err error) {
	postBytes, err := json.Marshal(ps.postData)
	if err != nil {
		return
	}

	headerBytes, _ := json.Marshal(ps.headers)
	if ps.logger != nil {
		ps.logger.Infof("Ready post to: '%s' with header: '%s' and data: '%s'",
			ps.url, string(headerBytes), string(postBytes))
	}

	fatReq, err = http.NewRequest("POST", ps.url, bytes.NewReader(postBytes))
	if err != nil {
		return
	}

	for k, v := range ps.headers {
		fatReq.Header.Set(k, v)
	}

	return
}

func (gs *GetSender) resolveResp(respContent []byte) (err error) {
	if gs.logger != nil {
		gs.logger.Infof("get from: '%s' get response: '%s'",
			gs.url, string(respContent))
	}

	gs.rawResp = respContent
	if gs.receiver != nil {
		return json.Unmarshal(respContent, gs.receiver)
	}

	return
}

// GetRawResp 获取原始response的数据
func (gs *GetSender) GetRawResp() (rawResp []byte) {
	return gs.rawResp
}
