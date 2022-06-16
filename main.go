package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/wjpxxx/letgo/encry"
)

func main() {
	if len(os.Args) == 2 {
		if os.Args[1] == "local" {
			local()
		} else if os.Args[1] == "remote" {
			remote()
		}
	} else {
		local()
	}

}

var remoteLocalIp = "0.0.0.0"  //服务器内网IP
var remoteIp = "0.0.0.0"       //服务器外网IP
var BS = 4096

func remote() {
	fmt.Println("启动远程端")
	listen, err := net.Listen("tcp", remoteLocalIp+":4567")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer listen.Close()
	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("accept", err.Error())
			continue
		}
		go handle(conn)
	}
}

func local() {
	fmt.Println("启动本地端")
	listen, err := net.Listen("tcp", "0.0.0.0:1234")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer listen.Close()
	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("accept", err.Error())
			continue
		}
		go handleLocal(conn)
	}
}
func handleLocal(conn net.Conn) {
	defer conn.Close()
	dist, err := net.Dial("tcp", remoteIp+":4567")
	if err != nil {
		fmt.Println("local连接目标失败", err.Error())
		return
	}
	closed := make(chan bool, 2)
	go LocalProxyUp(conn, dist, closed)
	go LocalProxyDown(dist, conn, closed)
	<-closed
	fmt.Println("退出程序")
}

