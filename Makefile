# Netcat 增强版 Makefile

# 变量定义
BINARY_NAME=netcat
MAIN_FILE=main.go
VERSION?=1.0.0
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go 编译参数
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTags=${BUILD_TIME}"

# 默认目标
.PHONY: all
all: build

# 构建
.PHONY: build
build:
	@echo "Building ${BINARY_NAME}..."
	go build ${LDFLAGS} -o ${BINARY_NAME} ${MAIN_FILE}
	@echo "Build completed: ${BINARY_NAME}"

# 清理
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -f ${BINARY_NAME}
	rm -f ${BINARY_NAME}-*
	@echo "Clean completed"

# 测试
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# 格式化代码
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ${MAIN_FILE}

# 代码检查
.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run

# 交叉编译
.PHONY: cross-build
cross-build: clean
	@echo "Cross compiling..."
	
	# Linux
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-amd64 ${MAIN_FILE}
	GOOS=linux GOARCH=386 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-386 ${MAIN_FILE}
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-arm64 ${MAIN_FILE}
	
	# Windows
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-windows-amd64.exe ${MAIN_FILE}
	GOOS=windows GOARCH=386 go build ${LDFLAGS} -o ${BINARY_NAME}-windows-386.exe ${MAIN_FILE}
	
	# macOS
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-amd64 ${MAIN_FILE}
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-arm64 ${MAIN_FILE}
	
	@echo "Cross build completed"

# 安装依赖
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# 运行示例
.PHONY: example
example: build
	@echo "Running example..."
	@echo "Starting TCP listener on port 8080..."
	@echo "In another terminal, run: ./netcat localhost 8080"
	./netcat -l -p 8080

# 帮助
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  clean       - Clean build artifacts"
	@echo "  test        - Run tests"
	@echo "  fmt         - Format code"
	@echo "  lint        - Run linter"
	@echo "  cross-build - Build for multiple platforms"
	@echo "  deps        - Install dependencies"
	@echo "  example     - Run example server"
	@echo "  help        - Show this help"

# 开发模式
.PHONY: dev
dev:
	@echo "Starting development mode..."
	@echo "Watching for changes..."
	@command -v air >/dev/null 2>&1 || { echo "Installing air..."; go install github.com/cosmtrek/air@latest; }
	air

# 发布
.PHONY: release
release: clean cross-build
	@echo "Creating release..."
	tar -czf ${BINARY_NAME}-${VERSION}-linux-amd64.tar.gz ${BINARY_NAME}-linux-amd64
	tar -czf ${BINARY_NAME}-${VERSION}-linux-arm64.tar.gz ${BINARY_NAME}-linux-arm64
	zip ${BINARY_NAME}-${VERSION}-windows-amd64.zip ${BINARY_NAME}-windows-amd64.exe
	zip ${BINARY_NAME}-${VERSION}-darwin-amd64.zip ${BINARY_NAME}-darwin-amd64
	@echo "Release packages created"

# 默认目标
.DEFAULT_GOAL := build