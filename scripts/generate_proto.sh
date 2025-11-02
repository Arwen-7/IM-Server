#!/bin/bash

# Protocol Buffer 代码生成脚本

set -e

echo "Generating Protocol Buffer code..."

# 检查 protoc 是否安装
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed"
    echo "Please install protoc:"
    echo "  macOS: brew install protobuf"
    echo "  Linux: sudo apt-get install protobuf-compiler"
    exit 1
fi

# 检查 protoc-gen-go 是否安装
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    echo "Please ensure \$GOPATH/bin is in your PATH"
fi

# 生成 Go 代码到 internal/protocol 目录
echo "Generating protobuf code to internal/protocol/..."
protoc --go_out=. --go_opt=paths=source_relative \
    --go_opt=Mapi/proto/im_protocol.proto=github.com/arwen/im-server/internal/protocol \
    api/proto/im_protocol.proto

# 移动生成的文件到正确位置
if [ -f "api/proto/im_protocol.pb.go" ]; then
    mv api/proto/im_protocol.pb.go internal/protocol/
    echo "Protocol Buffer code generated successfully at internal/protocol/im_protocol.pb.go"
else
    echo "Error: Failed to generate protobuf code"
    exit 1
fi

