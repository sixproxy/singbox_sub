#!/bin/bash

# sing-box配置生成器构建脚本
# 支持注入版本信息和构建时间

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 默认参数
OUTPUT_NAME="singbox_sub"
VERSION=""
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -o|--output)
            OUTPUT_NAME="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -h|--help)
            echo "用法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  -o, --output NAME    输出文件名 (默认: singbox_sub)"
            echo "  -v, --version VER    版本号 (默认: 从version.go读取)"
            echo "  -h, --help          显示帮助信息"
            echo ""
            echo "示例:"
            echo "  $0                           # 默认构建"
            echo "  $0 -o myapp                  # 指定输出文件名"
            echo "  $0 -v 1.2.0                 # 指定版本号"
            echo "  $0 -o myapp -v 1.2.0        # 指定输出文件名和版本号"
            exit 0
            ;;
        *)
            echo "未知参数: $1"
            echo "使用 $0 -h 查看帮助"
            exit 1
            ;;
    esac
done

# 如果没有指定版本，从version.go中读取
if [ -z "$VERSION" ]; then
    if [ -f "src/github.com/sixproxy/version/version.go" ]; then
        VERSION=$(grep 'VERSION = ' src/github.com/sixproxy/version/version.go | sed 's/.*VERSION = "\(.*\)"/\1/')
    fi
fi

echo "============================================"
echo "正在构建 sing-box配置生成器"
echo "============================================"
echo "版本号: ${VERSION:-unknown}"
echo "构建时间: $BUILD_TIME"
echo "输出文件: $OUTPUT_NAME"
echo "目标系统: $(go env GOOS)/$(go env GOARCH)"
echo "Go版本: $(go version | awk '{print $3}')"
echo "============================================"

# 构建ldflags参数
LDFLAGS=""
if [ -n "$VERSION" ]; then
    LDFLAGS="$LDFLAGS -X singbox_sub/src/github.com/sixproxy/version.VERSION=$VERSION"
fi
LDFLAGS="$LDFLAGS -X 'singbox_sub/src/github.com/sixproxy/version.buildTime=$BUILD_TIME'"

# 执行构建
echo "正在构建..."
if [ -n "$LDFLAGS" ]; then
    go build -ldflags "$LDFLAGS" -o "$OUTPUT_NAME" src/github.com/sixproxy/sub.go
else
    go build -o "$OUTPUT_NAME" src/github.com/sixproxy/sub.go
fi

# 显示构建结果
if [ -f "$OUTPUT_NAME" ]; then
    file_size=$(ls -lh "$OUTPUT_NAME" | awk '{print $5}')
    echo "============================================"
    echo "构建成功！"
    echo "文件大小: $file_size"
    echo "输出路径: $(pwd)/$OUTPUT_NAME"
    echo "============================================"
    echo ""
    echo "测试版本信息:"
    ./"$OUTPUT_NAME" version
else
    echo "构建失败！"
    exit 1
fi