package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
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

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// 常量定义
const (
	udpNetwork = "udp"
	tcpNetwork = "tcp"
	udpBufSize = 64 * 1024
)

// 应用信息
var (
	Name      = ""
	Version   = ""
	Commit    = ""
	BuildTags = ""
	logger    = log.New(os.Stderr, "", 0)
)

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

// 初始化配置
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

// 日志函数
func logf(f string, v ...interface{}) {
	if config.Verbose {
		logger.Output(2, fmt.Sprintf(f, v...))
	}
}

// 使用说明
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

// 配置验证
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

// 参数校验
func validateParameters() {
	// 参数互斥校验：两端不能同时加 -e
	if config.Command {
		if config.Listen {
			logf("[警告] 服务端已开启 -e（正向shell），请确保客户端不要同时加 -e，否则会错乱！")
		} else {
			logf("[警告] 客户端已开启 -e（反向shell），请确保服务端不要同时加 -e，否则会错乱！")
		}
	}

	// UDP 不支持反向 shell（客户端加 -e）
	if config.Network == "udp" && !config.Listen && config.Command {
		fmt.Fprintln(os.Stderr, "[错误] UDP 协议不支持反向 shell（客户端加 -e），请用 TCP 或服务端加 -e！")
		os.Exit(1)
	}
}

// 创建 TLS 配置
func createTLSConfig() (*tls.Config, error) {
	cert, err := generateSelfSignedCert()
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS10,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}, nil
}

// 创建客户端 TLS 配置
func createClientTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true, // 注意：生产环境应该验证证书
		MinVersion:         tls.VersionTLS10,
		MaxVersion:         tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}
}

// 设置信号处理
func setupSignalHandler(cleanup func()) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logf("Received signal, shutting down...")
		if cleanup != nil {
			cleanup()
		}
	}()
}

// 执行命令并返回结果
func executeCommand(cmdStr string) ([]byte, error) {
	shell, shellArgs := getShell()
	args := append(shellArgs, "-c", cmdStr)
	cmd := exec.Command(shell, args...)
	return cmd.CombinedOutput()
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
	// 使用 Go 标准库进行 GBK 到 UTF-8 的转换
	reader := transform.NewReader(strings.NewReader(string(p)), simplifiedchinese.GBK.NewDecoder())
	result, err := io.ReadAll(reader)
	if err != nil {
		return p // 如果转换失败，返回原始数据
	}
	return result
}

func (convert *Convert) Write(p []byte) (n int, err error) {
	convert.mu.Lock()
	defer convert.mu.Unlock()

	switch runtime.GOOS {
	case "windows":
		// 在 Windows 下进行 GBK 编码转换
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

// 获取 shell 和参数
func getShell() (string, []string) {
	switch runtime.GOOS {
	case "linux", "darwin":
		// 交互模式下不使用 -i 参数，避免 "no job control" 提示
		if config.Command {
			return "/bin/sh", []string{}
		}
		return "/bin/sh", []string{"-i"}
	case "freebsd":
		if config.Command {
			return "/bin/csh", []string{}
		}
		return "/bin/csh", []string{"-i"}
	case "windows":
		return "cmd.exe", []string{}
	default:
		if config.Command {
			return "/bin/sh", []string{}
		}
		return "/bin/sh", []string{"-i"}
	}
}

// 设置连接参数
func setupConnection(conn net.Conn) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if config.KeepAlive {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)
		}
		tcpConn.SetLinger(0)
	}
}

// 处理命令模式
func handleCommandMode(conn net.Conn) {
	defer func() {
		conn.Close()
		logf("Closed: %s", conn.RemoteAddr())
	}()

	shell, args := getShell()
	cmd := exec.Command(shell, args...)
	convert := newConvert(conn)

	cmd.Stdin = convert
	cmd.Stdout = convert
	cmd.Stderr = convert

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	cmd = exec.CommandContext(ctx, shell, args...)
	cmd.Stdin = convert
	cmd.Stdout = convert
	cmd.Stderr = convert

	if err := cmd.Run(); err != nil {
		logf("Command execution failed: %v", err)
	}
}

// 处理数据传输模式
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

// UDP 监听模式
func listenUDP(host string, port int, command bool) error {
	listenAddr := net.JoinHostPort(host, strconv.Itoa(port))

	if config.SSL {
		return fmt.Errorf("SSL not supported for UDP")
	}

	conn, err := net.ListenPacket("udp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen failed: %v", err)
	}
	defer conn.Close()

	logf("Listening on: %s://%s", config.Network, listenAddr)

	// 处理信号
	setupSignalHandler(func() {
		logf("Shutting down UDP listener...")
		conn.Close()
	})

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
			cmdStr := strings.TrimSpace(string(buf[:n]))
			if command {
				// 执行命令并返回结果
				output, err := executeCommand(cmdStr)
				if err != nil {
					output = append(output, []byte("\nError: "+err.Error())...)
				}
				_, err = conn.WriteTo(output, addr)
				if err != nil {
					logf("Write error: %v", err)
				}
			} else {
				fmt.Printf("From %s: %s", addr.String(), string(buf[:n]))
				_, err = conn.WriteTo(buf[:n], addr)
				if err != nil {
					logf("Write error: %v", err)
				}
			}
		}
	}

	return nil
}

