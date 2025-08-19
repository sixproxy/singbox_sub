# SingBox 订阅配置生成器

支持多协议的 Sing-box 订阅配置生成工具，具备智能 DNS 检测和 CDN 优化功能。

## ✨ 特性

- 🔧 **多协议支持**: Shadowsocks、Hysteria2、Trojan、AnyTLS
- 🌐 **智能 DNS**: 自动检测运营商 DNS
- 🚀 **CDN 优化**: 城市级精确定位，自动设置 client_subnet
- ⚙️ **配置系统**: YAML 配置覆盖，简化使用
- 🌍 **跨平台**: 支持 Linux、macOS、Windows
- 🏗️ **自动构建**: GitHub Actions 多平台编译

## 🚀 快速开始

### 1. 配置订阅

```bash
# 编辑配置文件
vim config/config.yaml
```

**config.yaml 示例:**
```yaml
subs:
  - url: "YOUR_SUBSCRIPTION_URL"
    insecure: false

dns:
  auto_optimize: true  # 启用智能 CDN 优化
```

### 2. 运行程序
- 最常使用
这个命令会生成sing-box的配置，并自动启动sing-box。如果只是需要生成配置，参考 **3.其他命令** 部分
```bash
# 运行
chmod +x sub
./sub
```

Linux系统生成的配置文件:   `linux_config.json`

iOS、Mac系统生成的配置文件: `mac_config.json`

**Tips:**

    因为sing-box的iOS客户端经常延期上线,为了使用sing-box方便,
    Mac端和iOS端的配置文件都使用比较旧的一个版本。
    这样可以稳定使用, 减少折腾。

### 3. 其他命令
- 查看命令行帮助
```bash
./sub -h
```
- 仅生成Linux配置，不执行部署
```bash
./sub -os linux
```
- 仅生成Mac配置，不执行部署
```bash
./sub -os darwin
```
- 在Linux上生成所有配置（只会自动部署Linux配置）
```bash
./sub -os all
```



## 📋 CDN 优化

程序自动检测你的地理位置和运营商，设置最优的 client_subnet：

- 🏙️ **支持城市**: 北京、上海、广州、深圳、杭州、南京、福州、泉州
- 🏢 **支持运营商**: 电信、联通、移动、教育网
- 🔒 **隐私保护**: 使用运营商网段，不暴露真实 IP
- ⚡ **性能提升**: ECS (EDNS Client Subnet) 优化 CDN 响应

## 🏗️ 构建

### 本地构建
```bash
# 构建所有平台
./scripts/build.sh

# 或使用 Go 直接构建
go build -o singbox-sub ./src/github.com/sixproxy/sub.go
```

### Docker 构建
```bash
docker build -t singbox-sub .
docker run -v $(pwd)/config.yaml:/app/config.yaml singbox-sub
```

## 📦 安装
### 下载二进制
前往 [Releases](https://github.com/sixproxy/singbox_sub/releases) 下载对应平台的预编译版本。

## 🔧 配置说明

### 基础配置
- `subs`: 订阅配置列表
  - `url`: 订阅地址
  - `insecure`: 跳过 SSL 验证
  - `prefix`: 节点名前缀
  - `enabled`: 是否启用

### DNS 配置
- `auto_optimize`: 启用自动 DNS 和 CDN 优化.如果启用,自动设置client_subnet
- `client_subnet`: 手动指定 ECS 网段
- `final`: DNS 最终服务器

## 🤝 贡献

欢迎提交 Issues 和 Pull Requests！

## 📄 许可证

MIT License