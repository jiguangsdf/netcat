# Netcat V2 SSL功能使用示例

## SSL功能概述

Netcat V2 支持SSL/TLS加密连接，包括：
- 自动生成自签名证书
- SSL客户端连接
- SSL服务器监听
- HTTPS Web服务器

## 基本用法

### 1. SSL服务器监听

```bash
# 启动SSL服务器监听
./netcat -l -ssl -p 8443

# 客户端连接
./netcat -ssl localhost 8443
```

### 2. HTTPS Web服务器

```bash
# 创建测试目录
mkdir -p public
echo "<html><body><h1>HTTPS Test</h1></body></html>" > public/index.html

# 启动HTTPS服务器
./netcat -web -ssl -p 8443 -path public

# 浏览器访问
# https://localhost:8443
```

### 3. SSL命令模式

```bash
# 服务器端（SSL反向shell）
./netcat -l -e -ssl -p 8443

# 客户端连接
./netcat -ssl localhost 8443
```

## 证书信息

自签名证书包含以下信息：
- **组织**: Netcat V2 Test Certificate
- **国家**: CN
- **有效期**: 1年
- **支持的域名**: localhost, 127.0.0.1, 主机名
- **支持的IP**: 127.0.0.1, ::1

## 安全注意事项

1. **自签名证书**: 仅用于测试，生产环境请使用受信任的证书
2. **证书验证**: 客户端连接时会跳过证书验证（InsecureSkipVerify）
3. **加密强度**: 使用2048位RSA密钥
4. **协议支持**: TLS 1.2及以上版本

## 测试命令

```bash
# 运行SSL功能测试
./test_ssl.sh

# 手动测试SSL连接
./netcat -l -ssl -p 8443 &
./netcat -ssl localhost 8443

# 测试HTTPS
curl -k https://localhost:8443
```

## 故障排除

### 常见问题

1. **证书错误**: 浏览器会显示证书不受信任，这是正常的（自签名证书）
2. **连接失败**: 检查端口是否被占用
3. **SSL握手失败**: 确保客户端和服务器都使用SSL模式

### 调试方法

```bash
# 启用详细日志
./netcat -v -ssl localhost 8443

# 检查端口监听
netstat -an | grep 8443
lsof -i :8443
```

## 生产环境建议

1. 使用受信任的CA签发的证书
2. 配置证书文件路径参数
3. 启用证书验证
4. 使用强加密套件
5. 定期更新证书 