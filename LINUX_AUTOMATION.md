# Linux自动化部署

## 概述

应用程序现在支持Linux系统的完全自动化部署，包括服务停止、配置生成、文件部署和服务启动的全流程自动化。

## 主要功能

### ✅ 自动化流程
1. **程序启动时**: 自动停止现有的sing-box服务
2. **配置生成后**: 自动将`linux_config.json`拷贝到`/etc/sing-box/config.json`
3. **部署完成后**: 自动启动sing-box服务

### ✅ 智能检测
- **系统检测**: 仅在Linux系统上执行自动化流程
- **脚本检查**: 自动检查必需的bash脚本是否存在
- **权限处理**: 优雅处理权限不足的情况
- **错误恢复**: 详细的错误日志和容错机制

### ✅ 灵活控制
- **开发模式**: 在非Linux系统上仅生成配置，不执行部署
- **强制模式**: 支持在Linux上生成其他系统配置而不自动部署
- **详细模式**: 提供完整的部署过程日志

## 使用方法

### Linux生产环境

```bash
# 完整自动化部署（推荐）
./singbox_sub

# 详细模式查看部署过程
./singbox_sub -v

# 执行流程：
# 1. 停止sing-box服务 -> bash/stop_singbox.sh
# 2. 生成Linux配置 -> linux_config.json
# 3. 部署配置文件 -> /etc/sing-box/config.json
# 4. 启动sing-box服务 -> bash/start_singbox.sh
```

### 开发环境（非Linux）

```bash
# 仅生成Linux配置，不执行部署
./singbox_sub -os linux

# 生成所有配置
./singbox_sub -os all
```

### 特殊场景

```bash
# 在Linux上生成macOS配置（不会自动部署）
./singbox_sub -os darwin

# 在Linux上生成所有配置（只会自动部署Linux配置）
./singbox_sub -os all
```

## 必需的文件

### Bash脚本
- **`bash/stop_singbox.sh`**: 停止sing-box服务的脚本
- **`bash/start_singbox.sh`**: 启动sing-box服务的脚本

### 权限要求
- **执行权限**: bash脚本需要可执行权限
- **写入权限**: 需要对`/etc/sing-box/`目录的写入权限
- **服务权限**: 可能需要sudo权限来停止/启动系统服务

## 日志输出示例

### 完整自动化流程
```bash
./singbox_sub -v
```

输出示例：
```
[2025-08-19 12:11:20] [INFO] [sub.go:99] 当前操作系统: linux
[2025-08-19 12:11:20] [INFO] [sub.go:210] 正在停止sing-box服务...
[2025-08-19 12:11:20] [INFO] [sub.go:225] sing-box服务已停止
[2025-08-19 12:11:20] [INFO] [sub.go:105] 使用自动检测的系统类型: linux
[2025-08-19 12:11:20] [INFO] [sub.go:134] 开始生成Linux配置文件...
[2025-08-19 12:11:20] [INFO] [template.go:263] 配置文件已写入: linux_config.json
[2025-08-19 12:11:20] [INFO] [sub.go:139] Linux配置文件生成成功
[2025-08-19 12:11:20] [INFO] [sub.go:234] 正在部署Linux配置文件...
[2025-08-19 12:11:20] [DEBUG] [sub.go:247] 创建配置目录: /etc/sing-box
[2025-08-19 12:11:20] [DEBUG] [sub.go:255] 拷贝配置文件: linux_config.json -> /etc/sing-box/config.json
[2025-08-19 12:11:20] [INFO] [sub.go:268] 配置文件已成功部署到: /etc/sing-box/config.json
[2025-08-19 12:11:20] [INFO] [sub.go:273] 正在启动sing-box服务...
[2025-08-19 12:11:20] [INFO] [sub.go:288] sing-box服务已启动
```

### 脚本缺失处理
```
[2025-08-19 12:11:20] [WARN] [sub.go:214] 停止脚本不存在: bash/stop_singbox.sh，跳过停止步骤
[2025-08-19 12:11:20] [WARN] [sub.go:277] 启动脚本不存在: bash/start_singbox.sh，跳过启动步骤
```

### 权限不足处理
```
[2025-08-19 12:11:20] [ERROR] [sub.go:250] 创建配置目录失败: permission denied
[2025-08-19 12:11:20] [ERROR] [sub.go:285] 启动sing-box服务失败: permission denied
```

## 错误处理

### 常见问题及解决方案

**1. 权限不足**
```bash
# 使用sudo运行
sudo ./singbox_sub

# 或者给程序设置适当权限
chmod +x ./singbox_sub
```

**2. 脚本缺失**
```bash
# 确保脚本存在并有执行权限
ls -la bash/
chmod +x bash/*.sh
```

**3. 目录不存在**
```bash
# 手动创建目录
sudo mkdir -p /etc/sing-box
sudo chown $USER:$USER /etc/sing-box
```

**4. 服务启动失败**
```bash
# 检查sing-box是否正确安装
which sing-box

# 检查配置文件语法
sing-box check -c /etc/sing-box/config.json
```

## 安全考虑

### 文件权限
- 配置文件设置为`0644`权限
- 目录权限设置为`0755`
- 脚本需要可执行权限

### 服务管理
- 优雅停止现有服务，避免数据丢失
- 启动前验证配置文件
- 提供详细的错误信息便于调试

## 部署最佳实践

### 1. 初次部署
```bash
# 1. 确保脚本权限
chmod +x bash/*.sh

# 2. 备份现有配置（如果有）
sudo cp /etc/sing-box/config.json /etc/sing-box/config.json.backup

# 3. 执行部署
./singbox_sub -v
```

### 2. 日常更新
```bash
# 简单更新配置
./singbox_sub

# 或详细查看过程
./singbox_sub -v
```

### 3. 故障恢复
```bash
# 恢复备份配置
sudo cp /etc/sing-box/config.json.backup /etc/sing-box/config.json

# 手动启动服务
bash/start_singbox.sh
```

## 兼容性

- ✅ **支持**: 标准Linux发行版
- ✅ **支持**: OpenWrt/LEDE系统
- ✅ **支持**: 容器化环境（需要适当权限）
- ❌ **不支持**: Windows WSL（文件权限限制）

现在Linux系统具有了完全自动化的sing-box部署能力！