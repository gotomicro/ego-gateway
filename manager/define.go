package manager

import (
	"bytes"
	"unicode"

	"google.golang.org/grpc/codes"
)

type Service struct {
	// Name 唯一应用名
	Name string
	// 服务注册方式，如：dns:///appname 等
	Addr string
}

type Data struct {
	Body []byte
}

func (d *Data) GetBody() []byte {
	return d.Body
}

type Result struct {
	Data map[string]interface{} `json:"data"`
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`

	GRPCCode codes.Code `json:"-"`
	GRPCMsg  string     `json:"-"`
}

func (r *Result) HttpStatus() int {
	return convertCode(r.GRPCCode)
}

func (r *Result) GetData() interface{} {
	return r.Data
}

// CamelString 将路径从小写下划线改成大小写驼峰格式
func CamelString(path []byte) string {
	idx := bytes.LastIndexByte(path, '.')
	needConvert := idx > 0 && path[idx+1] >= 'a' && path[idx+1] <= 'z'
	if !needConvert {
		return string(path)
	}
	l := len(path)
	fl := l
	for i, j := idx, idx; i < l; {
		switch path[i] {
		case '.', '/', '_':
			if path[i] != '_' {
				path[j] = path[i]
				j++
			} else {
				fl--
			}
			i++
			if i == l {
				continue
			}
			path[j] = byte(unicode.ToUpper(rune(path[i])))
		default:
			path[j] = path[i]
		}
		j++
		i++
	}
	return string(path[0:fl])
}
