#!/bin/bash

# Netcat 增强版测试脚本

echo "=== Netcat 增强版功能测试 ==="

# 检查netcat是否存在
if [ ! -f "./netcat" ]; then
    echo "错误: netcat 可执行文件不存在，请先运行 'make build'"
    exit 1
fi

echo "✓ netcat 可执行文件存在"

# 测试帮助信息
echo -e "\n1. 测试帮助信息..."
./netcat -help > /dev/null
if [ $? -eq 0 ]; then
    echo "✓ 帮助信息正常"
else
    echo "✗ 帮助信息异常"
fi

# 测试配置验证
echo -e "\n2. 测试配置验证..."
./netcat -p 99999 2>&1 | grep -q "invalid port number"
if [ $? -eq 0 ]; then
    echo "✓ 端口验证正常"
else
    echo "✗ 端口验证异常"
fi

# 测试TCP监听模式
echo -e "\n3. 测试TCP监听模式..."
./netcat -l -p 8080 &
LISTEN_PID=$!
sleep 1

# 检查进程是否在运行
if kill -0 $LISTEN_PID 2>/dev/null; then
    echo "✓ TCP监听模式正常"
    kill $LISTEN_PID 2>/dev/null
else
    echo "✗ TCP监听模式异常"
fi

# 测试UDP监听模式
echo -e "\n4. 测试UDP监听模式..."
./netcat -l -n udp -p 8081 &
UDP_PID=$!
sleep 1

if kill -0 $UDP_PID 2>/dev/null; then
    echo "✓ UDP监听模式正常"
    kill $UDP_PID 2>/dev/null
else
    echo "✗ UDP监听模式异常"
fi

# 测试连接超时
echo -e "\n5. 测试连接超时..."
./netcat -timeout 2s localhost 9999 2>&1 | grep -q "failed to connect"
if [ $? -eq 0 ]; then
    echo "✓ 连接超时正常"
else
    echo "✗ 连接超时异常"
fi

# 测试重试机制
echo -e "\n6. 测试重试机制..."
./netcat -retries 2 -timeout 1s localhost 9999 2>&1 | grep -q "Retry"
if [ $? -eq 0 ]; then
    echo "✓ 重试机制正常"
else
    echo "✗ 重试机制异常"
fi

# 测试Web服务器模式
echo -e "\n7. 测试Web服务器模式..."
# 创建测试目录
mkdir -p test_web
echo "<html><body><h1>Test</h1></body></html>" > test_web/index.html

./netcat -web -p 8082 -path test_web &
WEB_PID=$!
sleep 1

if kill -0 $WEB_PID 2>/dev/null; then
    echo "✓ Web服务器模式正常"
    kill $WEB_PID 2>/dev/null
else
    echo "✗ Web服务器模式异常"
fi

# 清理测试文件
rm -rf test_web

# 测试信号处理
echo -e "\n8. 测试信号处理..."
./netcat -l -p 8083 &
SIG_PID=$!
sleep 1

if kill -0 $SIG_PID 2>/dev/null; then
    kill -TERM $SIG_PID
    sleep 1
    if ! kill -0 $SIG_PID 2>/dev/null; then
        echo "✓ 信号处理正常"
    else
        echo "✗ 信号处理异常"
        kill -KILL $SIG_PID 2>/dev/null
    fi
else
    echo "✗ 信号处理测试失败"
fi

echo -e "\n=== 测试完成 ==="
echo "所有测试已完成，请检查上述结果。"

# 清理可能的残留进程
pkill -f "netcat -l" 2>/dev/null || true 