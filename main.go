package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/axgle/mahonia"
)

var (
	// application name
	Name = ""
	// application version string
	Version = ""
	// commit
	Commit = ""
	// build tags
	BuildTags = ""
	// application variable
	udpNetwork = "udp"
	tcpNetwork = "tcp"
	udpBufSize = 64 * 1024
)

var (
	logger = log.New(os.Stderr, "", 0)
)

func logf(f string, v ...interface{}) {
	if config.Verbose {
		logger.Output(2, fmt.Sprintf(f, v...))
	}
}

// Convert 处理字符编码转换
type Convert struct {
	conn net.Conn
	mu   sync.Mutex
}

func newConvert(c net.Conn) *Convert {
	return &Convert{conn: c}
}

func (convert *Convert) translate(p []byte, encoding string) []byte {
	srcDecoder := mahonia.NewDecoder(encoding)
	if srcDecoder == nil {
		return p // 如果解码器创建失败，返回原始数据
	}
	_, resBytes, _ := srcDecoder.Translate(p, true)
	return resBytes
}

func (convert *Convert) Write(p []byte) (n int, err error) {
	convert.mu.Lock()
	defer convert.mu.Unlock()

	switch runtime.GOOS {
	case "windows":
		resBytes := convert.translate(p, "gbk")
		m, err := convert.conn.Write(resBytes)
		if m != len(resBytes) {
			return m, err
		}
		return len(p), err
	default:
		return convert.conn.Write(p)
	}
}

func (convert *Convert) Read(p []byte) (n int, err error) {
	convert.mu.Lock()
	defer convert.mu.Unlock()
	return convert.conn.Read(p)
}

func (convert *Convert) Close() error {
	convert.mu.Lock()
	defer convert.mu.Unlock()
	return convert.conn.Close()
}

// Config 配置结构体
type Config struct {
	Help       bool
	Verbose    bool
	Listen     bool
	Port       int
	Network    string
	Web        bool
	Path       string
	Command    bool
	Host       string
	Timeout    time.Duration
	Retries    int
	SSL        bool
	KeepAlive  bool
	BufferSize int
}

var config Config

func init() {
	flag.IntVar(&config.Port, "p", 4000, "host port to connect or listen")
	flag.BoolVar(&config.Help, "help", false, "print this help")
	flag.BoolVar(&config.Verbose, "v", true, "verbose mode")
	flag.BoolVar(&config.Listen, "l", false, "listen mode")
	flag.BoolVar(&config.Command, "e", false, "shell mode")
	flag.BoolVar(&config.Web, "web", false, "web static server")
	flag.StringVar(&config.Path, "path", "public", "web static path")
	flag.StringVar(&config.Network, "n", "tcp", "network protocol")
	flag.StringVar(&config.Host, "h", "0.0.0.0", "host addr to connect or listen")
	flag.DurationVar(&config.Timeout, "timeout", 30*time.Second, "connection timeout")
	flag.IntVar(&config.Retries, "retries", 3, "connection retry attempts")
	flag.BoolVar(&config.SSL, "ssl", false, "use SSL/TLS")
	flag.BoolVar(&config.KeepAlive, "keepalive", true, "enable keep-alive")
	flag.IntVar(&config.BufferSize, "buffer", 4096, "buffer size for data transfer")
	flag.Usage = usage
	flag.Parse()
}

