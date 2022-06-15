# 简单几行代码实现代理功能

代理分为服务端和客户端,服务端如果部署在国外,即可实现科学上网

var remoteLocalIp = "0.0.0.0"   //定义服务端内网IP
var remoteIp = "0.0.0.0"        //定义服务端外网IP


启动服务端

```sh
$ go run main.go remote
```

启动客户端

```sh
$ go run main.go local
```

注意仅提供给科学上网