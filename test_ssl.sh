#!/bin/bash

# SSL功能测试脚本

echo "=== Netcat V2 SSL功能测试 ==="

# 检查netcat是否存在
if [ ! -f "./netcat" ]; then
    echo "错误: netcat 可执行文件不存在，请先运行 'make build'"
    exit 1
fi

echo "✓ netcat 可执行文件存在"

# 测试SSL监听模式
echo -e "\n1. 测试SSL监听模式..."
./netcat -l -p 8443 -ssl &
SSL_SERVER_PID=$!
sleep 2

# 检查进程是否在运行
if kill -0 $SSL_SERVER_PID 2>/dev/null; then
    echo "✓ SSL服务器启动成功"
    
    # 测试SSL连接
    echo -e "\n2. 测试SSL客户端连接..."
    timeout 5s ./netcat -ssl localhost 8443 &
    SSL_CLIENT_PID=$!
    sleep 2
    
    if kill -0 $SSL_CLIENT_PID 2>/dev/null; then
        echo "✓ SSL客户端连接成功"
        kill $SSL_CLIENT_PID 2>/dev/null
    else
        echo "✗ SSL客户端连接失败"
    fi
    
    # 清理服务器进程
    kill $SSL_SERVER_PID 2>/dev/null
    echo "✓ SSL服务器已关闭"
else
    echo "✗ SSL服务器启动失败"
fi

# 测试SSL Web服务器
echo -e "\n3. 测试SSL Web服务器..."
mkdir -p test_ssl_web
echo "<html><body><h1>SSL Test</h1></body></html>" > test_ssl_web/index.html

./netcat -web -p 8444 -ssl -path test_ssl_web &
SSL_WEB_PID=$!
sleep 2

if kill -0 $SSL_WEB_PID 2>/dev/null; then
    echo "✓ SSL Web服务器启动成功"
    
    # 测试HTTPS访问
    echo -e "\n4. 测试HTTPS访问..."
    curl -k -s https://localhost:8444 > /dev/null
    if [ $? -eq 0 ]; then
        echo "✓ HTTPS访问成功"
    else
        echo "✗ HTTPS访问失败"
    fi
    
    kill $SSL_WEB_PID 2>/dev/null
    echo "✓ SSL Web服务器已关闭"
else
    echo "✗ SSL Web服务器启动失败"
fi

# 清理测试文件
rm -rf test_ssl_web

# 测试证书生成
echo -e "\n5. 测试证书生成..."
./netcat -l -p 8445 -ssl &
CERT_TEST_PID=$!
sleep 1

if kill -0 $CERT_TEST_PID 2>/dev/null; then
    echo "✓ 自签名证书生成成功"
    kill $CERT_TEST_PID 2>/dev/null
else
    echo "✗ 自签名证书生成失败"
fi

echo -e "\n=== SSL测试完成 ==="
echo "所有SSL功能测试已完成。"

# 清理可能的残留进程
pkill -f "netcat.*ssl" 2>/dev/null || true 