# ego-gateway
EGO 动态HTTP转gRPC
* 通过gRPC的反射接口获取到服务的元数据信息
* 通过gRPC的resolver拉取后端IP列表
    * 直连[支持]
    * dns[支持]
    * etcd[todo]
    * k8s[todo]
* 负载均衡策略
    * rr[支持、默认]
    * p2c[todo]
* 支持修改路由别名[todo]    
* 支持动态修改配置添加节点[支持]

```shell
## 启动gRPC服务
cd examples/server
export EGO_DEBUG=true && go run main.go --config=config/config.toml

## 启动gRPC gateway服务
cd 主目录
export EGO_DEBUG=true && go run main.go --config=config/config.toml

## 访问测试
curl -XPOST -d '{"name":"grpc proxy"}' -i -H 'X-Proxy-Server: test' -H 'Content-Type: application/json'  http://127.0.0.1:9001/helloworld.Greeter/SayHello
```
