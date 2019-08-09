package easyhttp

import "net/http"

// Logger 日志记录接口
type Logger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
}

// Unpacker request参数解析器
type Unpacker struct {
	req      *http.Request
	receiver interface{}
	logger   Logger
}

type baseSender struct {
	url      string
	headers  map[string]string
	logger   Logger
	receiver interface{}
	rawResp  []byte
}

// GetSender get请求发送器
type GetSender struct {
	baseSender
	getParams map[string]string
}

// PostSender post请求发送器
type PostSender struct {
	GetSender
	postData interface{}
}

// RespReceiver request结果接收器
type RespReceiver struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// NewRespReceiver 创建request结果接收器
func NewRespReceiver() (rr *RespReceiver) {
	rr = new(RespReceiver)
	rr.Status = -1
	return
}

// Paginator 分页显示结果收集器
type Paginator struct {
	Rows  interface{} `json:"rows"`
	Total int         `json:"total"`
}

// NewPaginator 创建分页显示结果收集器
func NewPaginator() (pgt *Paginator) {
	pgt = new(Paginator)
	pgt.Rows = make([]string, 0)
	return
}
