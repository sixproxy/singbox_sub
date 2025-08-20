# SingBox 订阅配置生成器
## ✨ 特性
- 🔧 **多协议支持**: Shadowsocks、Hysteria2、Trojan、AnyTLS、VLESS+Reality
- 🌐 **智能 DNS**: 自动检测运营商 DNS
- 🚀 **CDN 优化**: 城市级精确定位，自动设置 client_subnet
- ⚙️ **配置系统**: YAML 配置覆盖，简化使用
- 🌍 **跨平台**: 支持 Linux、macOS、iOS

## 🚀 快速开始
### 1.下载安装包 && 解压 
以linux平台X86架构，64位安装包为例子

下载
```bash
wget -O sub.tar.gz https://ghfast.top/https://github.com/sixproxy/singbox_sub/releases/download/v1.2.1/sub-linux-amd64.tar.gz
```

解压
```bash
tar -zxvf sub.tar.gz
```

进入解压目录
```bash
cd linux-amd64/
```


### 2. 配置订阅

更新到最新版
```bash
./sub update
```

编辑配置文件
```bash
vim config/config.yaml
```

**config.yaml 示例:**
```yaml
subs:
  - url: "YOUR_SUBSCRIPTION_URL"  # 填写订阅地址
    insecure: false

dns:
  auto_optimize: true             # 启用智能 CDN 优化
```

### 3. 运行程序
给可执行权限
```bash
chmod +x sub
```

运行
```bash

./sub
```

**Tips:**

    因为sing-box的iOS客户端经常延期上线,为了使用sing-box方便,
    Mac端和iOS端的配置文件都使用比较旧的一个版本。
    这样可以稳定使用, 减少折腾。

### 5. 其他命令
- 查询版本
```bash
./sub version
```

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
- Linux系统生成的配置文件:   `linux_config.json`
- iOS、Mac系统生成的配置文件: `mac_config.json`


## 🏗️ 构建

### 本地构建
```bash
# Go 直接构建
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