func usage() {
	fmt.Fprintf(os.Stderr, fmt.Sprintf(`name: %s 
version: %s
commit: %s
build_tags: %s
go: %s

usage: netcat [options] [host] [port]

options:
`, Name, Version, Commit, BuildTags,
		fmt.Sprintf("go version %s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)),
	)
	flag.PrintDefaults()
}

// validateConfig 验证配置参数
func validateConfig() error {
	if config.Port < 1 || config.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", config.Port)
	}

	if config.Network != "tcp" && config.Network != "udp" {
		return fmt.Errorf("unsupported network protocol: %s", config.Network)
	}

	if config.Retries < 0 {
		return fmt.Errorf("retries cannot be negative: %d", config.Retries)
	}

	if config.BufferSize < 1 {
		return fmt.Errorf("buffer size must be positive: %d", config.BufferSize)
	}

	return nil
}

// getShell 根据操作系统获取合适的shell
func getShell() string {
	switch runtime.GOOS {
	case "linux":
		return "/bin/sh"
	case "freebsd":
		return "/bin/csh"
	case "darwin":
		return "/bin/sh"
	case "windows":
		return "cmd.exe"
	default:
		return "/bin/sh"
	}
}

// setupConnection 设置连接参数
func setupConnection(conn net.Conn) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if config.KeepAlive {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)
		}
		tcpConn.SetLinger(0)
	}
}

// listenTCP TCP监听模式
func listenTCP(host string, port int, command bool) error {
	listenAddr := net.JoinHostPort(host, strconv.Itoa(port))

	// 创建监听器
	var listener net.Listener
	var err error

	if config.SSL {
		// 简单的自签名证书，实际使用中应该提供真实的证书
		cert, err := generateSelfSignedCert()
		if err != nil {
			return fmt.Errorf("failed to generate certificate: %v", err)
		}

		config := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		listener, err = tls.Listen("tcp", listenAddr, config)
	} else {
		listener, err = net.Listen("tcp", listenAddr)
	}

	if err != nil {
		return fmt.Errorf("listen failed: %v", err)
	}
	defer listener.Close()

	logf("Listening on: %s://%s", config.Network, listenAddr)

	// 处理信号，优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logf("Shutting down listener...")
		listener.Close()
	}()

	// 接受连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			logf("Accept failed: %v", err)
			continue
		}

		logf("Connection received: %s", conn.RemoteAddr())
		setupConnection(conn)

		if command {
			go handleCommandMode(conn)
		} else {
			go handleDataMode(conn)
		}
	}

	return nil
}

// handleCommandMode 处理命令模式
func handleCommandMode(conn net.Conn) {
	defer func() {
		conn.Close()
		logf("Closed: %s", conn.RemoteAddr())
	}()

	shell := getShell()
	cmd := exec.Command(shell)
	convert := newConvert(conn)

	cmd.Stdin = convert
	cmd.Stdout = convert
	cmd.Stderr = convert

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	cmd = exec.CommandContext(ctx, shell)
	cmd.Stdin = convert
	cmd.Stdout = convert
	cmd.Stderr = convert

	if err := cmd.Run(); err != nil {
		logf("Command execution failed: %v", err)
	}
}

// handleDataMode 处理数据传输模式
func handleDataMode(conn net.Conn) {
	defer conn.Close()

	// 从连接读取数据并输出到stdout
	go func() {
		buffer := make([]byte, config.BufferSize)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					logf("Read error: %v", err)
				}
				break
			}
			if n > 0 {
				os.Stdout.Write(buffer[:n])
			}
		}
	}()

	// 从stdin读取数据并发送到连接
	fi, err := os.Stdin.Stat()
	if err != nil {
		logf("Stdin stat failed: %v", err)
		return
	}

	if (fi.Mode() & os.ModeCharDevice) == 0 {
		// 非交互模式，一次性读取所有数据
		buffer, err := io.ReadAll(os.Stdin)
		if err != nil {
			logf("Failed to read stdin: %v", err)
			return
		}
		_, err = conn.Write(buffer)
		if err != nil {
			logf("Failed to write to connection: %v", err)
		}
	} else {
		// 交互模式
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text() + "\n"
			_, err := conn.Write([]byte(line))
			if err != nil {
				logf("Failed to write to connection: %v", err)
				break
			}
		}
		if err := scanner.Err(); err != nil {
			logf("Scanner error: %v", err)
		}
	}
}

