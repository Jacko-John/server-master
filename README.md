# ServerMaster

<div align="center">

**一个强大的 Clash 订阅与规则集管理解决方案**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

---

## 简介

ServerMaster 是一个基于 Go 语言开发的代理配置管理套件,专门用于处理 Clash 订阅和规则集的统一管理与分发。它采用客户端-服务端架构,提供灵活的订阅合并、规则集缓存、定时任务等核心功能。

### 核心组件

- **ServerMaster** - 中心化配置管理服务,负责订阅生成、规则集管理和 API 服务
- **SMClient** - 本地同步客户端,支持配置同步和 Mihomo 内核进程管理

---

## 主要特性

### 服务端能力

- 订阅合并与管理 - 支持本地节点与多个外部订阅源的智能合并
- 规则集缓存 - 自动下载和缓存远程规则集文件,支持本地分发
- 动态端口映射 - 通过 iptables 实现端口随机化,提升安全性
- 定时任务调度 - 灵活的 cron 任务系统,支持后台自动更新
- 灵活配置管理 - 基于 YAML 的配置文件,支持多租户 Token 认证

### 客户端能力

- 配置同步 - 自动从 ServerMaster 拉取最新配置
- 进程守护 - 内置 Mihomo (Clash Meta) 内核管理器,支持自动重启
- 本地订阅 - 可添加独立的第三方订阅源并合并到本地配置
- 配置覆盖 - 灵活覆盖服务端下发的配置参数 (DNS、端口等)
- 守护模式 - 支持后台运行并定时更新配置

---

## 快速开始

### 环境要求

- Go 1.21 或更高版本
- Linux 系统 (iptables 功能需要 root 权限)

### 构建项目

```bash
# 克隆仓库
git clone https://github.com/yourusername/server-master.git
cd server-master

# 构建所有组件
make build

# 或者单独构建
make build-server   # 构建 ServerMaster
make build-client   # 构建 SMClient
```

### 服务端配置

1. 复制配置示例文件:
```bash
cp example/config.yaml workspace.d/config.yaml
```

2. 编辑配置文件,设置必要的参数:
```yaml
tokens:
  - "your-secret-token"    # 必需: 访问 Token

proxy-path: "workspace.d/proxy.yaml"   # 本地节点配置
rule-path: "workspace.d/ruleset/"      # 规则集目录
```

3. 启动服务:
```bash
./ServerMaster -c workspace.d/config.yaml
```

服务将监听在 `:8080` (默认),可通过以下地址访问订阅:
```
http://your-server:8080/sub?token=your-secret-token
```

### 客户端使用

1. 复制客户端配置:
```bash
cp example/client.yaml client.yaml
```

2. 编辑配置,指向服务端地址:
```yaml
server-url: "http://your-server:8080/sub?token=your-secret-token"

mihomo:
  enable: true
  bin-path: "/usr/local/bin/mihomo"
  work-dir: "./mihomo"
```

3. 运行客户端:
```bash
# 单次同步
./SMClient -c client.yaml

# 守护模式 (推荐)
./SMClient -c client.yaml -d
```

---

## 配置说明

### 服务端配置 (config.yaml)

```yaml
# 基础设置
listen: ":8080"              # 监听地址
gin-mode: "release"          # 运行模式

# 日志配置
log:
  level: "info"              # 日志级别: debug/info/warn/error
  format: "json"             # 格式: json/text

# 认证设置 (必需)
tokens:
  - "token1"
  - "token2"

# 文件路径
proxy-path: "workspace.d/proxy.yaml"
rule-path: "workspace.d/ruleset/"
log-path: "server.log"

# 订阅信息
subscription:
  filename: "MyConfig.yaml"
  update-interval: 18        # 更新间隔 (小时)
  profile-url: "https://your-site.com"

# 外部订阅合并
additions:
  - url: "https://remote-sub.com/sub"
    group-name: "香港节点"
    group-type: "select"
    prepend-rules:
      - "DOMAIN-SUFFIX,google.com,香港节点"

# 定时任务
cron:
  # 动态端口映射
  dynamic-port:
    enable: false
    min: 10000
    max: 65535
    active-num: 3
    trojan-port: 443
    cycle: "@every 1m"

  # 规则集自动更新
  rule-set:
    enable: true
    cycle: "@every 1h"
    direct:
      - "https://raw.githubusercontent.com/.../direct.txt"
    proxy:
      - "https://raw.githubusercontent.com/.../proxy.txt"
    reject:
      - "https://raw.githubusercontent.com/.../reject.txt"
```

