# Netcat V2

一个功能增强的网络工具，支持正向shell和反向shell，详见下方用法。

---

## 主要用法

### 正向shell（服务端执行命令，客户端控制）

```bash
# 服务端（被控端，执行命令）
./netcat -l -v -e
# 客户端（控制端，输入命令）
./netcat -h 127.0.0.1 -p 4000
```
- 客户端输入命令，服务端执行，结果返回客户端。

### 反向shell（客户端执行命令，服务端控制）

```bash
# 服务端（控制端，输入命令）
./netcat -l -v
# 客户端（被控端，执行命令）
./netcat -h 127.0.0.1 -p 4000 -e
```
- 服务端输入命令，客户端执行，结果返回服务端。

---

## 其他功能

- 普通数据转发：两端都不加-e
- SSL/TLS、Web服务、UDP等见下文

---

## 测试

可用`make test`自动验证正向shell和反向shell功能。

---

## 主要改进（V2）

### 🛡️ 健壮性增强
- **完善的错误处理**: 所有网络操作都有适当的错误处理
- **资源管理**: 正确关闭连接和监听器，避免资源泄漏
- **并发安全**: 使用互斥锁保护共享资源
- **配置验证**: 启动前验证所有配置参数

### ⚡ 新功能
- **超时控制**: 可配置连接超时时间
- **重试机制**: 连接失败时自动重试
- **SSL/TLS支持**: 支持加密连接
- **Keep-Alive**: TCP连接保活机制
- **优雅关闭**: 支持信号处理，优雅退出
- **完整UDP支持**: UDP监听持续运行，不再只处理一个包

### 🔧 配置选项
- `-timeout`: 连接超时时间 (默认30秒)
- `-retries`: 重试次数 (默认3次)
- `-ssl`: 启用SSL/TLS加密
- `-keepalive`: 启用TCP保活 (默认启用)
- `-buffer`: 数据传输缓冲区大小 (默认4096字节)

---

## 快速上手（V2）

### 基本用法

```bash
# TCP监听模式
./netcat -l -p 8080

# TCP连接模式
./netcat localhost 8080

# UDP监听模式
./netcat -l -n udp -p 8080

# UDP连接模式
./netcat -n udp localhost 8080
```

### 高级功能

```bash
# 带超时和重试的连接
./netcat -timeout 10s -retries 5 localhost 8080

# SSL加密连接
./netcat -ssl localhost 8443

# SSL监听模式
./netcat -l -ssl -p 8443

# SSL Web服务器
./netcat -web -ssl -p 8443 -path ./public

# 命令模式 (反向shell)
./netcat -l -e -p 8080

# Web静态文件服务器
./netcat -web -p 8080 -path ./public

# 自定义缓冲区大小
./netcat -buffer 8192 localhost 8080
```

---

## 命令行选项（V2）

```
name: netcat 
version: 
commit: 
build_tags: 
go: go version go1.21.0 darwin/amd64

usage: netcat [options] [host] [port]

options:
  -p int
        host port to connect or listen (default 4000)
  -help
        print this help
  -v    verbose mode (default true)
  -l    listen mode
  -e    shell mode
  -web  web static server
  -path string
        web static path (default "public")
  -n string
        network protocol (default "tcp")
  -h string
        host addr to connect or listen (default "0.0.0.0")
  -timeout duration
        connection timeout (default 30s)
  -retries int
        connection retry attempts (default 3)
  -ssl  use SSL/TLS
  -keepalive
        enable keep-alive (default true)
  -buffer int
        buffer size for data transfer (default 4096)
```

---

## 场景演示与截图（V2）

### 1. 正向命令执行
![](images/p1@2x.png)

### 2. 反向命令执行
![](images/p2@2x.png)

### 3. 文件传输
![](images/p3@2x.png)

### 4. 标准输入输出
![](images/p4@2x.png)

### 5. Web静态服务器
![](images/p5.png)

---

## 功能特性（V2）

### 1. 网络协议支持
- **TCP**: 完整的TCP客户端/服务器功能
- **UDP**: 完整的UDP客户端/服务器功能
- **SSL/TLS**: TCP连接支持加密传输

### 2. 监听模式
- 支持多客户端连接
- 优雅关闭处理
- 信号处理 (Ctrl+C)
- 连接日志记录

### 3. 连接模式
- 自动重试机制
- 超时控制
- 连接保活
- 错误恢复

### 4. 数据传输
- 双向数据传输
- 缓冲区优化
- 字符编码处理 (Windows GBK支持)
- 交互式和非交互式模式

### 5. 命令模式
- 反向shell功能
- 跨平台shell支持
- 命令执行超时控制

### 6. Web服务器
- 静态文件服务
- HTTP请求日志
- 优雅关闭
- SSL/TLS支持（HTTPS）

---

## 错误处理（V2）

### 网络错误
- 连接失败自动重试
- 超时处理
- 连接中断恢复

### 系统错误
- 信号处理
- 资源清理
- 优雅退出

### 配置错误
- 参数验证
- 端口范围检查
- 协议支持检查

---

## 性能优化（V2）

### 内存管理
- 固定大小缓冲区
- 及时释放资源
- 避免内存泄漏

### 网络优化
- TCP保活机制
- 连接复用
- 缓冲区大小可配置

### 并发处理
- 多客户端支持
- 线程安全
- 资源竞争保护

---

## 安全特性（V2）

### SSL/TLS支持
- 加密传输
- 自签名证书自动生成
- 证书验证 (可配置)
- 安全连接
- 支持HTTPS Web服务器

### 访问控制
- 连接日志
- 错误日志
- 调试信息

---

## 常见使用场景

### 网络调试
```bash
# 测试端口连通性
./netcat -timeout 5s localhost 80

# 监听端口查看连接
./netcat -l -p 8080
```

### 文件传输
```bash
# 发送文件
cat file.txt | ./netcat localhost 8080

# 接收文件
./netcat -l -p 8080 > received.txt
```

### 反向Shell
```bash
# 服务器端
./netcat -l -e -p 8080

# 客户端连接
./netcat localhost 8080
```

### Web服务
```bash
# 启动静态文件服务器
./netcat -web -p 8080 -path ./public

# 启动HTTPS静态文件服务器
./netcat -web -ssl -p 8443 -path ./public
```

---

## 构建

```bash
# 编译
go build -o netcat main.go

# 交叉编译
GOOS=linux GOARCH=amd64 go build -o netcat-linux main.go
GOOS=windows GOARCH=amd64 go build -o netcat.exe main.go
```

---

## 许可证

MIT License