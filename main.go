package main

import (
	"crypto/aes"
	"crypto/cipher"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Connection 表示一个TCP连接和WebSocket连接的组合
type Connection struct {
	UUID     string
	WsConn   *websocket.Conn
	TcpConn  net.Conn
	IsClosed bool
}

// 全局连接映射和锁
var (
	connections = make(map[string]*Connection)
	connMutex   sync.RWMutex
)

// 添加连接到映射
func addConnection(conn *Connection) {
	connMutex.Lock()
	defer connMutex.Unlock()
	connections[conn.UUID] = conn
}

// 根据UUID获取连接
func getConnection(uuid string) (*Connection, bool) {
	connMutex.RLock()
	defer connMutex.RUnlock()
	conn, exists := connections[uuid]
	return conn, exists
}

// 删除连接并关闭相关资源
func removeConnection(uuid string) {
	connMutex.Lock()
	defer connMutex.Unlock()

	if conn, exists := connections[uuid]; exists {
		if conn.WsConn != nil {
			conn.WsConn.Close()
		}
		if conn.TcpConn != nil {
			conn.TcpConn.Close()
		}
		conn.IsClosed = true
		delete(connections, uuid)
	}
}

// 服务端处理WebSocket连接
func handleWebSocketConnection(wsConn *websocket.Conn) {
	defer wsConn.Close()

	// 读取客户端发送的UUID
	_, message, err := wsConn.ReadMessage()
	if err != nil {
		log.Printf("读取UUID失败: %v", err)
		return
	}

	clientUUID := string(message)
	log.Printf("新连接: %s", clientUUID)

	// 创建新连接对象
	conn := &Connection{
		UUID:     clientUUID,
		WsConn:   wsConn,
		IsClosed: false,
	}

	addConnection(conn)
	defer removeConnection(clientUUID)

	// 建立到内网的TCP连接
	tcpAddr := flag.Lookup("target").Value.String()
	tcpConn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		log.Printf("连接到目标TCP服务失败: %v", err)
		return
	}

	conn.TcpConn = tcpConn
	defer tcpConn.Close()

	// 启动双向数据转发
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer removeConnection(clientUUID)

		buffer := make([]byte, 16384) // 16KB缓冲区
		for {
			if conn.IsClosed {
				return
			}

			n, err := tcpConn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					log.Printf("读取TCP连接失败: %v", err)
				}
				return
			}

			if err := wsConn.WriteMessage(websocket.BinaryMessage, buffer[:n]); err != nil {
				log.Printf("写入WebSocket失败: %v", err)
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer removeConnection(clientUUID)

		for {
			if conn.IsClosed {
				return
			}

			_, message, err := wsConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("读取WebSocket失败: %v", err)
				}
				return
			}

			if _, err := tcpConn.Write(message); err != nil {
				log.Printf("写入TCP连接失败: %v", err)
				return
			}
		}
	}()

	wg.Wait()
}

// 服务端启动函数
func startServer(listenAddr, targetAddr string) {
	log.Printf("启动服务端: %s -> %s", listenAddr, targetAddr)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket升级失败: %v", err)
			return
		}

		go handleWebSocketConnection(wsConn)
	})

	err := http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}

// 转换URL: ws://10.69.12.112:12333 -> wss://vpncas.ahut.edu.cn/ws/ws-12333/77726476706e69737468656265737421+aes_cfb_128bit_nopadding(10.69.12.112,wrdvpnisthebest!)
func transformURL(u *url.URL) string {
	// 创建新URL对象
	newURL := &url.URL{
		Scheme: "wss",
		Host:   "vpncas.ahut.edu.cn",
	}

	// 提取原始协议和端口
	protocol := u.Scheme
	port := "12333" // 默认端口

	if strings.Contains(u.Host, ":") {
		_, p, err := net.SplitHostPort(u.Host)
		if err == nil {
			port = p
		}
	} else if protocol == "wss" || protocol == "https" {
		port = "443"
	}

	// 构建协议-端口路径段
	protoPortSegment := fmt.Sprintf("%s-%s", protocol, port)

	// 提取原始IP地址
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host // 如果没有端口，直接使用主机名
	}

	// 加密IP地址
	encryptedIP := encryptIP(host)

	// 构建完整路径
	newURL.Path = fmt.Sprintf("/%s/77726476706e69737468656265737421%s", protoPortSegment, encryptedIP)

	return newURL.String()
}