### 客户端配置 (client.yaml)

```yaml
# 服务端连接
server-url: "http://server:8080/sub?token=xxx"

# 同步设置
update-interval: 15          # 更新间隔 (分钟)
config-path: "./config.yaml"

# 日志配置
log:
  level: "info"
  path: "./client.log"
  format: "text"

# Mihomo 管理
mihomo:
  enable: true
  bin-path: "/usr/local/bin/mihomo"
  work-dir: "./mihomo"
  log-path: "mihomo.log"

# 本地规则 (最高优先级)
prepend-rules:
  - "DOMAIN-SUFFIX,google.com,Proxy"
  - "IP-CIDR,192.168.0.0/16,DIRECT"

# 配置覆盖
overrides:
  mixed-port: 7890
  allow-lan: true
  mode: "rule"
  dns:
    enable: true
    enhanced-mode: "fake-ip"
    nameserver:
      - "https://dns.alidns.com/dns-query"

# 本地额外订阅
additions:
  - url: "https://other-sub.com/config.yaml"
    group-name: "Other-Group"
    group-type: "select"
```

---

## 项目架构

```
server-master/
├── cmd/
│   ├── server/          # ServerMaster 入口
│   └── client/          # SMClient 入口
├── internal/
│   ├── app/             # 应用生命周期管理
│   ├── api/             # HTTP 处理器和路由
│   ├── service/         # 核心业务逻辑
│   ├── model/           # 数据结构定义
│   ├── config/          # 配置加载与验证
│   └── client/          # 客户端逻辑
├── pkg/
│   ├── logger/          # 结构化日志工具
│   └── utils/           # 通用工具库
├── example/             # 配置文件示例
├── workspace.d/         # 工作目录 (运行时生成)
└── makefile             # 构建脚本
```

### 技术栈

- **Web 框架**: Gin
- **日志**: log/slog
- **并发**: golang.org/x/sync/errgroup
- **配置**: YAML (gopkg.in/yaml.v3)
- **进程管理**: os/exec + 信号处理

---

## API 接口

### 获取订阅配置

```
GET /sub?token={TOKEN}
```

返回合并后的 Clash 配置文件。

### 获取规则集文件

```
GET /file/{filename}
```

返回指定名称的规则集文件 (从 `rule-path` 目录)。

---

## 开发指南

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/service/...
go test -v ./internal/service/...

# 运行单个测试
go test -run TestGenerateConfig ./internal/service/
```

### 代码风格

- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 所有公开的函数添加注释
- 错误处理优先使用 `errors.Wrap`

### 添加新的 Cron 任务

1. 在 `internal/service/` 下创建新文件
2. 实现 `Task` 接口 (必需):
```go
type Task interface {
    Name() string           // 任务唯一标识
    Spec() string           // Cron 表达式 (如 "@every 1h")
    Run()                   // 任务执行逻辑
}
```
3. 可选实现初始化和清理接口:
```go
type Initializer interface {
    Init() error            // 任务启动前执行
}

type Cleaner interface {
    Cleanup()               // 任务停止时执行
}
```
4. 在 `container.go` 的 `Container` 结构体中添加服务字段
5. 在 `NewContainer` 函数中初始化服务
6. 在 `app.go` 中通过 `cronService.AddTask()` 注册任务

---

## 常见问题

**Q: 客户端无法连接服务端?**

A: 检查以下几点:
- 服务端是否正常运行
- 防火墙是否开放对应端口
- Token 配置是否正确
- 网络是否可达

**Q: Mihomo 进程无法启动?**

A: 确认:
- `bin-path` 路径是否正确且有执行权限
- `work-dir` 目录是否存在且有写入权限
- 查看客户端日志获取详细错误信息

**Q: 规则集不更新?**

A: 检查:
- `cron.rule-set.enable` 是否为 `true`
- 远程 URL 是否可访问
- 查看服务端日志中的下载记录

---

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

---

## 贡献

欢迎提交 Issue 和 Pull Request!

## 致谢

- [Mihomo](https://github.com/MetaCubeX/mihomo) - Clash Meta 内核
- [Loyalsoldier/clash-rules](https://github.com/Loyalsoldier/clash-rules) - 优秀规则集

---

<div align="center">

Made with ❤️ by Jacko-John

</div>