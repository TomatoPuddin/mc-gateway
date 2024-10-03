package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
)

type Config struct {
	Port    int               `json:"port"`
	Hosts   map[string]string `json:"hosts"`
	Default string            `json:"default"`
}

var config Config

func loadConfig() error {
	file, err := os.Open("config.json")
	if err != nil {
		return err
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	return json.Unmarshal(byteValue, &config)
}

func main() {
	if err := loadConfig(); err != nil {
		panic(err)
	}

	// 监听TCP端口
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	fmt.Println("Listening on :25565")

	for {
		// 接受传入的连接
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
			continue
		}
		// 处理连接
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	// 确保连接关闭
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
		return
	}
	mc_host := GetMcHost(buf[:n])
	host, ok := config.Hosts[mc_host]
	if !ok {
		host = config.Default
	}

	client, err := net.Dial("tcp", host)
	if err != nil {
		fmt.Println("Error dialing:", err.Error())
		return
	}
	defer client.Close()

	client.Write(buf[:n])

	var wg sync.WaitGroup
	wg.Add(2)

	go handleRead(client, conn, &wg)
	go handleWrite(client, conn, &wg)

	wg.Wait()
}

func handleRead(srv, cli net.Conn, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		srv.Close()
		cli.Close()
	}()

	buf := make([]byte, 1024)

	for {
		n, err := srv.Read(buf)
		if err != nil {
			return
		}

		cli.Write(buf[:n])
	}
}

func handleWrite(srv, cli net.Conn, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		srv.Close()
		cli.Close()
	}()

	buf := make([]byte, 1024)

	for {
		n, err := cli.Read(buf)
		if err != nil {
			return
		}

		srv.Write(buf[:n])
	}
}

func GetMcHost(buf []byte) string {
	buf = buf[4:]
	host_len := buf[0]
	if len(buf)+1 < int(host_len) {
		return ""
	}

	return string(buf[1 : host_len+1])
}