// 使用AES-128-CFB加密IP地址，密钥为wrdvpnisthebest!
func encryptIP(ip string) string {
	key := []byte("wrdvpnisthebest!")

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalf("创建加密块失败: %v", err)
	}

	iv := key
	ciphertext := make([]byte, len(ip))

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext, []byte(ip))

	return fmt.Sprintf("%x", ciphertext)
}

// 客户端连接函数
func startClient(serverAddr, localAddr, cookie string) {
	// 解析原始服务器URL
	u, err := url.Parse(serverAddr)
	if err != nil {
		// 尝试补全协议
		u = &url.URL{
			Scheme: "ws",
			Host:   serverAddr,
		}
	}
	log.Printf("启动客户端: 本地监听 %s -> 服务器 %s", localAddr, transformURL(u))

	// 创建本地TCP监听
	tcpListener, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Fatalf("启动本地TCP监听失败: %v", err)
	}
	defer tcpListener.Close()

	log.Printf("本地TCP服务器已启动: %s", localAddr)

	for {
		// 接受本地连接
		localConn, err := tcpListener.Accept()
		if err != nil {
			log.Printf("接受本地连接失败: %v", err)
			continue
		}

		// 为每个连接生成唯一ID
		clientUUID := fmt.Sprintf("%d", time.Now().UnixNano())

		// 建立到服务器的WebSocket连接
		headers := http.Header{}
		if cookie != "" {
			headers.Add("Cookie", cookie)
		}

		//// 解析原始服务器URL
		//u, err := url.Parse(serverURL)
		//if err != nil {
		//	log.Fatalf("解析服务器URL失败: %v", err)
		//}

		// 转换为代理服务器URL
		proxyURL := transformURL(u)

		wsConn, _, err := websocket.DefaultDialer.Dial(proxyURL, headers)
		if err != nil {
			log.Printf("连接到服务器失败: %v", err)
			localConn.Close()
			continue
		}

		// 发送UUID到服务器
		if err := wsConn.WriteMessage(websocket.TextMessage, []byte(clientUUID)); err != nil {
			log.Printf("发送UUID失败: %v", err)
			wsConn.Close()
			localConn.Close()
			continue
		}

		// 创建连接对象
		conn := &Connection{
			UUID:     clientUUID,
			WsConn:   wsConn,
			TcpConn:  localConn,
			IsClosed: false,
		}

		addConnection(conn)

		// 启动双向数据转发
		go func() {
			defer removeConnection(clientUUID)

			buffer := make([]byte, 16384) // 16KB缓冲区
			for {
				if conn.IsClosed {
					return
				}

				n, err := localConn.Read(buffer)
				if err != nil {
					if err != io.EOF {
						log.Printf("读取本地TCP连接失败: %v", err)
					}
					return
				}

				if err := wsConn.WriteMessage(websocket.BinaryMessage, buffer[:n]); err != nil {
					log.Printf("写入WebSocket失败: %v", err)
					return
				}
			}
		}()

		go func() {
			defer removeConnection(clientUUID)

			for {
				if conn.IsClosed {
					return
				}

				_, message, err := wsConn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Printf("读取WebSocket失败: %v", err)
					}
					return
				}

				if _, err := localConn.Write(message); err != nil {
					log.Printf("写入本地TCP连接失败: %v", err)
					return
				}
			}
		}()
	}
}

func main() {
	// 定义命令行参数
	mode := flag.String("mode", "server", "运行模式: server 或 client")
	listen := flag.String("listen", ":12333", "服务器监听地址")
	target := flag.String("target", "127.0.0.1:80", "目标地址（服务端使用）")
	server := flag.String("server", "", "服务器地址（客户端使用）")
	cookie := flag.String("cookie", "", "要发送的Cookie（客户端使用）")

	flag.Parse()

	// **处理listen参数：自动补全冒号**
	//listenAddr := *listen
	//if !strings.Contains(listenAddr, ":") {
	//	listenAddr = "0.0.0.0:" + listenAddr // 补全为:端口号格式
	//}

	// 校验监听地址格式
	//_, err := net.ResolveTCPAddr("tcp", listenAddr)
	//if err != nil {
	//	log.Fatalf("无效的监听地址 %s: %v", listenAddr, err)
	//}

	// 根据模式启动服务端或客户端
	if *mode == "server" {
		startServer(*listen, *target)
	} else if *mode == "client" {
		startClient(*server, *listen, *cookie)
	} else {
		log.Fatalf("未知模式: %s", *mode)
	}
}