// UDP 客户端命令模式处理
func handleUDPClientCommand(host string, port int) error {
	dialAddr := net.JoinHostPort(host, strconv.Itoa(port))

	// 创建UDP连接
	conn, err := net.Dial("udp", dialAddr)
	if err != nil {
		return fmt.Errorf("UDP dial failed: %v", err)
	}
	defer conn.Close()

	logf("UDP client connected to %s", dialAddr)

	// 处理信号
	setupSignalHandler(func() {
		logf("UDP client shutting down...")
		conn.Close()
	})

	// 从stdin读取命令并发送
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd := scanner.Text()
		if cmd == "" {
			continue
		}

		// 发送命令
		_, err := conn.Write([]byte(cmd))
		if err != nil {
			logf("Failed to send command: %v", err)
			continue
		}

		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		// 接收响应
		buffer := make([]byte, 4096)
		n, err := conn.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				logf("No response received (timeout)")
			} else {
				logf("Failed to read response: %v", err)
			}
			continue
		}

		// 输出响应
		if n > 0 {
			os.Stdout.Write(buffer[:n])
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %v", err)
	}

	return nil
}

// 带重试的连接
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
			tlsConfig := createClientTLSConfig()
			conn, err = tls.Dial("tcp", dialAddr, tlsConfig)
		} else {
			var d net.Dialer
			conn, err = d.DialContext(ctx, network, dialAddr)
		}

		cancel()

		if err == nil {
			logf("Successfully connected to %s://%s", network, dialAddr)
			setupConnection(conn)
			return conn, nil
		}

		logf("Connection attempt %d failed: %v", i+1, err)
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %v", config.Retries+1, err)
}

// TCP 监听模式
func listen(network, host string, port int, command bool) error {
	listenAddr := net.JoinHostPort(host, strconv.Itoa(port))

	// 创建监听器（支持 TLS）
	var listener net.Listener
	var err error

	if config.SSL {
		tlsConfig, err := createTLSConfig()
		if err != nil {
			return err
		}
		listener, err = tls.Listen("tcp", listenAddr, tlsConfig)
	} else {
		listener, err = net.Listen("tcp", listenAddr)
	}

	if err != nil {
		return fmt.Errorf("Listen failed: %s", err)
	}
	defer listener.Close()

	logf("Listening on: %s://%s", network, listenAddr)

	// 处理信号，优雅关闭
	setupSignalHandler(func() {
		logf("Shutting down listener...")
		listener.Close()
	})

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

		// 如果是 TLS 连接，完成握手
		if config.SSL {
			if tlsConn, ok := conn.(*tls.Conn); ok {
				logf("[TLS] Starting handshake...")
				if err := tlsConn.Handshake(); err != nil {
					logf("TLS handshake failed: %v", err)
					conn.Close()
					continue
				}
				logf("[TLS] Handshake completed")
			}
		}

		if command {
			go handleCommandMode(conn)
		} else {
			go handleDataMode(conn)
		}
	}

	return nil
}

// Web 服务器模式
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

	// 如果启用SSL，配置TLS
	if config.SSL {
		tlsConfig, err := createTLSConfig()
		if err != nil {
			return err
		}
		server.TLSConfig = tlsConfig
	}

	// 优雅关闭
	setupSignalHandler(func() {
		logf("Shutting down web server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	})

	if config.SSL {
		return server.ListenAndServeTLS("", "")
	}
	return server.ListenAndServe()
}

// 生成自签名证书（仅用于测试）
func generateSelfSignedCert() (tls.Certificate, error) {
	// 生成RSA私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate private key: %v", err)
	}

	// 创建证书模板
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate serial number: %v", err)
	}

	// 获取主机名
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	// 创建证书模板
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Netcat V2 Test Certificate"},
			Country:      []string{"CN"},
			Province:     []string{"Test"},
			Locality:     []string{"Test"},
			CommonName:   hostname,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1年有效期
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{hostname, "localhost", "127.0.0.1"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	// 创建证书
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create certificate: %v", err)
	}

	// 创建tls.Certificate
	cert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  privateKey,
	}

	return cert, nil
}

// 主函数
func main() {
	if config.Help {
		flag.Usage()
		return
	}

	// 参数校验
	validateParameters()

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
			err = listen(config.Network, config.Host, config.Port, config.Command)
		default:
			err = fmt.Errorf("unsupported network protocol: %s", config.Network)
		}
	} else {
		if config.Network == "udp" && config.Command {
			// UDP 客户端命令模式：发送命令并接收响应
			err = handleUDPClientCommand(config.Host, config.Port)
		} else {
			conn, err := dialWithRetry(config.Network, config.Host, config.Port)
			if err != nil {
				logf("Error: %v", err)
				os.Exit(1)
			}
			if config.Command {
				handleCommandMode(conn)
			} else {
				handleDataMode(conn)
			}
		}
	}

	if err != nil {
		logf("Error: %v", err)
		os.Exit(1)
	}
}