// listenUDP UDP监听模式
func listenUDP(host string, port int, command bool) error {
	listenAddr := net.JoinHostPort(host, strconv.Itoa(port))

	var conn net.PacketConn
	var err error

	if config.SSL {
		return fmt.Errorf("SSL not supported for UDP")
	}

	conn, err = net.ListenPacket("udp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen failed: %v", err)
	}
	defer conn.Close()

	logf("Listening on: %s://%s", config.Network, listenAddr)

	// 处理信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logf("Shutting down UDP listener...")
		conn.Close()
	}()

	buf := make([]byte, udpBufSize)

	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			logf("Read error: %v", err)
			continue
		}

		if n > 0 {
			logf("Received %d bytes from %s", n, addr.String())
			fmt.Printf("From %s: %s", addr.String(), string(buf[:n]))

			// 如果需要回显数据
			if !command {
				_, err = conn.WriteTo(buf[:n], addr)
				if err != nil {
					logf("Write error: %v", err)
				}
			}
		}
	}

	return nil
}

// dialWithRetry 带重试的连接
func dialWithRetry(network, host string, port int) (net.Conn, error) {
	dialAddr := net.JoinHostPort(host, strconv.Itoa(port))

	var conn net.Conn
	var err error

	for i := 0; i <= config.Retries; i++ {
		if i > 0 {
			logf("Retry %d/%d connecting to %s", i, config.Retries, dialAddr)
			time.Sleep(time.Duration(i) * time.Second)
		}

		ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)

		if config.SSL && network == "tcp" {
			conn, err = tls.Dial("tcp", dialAddr, &tls.Config{
				InsecureSkipVerify: true, // 注意：生产环境应该验证证书
			})
		} else {
			var d net.Dialer
			conn, err = d.DialContext(ctx, network, dialAddr)
		}

		cancel()

		if err == nil {
			setupConnection(conn)
			return conn, nil
		}

		logf("Connection attempt %d failed: %v", i+1, err)
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %v", config.Retries+1, err)
}

// dial 连接模式
func dial(network, host string, port int, command bool) error {
	conn, err := dialWithRetry(network, host, port)
	if err != nil {
		return fmt.Errorf("dial failed: %v", err)
	}
	defer conn.Close()

	logf("Connected to: %s://%s", network, conn.RemoteAddr())

	if command {
		handleCommandMode(conn)
		return nil
	} else {
		handleDataMode(conn)
		return nil
	}
}

// listenWeb Web服务器模式
func listenWeb(host string, port int, path string) error {
	listenAddr := net.JoinHostPort(host, strconv.Itoa(port))

	// 检查静态文件目录是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("static path does not exist: %s", path)
	}

	logf("Starting web server on: %s, serving: %s", listenAddr, path)

	// 添加一些基本的HTTP头
	fileServer := http.FileServer(http.Dir(path))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logf("HTTP %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		fileServer.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:    listenAddr,
		Handler: handler,
	}

	// 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		logf("Shutting down web server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	return server.ListenAndServe()
}

// generateSelfSignedCert 生成自签名证书（仅用于测试）
func generateSelfSignedCert() (tls.Certificate, error) {
	// 这里应该生成真实的证书，为了简化使用固定的测试证书
	// 实际使用中应该从文件读取或使用真实的证书
	return tls.Certificate{}, fmt.Errorf("SSL certificate generation not implemented")
}

func main() {
	if config.Help {
		flag.Usage()
		return
	}

	// 验证配置
	if err := validateConfig(); err != nil {
		logf("Configuration error: %v", err)
		os.Exit(1)
	}

	// 设置信号处理
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sigs
		logf("Received signal, exiting...")
		os.Exit(0)
	}()

	var err error

	// 根据模式执行相应功能
	if config.Web {
		err = listenWeb(config.Host, config.Port, config.Path)
	} else if config.Listen {
		switch config.Network {
		case udpNetwork:
			err = listenUDP(config.Host, config.Port, config.Command)
		case tcpNetwork:
			err = listenTCP(config.Host, config.Port, config.Command)
		default:
			err = fmt.Errorf("unsupported network protocol: %s", config.Network)
		}
	} else {
		err = dial(config.Network, config.Host, config.Port, config.Command)
	}

	if err != nil {
		logf("Error: %v", err)
		os.Exit(1)
	}
}
