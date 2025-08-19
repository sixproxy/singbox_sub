# Homebrew发布设置指南

## 1. 创建Homebrew Tap仓库

### 步骤1：创建新仓库
在GitHub上创建一个新仓库，命名为 `homebrew-tap`
- 仓库名必须以 `homebrew-` 开头
- 设置为Public（Homebrew要求）
- 添加README和LICENSE

### 步骤2：初始化仓库结构
```bash
# 克隆你的tap仓库
git clone https://github.com/your-username/homebrew-tap.git
cd homebrew-tap

# 创建Formula目录
mkdir Formula

# 创建初始README
cat > README.md << 'EOF'
# Homebrew Tap for SingBox Sub

This tap contains formulae for SingBox subscription tools.

## Installation

```bash
# Add the tap
brew tap your-username/tap

# Install singbox-sub
brew install singbox-sub
```

## Available Formulae

- **singbox-sub**: Sing-box subscription configuration generator with SSR support
EOF

# 提交并推送
git add .
git commit -m "Initial tap setup"
git push origin main
```

## 2. 配置GitHub Secrets

在你的主项目仓库中添加以下Secret：

### 必需的Secrets：
1. **HOMEBREW_GITHUB_TOKEN**
   - 在GitHub生成Personal Access Token
   - 权限需要：`public_repo`, `workflow`
   - 用于推送到homebrew-tap仓库

### 添加步骤：
1. 去 GitHub Settings → Developer settings → Personal access tokens
2. 生成新token，选择权限：
   - `public_repo` (访问公共仓库)
   - `workflow` (更新GitHub Actions)
3. 复制token
4. 在项目仓库 Settings → Secrets and variables → Actions
5. 添加 `HOMEBREW_GITHUB_TOKEN` = 你的token

## 3. 发布流程

### 自动发布（推荐）
```bash
# 1. 确保代码已推送
git add .
git commit -m "feat: add new features"
git push origin main

# 2. 创建并推送标签
git tag v1.0.0
git push origin v1.0.0

# 3. GitHub Action会自动：
#    - 编译macOS版本
#    - 创建tar.gz包
#    - 计算SHA256
#    - 更新homebrew-tap仓库
#    - 创建或更新Formula
```

### 手动发布
```bash
# 如果你想手动更新Formula
cd homebrew-tap

# 编辑Formula
vim Formula/singbox-sub.rb

# 提交更改
git add Formula/singbox-sub.rb
git commit -m "Update singbox-sub to v1.0.0"
git push origin main
```

## 4. 用户安装方式

### 从你的tap安装：
```bash
# 添加tap
brew tap your-username/tap

# 安装
brew install singbox-sub

# 使用
singbox-sub --version
```

### 直接安装（如果发布到官方Homebrew）：
```bash
brew install singbox-sub
```

## 5. 提交到官方Homebrew（可选）

如果你的工具足够受欢迎，可以提交到官方Homebrew：

```bash
# 使用brew命令自动创建PR
brew bump-formula-pr --url=https://github.com/your-username/singbox_sub/releases/download/v1.0.0/singbox-sub-v1.0.0-darwin.tar.gz singbox-sub

# 或者手动创建PR到 homebrew/homebrew-core
```

## 6. 故障排除

### 常见问题：

1. **Permission denied**
   - 检查HOMEBREW_GITHUB_TOKEN权限
   - 确保token有访问homebrew-tap仓库的权限

2. **Formula validation failed**
   - 检查URL是否可访问
   - 验证SHA256是否正确
   - 确保二进制文件可执行

3. **Version conflicts**
   - 确保tag版本格式正确（如：v1.0.0）
   - 检查是否已存在相同版本

### 测试Formula：
```bash
# 本地测试
brew install --build-from-source your-username/tap/singbox-sub

# 验证安装
singbox-sub --version

# 卸载测试
brew uninstall singbox-sub
```

## 7. 高级配置

### 支持多个版本：
```ruby
class SingboxSub < Formula
  desc "Sing-box subscription configuration generator"
  homepage "https://github.com/your-username/singbox_sub"
  
  # 稳定版本
  url "https://github.com/your-username/singbox_sub/releases/download/v1.0.0/singbox-sub-v1.0.0-darwin.tar.gz"
  sha256 "abc123..."
  
  # 开发版本
  head "https://github.com/your-username/singbox_sub.git", branch: "develop"
  
  def install
    bin.install "singbox-sub-darwin-universal" => "singbox-sub"
  end
end
```

### 添加依赖：
```ruby
depends_on "go" => :build  # 构建时依赖
depends_on "openssl"       # 运行时依赖
```

## 8. 自动化测试

你可以添加额外的GitHub Action来测试Homebrew installation：

```yaml
# .github/workflows/test-homebrew.yml
name: Test Homebrew Installation
on:
  release:
    types: [published]

jobs:
  test-homebrew:
    runs-on: macos-latest
    steps:
    - name: Test installation
      run: |
        brew tap ${{ github.repository_owner }}/tap
        brew install singbox-sub
        singbox-sub --version
        brew uninstall singbox-sub
```