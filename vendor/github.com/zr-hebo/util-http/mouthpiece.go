package easyhttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ErrorStatus 返回包含不同状态的错误信息
type ErrorStatus interface {
	Status() int
}

// Mouthpiece 返回response的结果，记录错误日志
type Mouthpiece struct {
	resp    http.ResponseWriter
	Err     error  `json:"-"`
	Message string `json:"message"`
	Status  int    `json:"status"`

	Data interface{} `json:"data,omitempty"`
}

// NewMouthpiece 创建传话筒
func NewMouthpiece(resp http.ResponseWriter) (mp *Mouthpiece) {
	mp = new(Mouthpiece)
	mp.resp = resp
	mp.Status = -1
	return
}

// SetError 设置错误信息
func (mp *Mouthpiece) SetError(err error) {
	mp.Err = err
}

// Convey 将执行结果使用http response返回
func (mp *Mouthpiece) String() (strContent string) {
	jsonContent, err := json.Marshal(mp)
	if err != nil {
		strContent = err.Error()
	}

	strContent = string(jsonContent)
	strContent = fmt.Sprintf("准备Response：%s", strContent)
	return
}

// Convey 将执行结果使用http response返回
func (mp *Mouthpiece) Convey() (err error) {
	if mp.Err != nil {
		if se, ok := mp.Err.(ErrorStatus); ok {
			mp.Status = se.Status()

		} else {
			mp.Status = -1
		}
		mp.Message = mp.Err.Error()

	} else {
		mp.Status = 0
		mp.Message = "OK"
	}

	err = Response(mp.resp, mp)
	return
}

// Response 将结果打包成json返回给http
func Response(resp http.ResponseWriter, result interface{}) (err error) {
	respMsg, err := json.Marshal(result)
	if err != nil {
		return
	}

	respStr := string(respMsg)
	replacer := strings.NewReplacer(
		"\\u0026", "&",
		"\\u003c", "<",
		"\\u003e", ">")
	respMsg = []byte(replacer.Replace(respStr))

	_, err = resp.Write(respMsg)
	return
}
