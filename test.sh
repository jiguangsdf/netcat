#!/bin/bash

# Netcat V2 功能测试脚本

echo "=== Netcat V2 功能测试 ==="

# 检查netcat是否存在
if [ ! -f "./netcat" ]; then
    echo "错误: netcat 可执行文件不存在，请先运行 'make build'"
    exit 1
fi

echo "✓ netcat 可执行文件存在"

# 1. 正向shell测试（服务端-e，客户端普通）
echo -e "\n1. 正向shell测试..."
./netcat -l -p 4100 -e > shell_out.txt 2>&1 &
SHELL_PID=$!
sleep 1

echo "whoami" | ./netcat -h 127.0.0.1 -p 4100 > shell_result.txt
sleep 1
kill $SHELL_PID 2>/dev/null

if grep -q "$(whoami)" shell_result.txt; then
    echo "✓ 正向shell功能正常"
else
    echo "✗ 正向shell功能异常"
    cat shell_result.txt
fi
rm -f shell_result.txt shell_out.txt

# 2. 反向shell测试（客户端-e，服务端普通）
echo -e "\n2. 反向shell测试..."
./netcat -l -p 4101 > revshell_out.txt 2>&1 &
REV_PID=$!
sleep 1

echo "whoami" | ./netcat -h 127.0.0.1 -p 4101 -e > revshell_result.txt
sleep 1
kill $REV_PID 2>/dev/null

if grep -q "$(whoami)" revshell_result.txt; then
    echo "✓ 反向shell功能正常"
else
    echo "✗ 反向shell功能异常"
    cat revshell_result.txt
fi
rm -f revshell_result.txt revshell_out.txt

echo -e "\n=== 测试完成 ==="
echo "所有功能测试已完成，请检查上述结果。"

# 清理可能的残留进程
pkill -f "netcat -l" 2>/dev/null || true 