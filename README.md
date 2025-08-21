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

下载 && 解压 && 进入解压目录
```bash
wget -O sub.tar.gz https://github.com/sixproxy/singbox_sub/releases/download/v1.2.9/sub-linux-amd64.tar.gz \
&& tar -zxvf sub.tar.gz \
&& cd linux-amd64/
```

### 2. 配置订阅
更新到最新版 && 编辑配置文件
```bash
./sub update && vim config/config.yaml
```

**config.yaml 说明:**
```yaml
subs:
  - url: ""                                   # 你VPN订阅地址
    insecure: false                           # 是否跳过SSL验证
dns:
  auto_optimize: true                         # 自动获取本地DNS和做CDN优化
github:
  mirror_url: "https://ghfast.top"            # GitHub镜像地址,用于加速
```

### 3. 运行程序
给可执行权限 && 运行
```bash
chmod +x sub && ./sub
```

**Tips:**

    因为sing-box的iOS客户端经常延期上线,为了使用sing-box方便,
    Mac端和iOS端的配置文件都使用比较旧的一个版本。
    这样可以稳定使用, 减少折腾。

### 4.模版配置
如果想自定义模版，可以参考[wiki](https://github.com/sixproxy/singbox_sub/wiki)

**小白建议就用我的模版就够了**

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

### DNS 配置
- `auto_optimize`: 启用自动 DNS 和 CDN 优化.如果启用,自动设置client_subnet
- `client_subnet`: 手动指定 ECS 网段
- `final`: DNS 最终服务器

## 🤝 新功能开发中
- [ ] 添加sing-box最新稳定版下载功能
- [ ] 添加自动挑选可用github镜像功能
- [ ] 如果sing-box启动失败, 打印出sing-box启动失败的具体原因, 回滚sing-box之前配置，并重新启动sing-box.
- [ ] Wiki添加模版配置说明
- [ ] 增加tuic协议支持
- [ ] 增加wg协议支持
- [ ] 完善ss协议支持
- [ ] 完善socks协议支持
- [ ] 可自由定制前缀
- [ ] 支持配置自定义github镜像
- [ ] 提供web页面管理

## 📄 许可证

MIT License