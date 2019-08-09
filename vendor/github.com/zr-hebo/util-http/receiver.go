package easyhttp

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// NewUnpacker 创建request参数解析器
func NewUnpacker(
	req *http.Request, receiver interface{}, logger Logger) (
	unpacker *Unpacker) {
	unpacker = new(Unpacker)
	unpacker.req = req
	unpacker.receiver = receiver
	unpacker.logger = logger

	return
}

// Unpack 将request中的请求参数解析到结构体中
func (u *Unpacker) Unpack() (err error) {
	if u.receiver == nil {
		return
	}

	if u.req.Method == "GET" {
		err = u.unpackGetParams()

	} else if u.req.Method == "POST" {
		err = u.unpackJSONParams()
	}

	return
}

// unpackGetParams 解析GET参数到接收器中
func (u *Unpacker) unpackGetParams() (err error) {
	if err = u.req.ParseForm(); err != nil {
		return err
	}

	rt := reflect.TypeOf(u.receiver)
	rv := reflect.ValueOf(u.receiver)

	if rt.Kind() == reflect.Ptr && rt.Elem().Kind() == reflect.Struct {
		return u.unpackFieldFromParams(rv.Elem(), "")
	}

	return fmt.Errorf("解析参数类型需要为 *struct ，传入的是 %s", rt.String())

}

func (u *Unpacker) getFormVal(key string) (val string) {
	vars := mux.Vars(u.req)
	val = u.req.FormValue(key)
	if val == "" && vars != nil {
		val = vars[key]
	}

	return
}

func (u *Unpacker) unpackFieldFromParams(
	field reflect.Value, varName string) (err error) {
	rv := field
	rt := field.Type()

	switch rt.Kind() {
	case reflect.Ptr:
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		u.unpackFieldFromParams(rv.Elem(), varName)

	case reflect.Struct:
		for i := 0; i < rt.NumField(); i++ {
			tf := rt.Field(i)
			key := tf.Tag.Get("json")
			if key == "" {
				key = strings.ToLower(tf.Name)
			}

			val := u.getFormVal(key)
			if u.logger != nil {
				u.logger.Debugf("key:%v value:%v", key, val)
			}

			rfv := rv.Field(i)
			switch rfv.Kind() {
			case reflect.Ptr:
				if rfv.IsNil() {
					rfv.Set(reflect.New(rfv.Type().Elem()))
				}
				err = u.unpackFieldFromParams(rfv.Elem(), key)

			case reflect.Struct:
				err = u.unpackFieldFromParams(rfv, key)

			case reflect.Array, reflect.Map:
				continue

			default:
				if len(val) < 1 {
					continue
				}

				err = populate(rfv, val)
			}

			if err != nil {
				break
			}
		}

	case reflect.Array, reflect.Map:
		break

	default:
		val := u.getFormVal(varName)
		if len(val) > 0 {
			err = populate(field, val)
		}

	}

	return
}

func populate(rv reflect.Value, value string) (err error) {
	switch rv.Kind() {
	case reflect.String:
		rv.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		rv.SetInt(i)

	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		rv.SetBool(b)

	case reflect.Float32:
		f, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return err
		}
		rv.SetFloat(f)

	case reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		rv.SetFloat(f)

	default:
		return fmt.Errorf("unsupported kind %s", rv.Type().String())
	}

	return
}

func (u *Unpacker) unpackJSONParams() (err error) {
	if u.req == nil || u.req.Body == nil {
		return fmt.Errorf("request body 为空")
	}

	if u.req.Body != nil {
		defer u.req.Body.Close()
	}

	body, err := ioutil.ReadAll(u.req.Body)
	if err != nil {
		return
	}

	if u.logger != nil {
		u.logger.Info(string(body))
	}

	if len(body) > 0 {
		return json.Unmarshal(body, u.receiver)
	}

	return
}

func stringSliceContent(strs []string, str string) bool {
	for _, s := range strs {
		if s == str {
			return true
		}
	}

	return false
}
