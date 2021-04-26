package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"

	"ego-gateway/manager"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/server/egin"
)

//  export EGO_DEBUG=true && go run main.go --config=config.toml
func main() {
	if err := ego.New().Serve(func() *egin.Component {
		managerClient := manager.Init()
		server := egin.Load("server.http").Build()
		server.NoRoute(func(ctx *gin.Context) {
			proxyServer := ctx.GetHeader("X-Proxy-Server")
			if proxyServer == "" {
				elog.Error("proxy server name empty")
				ctx.Writer.WriteHeader(http.StatusNotFound)
				return
			}
			contentType := ctx.GetHeader("Content-Type")
			cc := managerClient.GetGRPCProxyClient(proxyServer)
			if cc == nil {
				elog.Error("proxy server client empty")
				ctx.Writer.WriteHeader(http.StatusNotFound)
				return
			}
			var body []byte
			if ctx.Request.Method == "GET" || ctx.Request.Method == "DELETE" {
				//body = []byte(ctx.Request.URL.RequestURI())
			} else {
				buf, _ := ioutil.ReadAll(ctx.Request.Body)
				ctx.Request.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
				body = buf
			}
			path := ctx.Request.RequestURI
			path = strings.TrimPrefix(path, "/")
			path = strings.TrimSuffix(path, "/")
			paths := strings.Split(manager.CamelString([]byte(path)), "/")
			if len(paths) != 2 {
				elog.Error("proxy server service method empty")
				ctx.Writer.WriteHeader(http.StatusNotFound)
				return
			}

			service := paths[0]
			method := paths[1]

			var data = &manager.Data{
				Body: nil,
			}
			switch contentType {
			case "":
				data.Body = body
			case "application/json":
				data.Body = body
			default:
				elog.Error("unsupport content type", elog.String("contentType", contentType))
				ctx.Writer.WriteHeader(http.StatusNotFound)
				return
			}

			resp, err := cc.ProxyCall(ctx.Request.Context(), service, method, data)
			if err != nil {
				elog.Error("proxy call error", elog.FieldErr(err))
				ctx.Writer.WriteHeader(http.StatusInternalServerError)
				return
			}

			ctx.JSON(resp.HttpStatus(), resp)
			return
		})
		return server
	}()).Run(); err != nil {
		elog.Panic("startup", elog.FieldErr(err))
	}
}