func handle(conn net.Conn) {
	defer conn.Close()
	nf := UnPack(conn)
	if nf == nil {
		return
	}
	//fmt.Println("接收加密",nf[:n])
	//解密
	buffer := Output(nf)
	if buffer == nil {
		return
	}
	//fmt.Println("========================请求1=============================")
	//fmt.Println(buffer, "长度:", len(buffer))
	//buffer:=nf[:n]
	//fmt.Println("recv",string(buffer))
	if bytes.IndexByte(buffer, '\n') == -1 {
		return
	}
	line := buffer[:bytes.IndexByte(buffer, '\n')]
	var method, host, address string
	fmt.Sscanf(string(line), "%s%s", &method, &host)
	fmt.Println("method", method, "host:", host)
	hostPortURL, err := url.Parse(host)
	if err != nil {
		address=host
	} else {
		if hostPortURL.Opaque == "443" { //https访问
			address = hostPortURL.Scheme + ":443"
		} else { //http访问
			if strings.Index(hostPortURL.Host, ":") == -1 { //host不带端口， 默认80
				address = hostPortURL.Host + ":80"
			} else {
				address = hostPortURL.Host
			}
		}
	}
	//fmt.Println("连接地址:", address)
	dist, err := net.Dial("tcp", address)
	if err != nil {
		//fmt.Println("remote连接目标失败", err.Error())
		return
	}
	if method == "CONNECT" {
		//bf :=[]byte(encry.DesEncryptCBC( "HTTP/1.1 200 Connection established\r\n\r\n", key))
		bf := Input([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		conn.Write(Pack(bf))
	} else {
		//请求真正网站
		dist.Write(buffer)
	}
	closed := make(chan bool, 2)
	go RemoteProxyUp(conn, dist, closed)
	go RemoteProxyDown(dist, conn, closed)
	<-closed
	fmt.Println("退出程序")
}

var key = "12345678"

//本地传到上行
func LocalProxyUp(from net.Conn, to net.Conn, closed chan bool) {
	buffer := make([]byte, BS)
	for {
		n, err := from.Read(buffer)
		if err != nil {
			//fmt.Println("from 读取出错",err.Error())
			break
		}
		//md5 := encry.Md5(string(buffer[:n]))
		//fmt.Println("========================请求:" + md5 + "=============================")
		//fmt.Println(buffer[:n], "长度:", len(buffer[:n]))
		if n > 0 {
			//加密
			bf := Input(buffer[:n])
			//bf := Input([]byte("hello world my neiber"))
			//fmt.Println("加密后",bf)
			//bf:=buffer[:n]
			_, err = to.Write(Pack(bf))
			if err != nil {
				//fmt.Println("to 写入出错",err.Error())
				//fmt.Println("=======================写入出错:" + md5 + "=======================")
				break
			}
			//fmt.Println("=======================发送成功:" + md5 + "=======================")
		}

	}
	closed <- true
}

//本地下行接收
func LocalProxyDown(from net.Conn, to net.Conn, closed chan bool) {
	for {
		nf := UnPack(from)
		if nf == nil {
			//fmt.Println("from 读取出错",err.Error())
			break
		}
		//解密
		bf := Output(nf)
		//bf:=buffer[:n]
		//fmt.Println("========================响应=============================")
		//fmt.Println(fmt.Sprintf("%s", bf))
		_, err := to.Write(bf)
		if err != nil {
			//fmt.Println("to 写入出错",err.Error())
			break
		}
	}
	closed <- true
}

//远程接收后传到上行
func RemoteProxyUp(from net.Conn, to net.Conn, closed chan bool) {
	for {
		nf := UnPack(from)
		if nf == nil {
			//fmt.Println("from 读取出错",err.Error())
			break
		}
		//解密
		bf := Output(nf)
		//fmt.Println("========================请求2=============================")
		//fmt.Println(bf, "长度:", len(bf))
		//fmt.Println(fmt.Sprintf("%s", bf))
		//fmt.Println("解密后",bf)
		//bf:=buffer[:n]
		//请求真正网站
		_, err := to.Write(bf)
		if err != nil {
			//fmt.Println("to 写入出错",err.Error())
			break
		}

	}
	closed <- true
}

//远程下行接收
func RemoteProxyDown(from net.Conn, to net.Conn, closed chan bool) {
	buffer := make([]byte, BS)
	for {
		n, err := from.Read(buffer)
		if err != nil {
			//fmt.Println("from 读取出错",err.Error())
			break
		}
		//fmt.Println("========================响应=============================")
		//fmt.Println(fmt.Sprintf("%s", buffer[:n]))
		bf := Input(buffer[:n])
		//fmt.Println(n)
		//bf := Input([]byte("hello world in nebier"))
		//bf:=buffer[:n]
		_, err = to.Write(Pack(bf))
		if err != nil {
			//fmt.Println("to 写入出错",err.Error())
			break
		}
	}
	closed <- true
}

//加密
func Input(in []byte) []byte {
	return []byte(encry.DesEncryptCBC(string(in), key))
	//return in
}

//加密
func Output(in []byte) []byte {
	return []byte(encry.DesDecryptCBC(string(in), key))
	//return in
}

//封包
func Pack(idata []byte) []byte {
	var ln = int32(len(idata))
	var pkg = new(bytes.Buffer)
	binary.Write(pkg, binary.LittleEndian, ln)
	binary.Write(pkg, binary.LittleEndian, idata)
	//fmt.Println("发送:",ln)
	bt := pkg.Bytes()
	//fmt.Println("发送",bt)
	return bt
}

//解包
func UnPack(conn net.Conn) []byte {
	var buf []byte = make([]byte, 4)
	conn.Read(buf)
	//fmt.Println("====================x:",x,xe,"================")
	buf2 := bytes.NewBuffer(buf)
	var ln int32
	binary.Read(buf2, binary.LittleEndian, &ln)
	//fmt.Println(fmt.Sprintf("包理论大小:%d", ln))
	var pack []byte
	var has int32 = 0
	for {
		tmp := make([]byte, ln-has)
		n, err := conn.Read(tmp)
		if err != nil {
			//fmt.Println("===================读取过长的出错拉=============",ln,has, err.Error())
			return nil
		}
		if n > 0 {
			pack = append(pack, tmp[:n]...)
			has += int32(n)
		}
		if has == ln {
			break
		}
	}
	//fmt.Println(fmt.Sprintf("理论长度:%d,实际长度:%d,包长度:%d", ln, has, len(pack)))
	//fmt.Println(fmt.Sprintf("读取大小:%d", has))
	//fmt.Println("接收",pack)
	if ln == 0 {
		return nil
	}
	if has != ln {
		return nil
	}
	//fmt.Println("抓包返回了")
	//fmt.Println("接收",append(buf,pack...))
	//var pkg = new(bytes.Buffer)
	//binary.Write(pkg, binary.LittleEndian, pack)
	//fmt.Println("发送:",ln)
	//bt := pkg.Bytes()
	return pack
}
