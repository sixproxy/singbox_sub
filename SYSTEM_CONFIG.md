# 智能系统配置生成

## 概述

应用程序现在支持智能的系统检测和配置生成，根据运行的操作系统自动选择合适的配置模板，同时支持手动指定目标系统。

## 主要特性

### ✅ 自动系统检测
- **自动识别**: macOS (darwin)、Linux、Windows
- **智能选择**: 根据检测到的系统生成对应配置
- **日志记录**: 详细记录检测过程和配置生成状态

### ✅ 手动系统指定
- **灵活控制**: 可强制生成特定系统的配置
- **跨平台**: 在任何系统上生成其他系统的配置
- **批量生成**: 支持一次生成所有系统配置

### ✅ 命令行选项
- **系统选择**: `-os` 参数指定目标系统
- **详细输出**: `-v` 参数启用DEBUG日志
- **帮助信息**: `-h` 参数显示使用说明

## 使用方法

### 基本用法

```bash
# 自动检测系统并生成对应配置
./singbox_sub

# 显示帮助信息
./singbox_sub -h

# 启用详细输出模式
./singbox_sub -v
```

### 指定目标系统

```bash
# 强制生成macOS配置
./singbox_sub -os darwin

# 强制生成Linux配置  
./singbox_sub -os linux

# 强制生成Windows配置
./singbox_sub -os windows

# 生成所有类型的配置文件
./singbox_sub -os all
```

### 组合选项

```bash
# 生成所有配置并启用详细输出
./singbox_sub -os all -v

# 强制生成Linux配置并启用详细输出
./singbox_sub -os linux -v
```

## 系统配置差异

### macOS配置 (darwin)
- **文件名**: `mac_config.json`
- **特色功能**:
  - TUN接口配置
  - macOS特定的路由规则
  - 系统DNS集成
  - 自动路由配置

### Linux配置 (linux)
- **文件名**: `linux_config.json`  
- **特色功能**:
  - 通用网络配置
  - 适用于各种Linux发行版
  - 标准代理设置
  - 灵活的路由规则

### Windows配置 (windows)
- **文件名**: `linux_config.json` (通用配置)
- **说明**: 目前使用Linux配置作为Windows的通用配置

## 日志输出示例

### 自动检测模式
```
[2025-08-19 12:04:43] [INFO] [sub.go:99] 当前操作系统: darwin
[2025-08-19 12:04:43] [INFO] [sub.go:105] 使用自动检测的系统类型: darwin
[2025-08-19 12:04:43] [INFO] [sub.go:114] 开始生成macOS配置文件...
[2025-08-19 12:04:43] [INFO] [template.go:390] 配置文件已写入: mac_config.json
[2025-08-19 12:04:43] [INFO] [sub.go:118] macOS配置文件生成成功
```

### 强制指定系统
```
[2025-08-19 12:04:43] [INFO] [sub.go:99] 当前操作系统: darwin
[2025-08-19 12:04:43] [INFO] [sub.go:108] 使用指定的目标系统: linux
[2025-08-19 12:04:43] [INFO] [sub.go:123] 开始生成Linux配置文件...
[2025-08-19 12:04:43] [INFO] [template.go:263] 配置文件已写入: linux_config.json
[2025-08-19 12:04:43] [INFO] [sub.go:127] Linux配置文件生成成功
```

### 生成所有配置
```
[2025-08-19 12:04:43] [INFO] [sub.go:99] 当前操作系统: darwin
[2025-08-19 12:04:43] [INFO] [sub.go:108] 使用指定的目标系统: all
[2025-08-19 12:04:43] [INFO] [sub.go:144] 生成所有类型的配置文件...
[2025-08-19 12:04:43] [INFO] [sub.go:147] 生成Linux配置文件...
[2025-08-19 12:04:43] [INFO] [template.go:263] 配置文件已写入: linux_config.json
[2025-08-19 12:04:43] [INFO] [sub.go:152] Linux配置文件生成成功
[2025-08-19 12:04:43] [INFO] [sub.go:156] 生成macOS配置文件...
[2025-08-19 12:04:43] [INFO] [template.go:390] 配置文件已写入: mac_config.json
[2025-08-19 12:04:43] [INFO] [sub.go:161] macOS配置文件生成成功
[2025-08-19 12:04:43] [INFO] [sub.go:164] 所有配置文件生成完成，请根据你的系统选择合适的配置
```

## 错误处理

### 不支持的系统类型
```
[2025-08-19 12:04:43] [ERROR] [sub.go:171] 不支持的目标系统: invalid
[2025-08-19 12:04:43] [INFO] [sub.go:172] 支持的系统类型: auto, darwin, linux, windows, all
```

### 配置生成失败
```
[2025-08-19 12:04:43] [ERROR] [sub.go:116] 生成macOS配置文件失败: permission denied
```

## 使用场景

### 1. 开发环境
```bash
# 开发者在macOS上开发，生成所有平台配置
./singbox_sub -os all -v
```

### 2. CI/CD环境
```bash
# 在Linux CI环境中生成Linux配置
./singbox_sub -os linux
```

### 3. 跨平台部署
```bash
# 在macOS上为Linux服务器生成配置
./singbox_sub -os linux
```

### 4. 调试模式
```bash
# 启用详细日志进行问题诊断
./singbox_sub -v
```

## 优势

1. **智能化**: 自动检测系统类型，无需手动指定
2. **灵活性**: 支持强制指定目标系统
3. **便利性**: 一键生成所有类型配置
4. **透明度**: 详细的日志记录整个过程
5. **跨平台**: 在任何系统上都能生成其他系统的配置
6. **错误友好**: 清晰的错误提示和建议

现在应用程序具有了企业级的系统适配能力！