# 自动更新和版本管理功能

## 新增功能

### 1. 版本信息显示
显示当前程序的详细版本信息，包括：
- 版本号
- Go编译器版本  
- 操作系统和架构
- 构建时间

```bash
# 使用方式1：子命令格式
./sub version

# 使用方式2：标志格式  
./sub -version
```

### 2. 自动更新功能
从GitHub Releases自动下载并安装最新版本：

```bash
# 使用方式1：子命令格式
./sub update

# 使用方式2：标志格式
./sub -update
```

## 构建脚本

使用增强的构建脚本来构建带有版本信息和构建时间的程序：

```bash
# 默认构建
./build.sh

# 指定版本号
./build.sh -v 1.2.0

# 指定输出文件名
./build.sh -o myapp

# 同时指定版本和输出文件名
./build.sh -v 1.2.0 -o myapp

# 查看构建脚本帮助
./build.sh -h
```

## 自动更新工作原理

1. **检查更新**：程序连接到GitHub API检查最新Release版本
2. **版本比较**：将当前版本与远程最新版本进行比较
3. **下载更新**：如果有新版本，下载对应平台的压缩包
4. **解压文件**：自动解压压缩包并提取二进制文件
5. **验证文件**：验证下载文件的完整性并设置执行权限
6. **备份原程序**：在替换前备份当前程序
7. **替换程序**：用新版本替换当前程序
8. **清理**：清理临时文件和备份文件

## 支持的平台

自动更新功能支持以下平台的自动检测和下载：

- **Windows**: `sub-windows-amd64.zip`, `sub-windows-arm64.zip`
- **macOS**: `sub-darwin-amd64.tar.gz`, `sub-darwin-arm64.tar.gz`
- **Linux**: `sub-linux-amd64.tar.gz`, `sub-linux-arm64.tar.gz`
- **FreeBSD**: `sub-freebsd-amd64.tar.gz`

## 安全特性

- **备份机制**：替换前自动备份原程序，出错时可自动恢复
- **原子操作**：所有文件操作都是原子性的，避免中途失败导致程序损坏
- **权限检查**：自动设置正确的执行权限
- **验证机制**：下载后验证文件完整性

## 错误处理

- 网络错误：超时重试机制
- 权限错误：提示用户权限不足
- 文件错误：自动恢复备份
- 平台不支持：友好提示信息

## 版本号管理

程序支持两种版本号设置方式：

1. **代码中设置**：在`src/github.com/sixproxy/version/version.go`中的`VERSION`变量
2. **构建时注入**：使用构建脚本的`-v`参数或通过ldflags注入

构建时注入示例：
```bash
go build -ldflags "-X singbox_sub/src/github.com/sixproxy/version.VERSION=1.2.0 -X 'singbox_sub/src/github.com/sixproxy/version.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" -o singbox_sub src/github.com/sixproxy/sub.go
```

## 使用示例

```bash
# 查看当前版本
./sub version

# 检查是否有更新
./sub update

# 正常使用（生成配置）
./sub

# 详细模式生成配置
./sub -v

# 查看帮助信息
./sub -h
```

## 命令行参数完整列表

```
选项:
  -os string    目标操作系统 (默认: auto)
                可选值: auto, darwin, linux, windows, all
  -v            详细输出模式 (启用DEBUG日志)
  -h            显示此帮助信息
  -version      显示版本信息
  -update       检查并更新到最新版本

子命令:
  version       显示版本信息
  update        检查并更新到最新版本
  help          显示帮助信息
```