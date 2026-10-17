# TarsWingGuard

> **核心防御，开箱即用。羽翼加成，一键解锁。**
>
> 将整个 TSG 网关模块化、MCP 化。用户只要填一个本地参数，即可解锁"羽翼防护"等大部分高级功能。

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.1.0-green.svg)](https://github.com/HP-jh/TarsWingGuard/releases)

## 产品哲学

```
┌─────────────────────────────────────────────────────────┐
│                    TarsWingGuard                        │
├─────────────────────────────────────────────────────────┤
│  【核心 Core】                【羽翼 Wings】              │
│  所有用户一上来就有：          填 1 个参数，解锁高级模块：  │
│  · 基础 WAF                    · 羽翼防护 (WAF高级+RASP)  │
│  · 限流 / RBAC                 · 知识大脑 (RAG+Mem0)     │
│  · 审计 / 沙箱                 · 视觉雷达 (UI-TARS)      │
│                                · 智能体沙箱 (gVisor)     │
│                                · 多渠道 / 画板 / 监控    │
│                                · 极致档 / 协作编排       │
│                                · 企业版 (SSO+合规)       │
└─────────────────────────────────────────────────────────┘
```

## 安装（一行命令）

**Linux / macOS:**
```bash
curl -fsSL https://raw.githubusercontent.com/HP-jh/TarsWingGuard/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
iwr -useb https://raw.githubusercontent.com/HP-jh/TarsWingGuard/main/install.ps1 | iex
```

**6 步小白向导：**
```
[1/6] 欢迎使用 TarsWingGuard 安装向导
[2/6] 检查依赖...
[3/6] 下载 twg...
[4/6] 校验文件...
[5/6] 安装到 /usr/local/bin...
[6/6] 初始化配置...
✅ 安装完成！
```

## 快速开始

### 方式一：套餐模式（推荐）

在 TSG `config.json` 中填 **1 个参数**：

```json
{
  "wing": {
    "mode": "tier",
    "tier": "pro"
  }
}
```

三档套餐：
- `basic` — 核心防御（3 模块：WAF / RBAC / 审计）
- `pro` — 羽翼防护（9 模块：basic + 羽翼防护/知识大脑/视觉雷达/多渠道/画板/监控）
- `ultimate` — 旗舰全开（13 模块：pro + 极致档/协作编排/企业版/智能体沙箱）

### 方式二：解锁模式

```json
{
  "wing": {
    "mode": "unlock",
    "unlock": "羽翼防护"
  }
}
```

任意非空解锁词即可启用 pro 套餐。

### 方式三：模块自选

```json
{
  "wing": {
    "mode": "modules",
    "modules": ["wing-guard", "knowledge", "monitor"]
  }
}
```

自动解析依赖链（如 `wing-guard` 依赖 `core-waf`，会自动带上）。

## 模块目录

| 模块 | 名称 | 套餐 | 说明 |
|------|------|------|------|
| `core-waf` | 核心防火墙 | basic | 基础 WAF + 限流 + IP黑白名单 |
| `core-rbac` | 核心权限 | basic | RBAC 基础实现 |
| `core-audit` | 核心审计 | basic | 审计日志 + 哈希链 |
| `wing-guard` | **羽翼防护** | pro | 高级 WAF + RASP + 零信任 |
| `knowledge` | 知识大脑 | pro | RAG + Mem0 + 工作流 |
| `vision` | 视觉雷达 | pro | UI-TARS Bridge + 截图分析 |
| `multi-channel` | 多渠道网关 | pro | IM / Email / Webhook |
| `whiteboard` | 画板工作流 | pro | Mermaid / PlantUML 渲染 |
| `monitor` | 监控雷达 | pro | Prometheus + Grafana |
| `extreme` | 极致档 | ultimate | 深度防御 + 行为分析 |
| `orchestration` | 协作编排 | ultimate | 多智能体协作 (LobeHub) |
| `enterprise` | 企业版 | ultimate | SSO / 审计合规 / 组织管理 |
| `agent-sandbox` | 智能体沙箱 | ultimate | gVisor + seccomp 隔离 |

## CLI 工具

```bash
# 列出所有模块
wingctl -list

# 查看 pro 套餐包含哪些模块
wingctl -resolve pro

# 查看三档套餐对比
wingctl -tiers

# 从 TSG config.json 解析启用的模块
wingctl -tsg-config /opt/tsg/config.json

# 安装前干跑验证
wingctl -catalog wing.yaml -dry-run
```

## MCP Server Demo

`wing-guard` 是一个完整的 MCP server 实现，支持 stdio / SSE / streamable 三 transport：

```bash
# stdio 模式（与 Claude Desktop / Cursor 等集成）
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | wing-guard -transport stdio

# 扫描威胁
# -> tools/call wing_guard_scan {"payload":"<script>alert(1)</script>"}
```

## 项目结构

```
TarsWingGuard/
├── cmd/
│   ├── wingctl/          # 目录 CLI 工具
│   └── wing-guard/       # 羽翼防护 MCP server 演示
├── pkg/
│   ├── catalog/          # 模块目录解析
│   ├── tier/             # 套餐解析器 (basic/pro/ultimate)
│   ├── mcp/              # MCP server 框架 (stdio/SSE/streamable)
│   └── config/           # TSG config.json 集成
├── internal/
│   └── guard/            # wing-guard 模块实现
├── wing.yaml             # 官方模块清单（13 模块）
├── wing.example.yaml     # 配置模板
├── install.sh            # Linux/macOS 一键安装
├── install.ps1           # Windows 一键安装
└── Makefile              # 构建脚本
```

## 独立仓库体系

每个大模块也是独立仓库，本仓库作为 **catalog 中心**：

| 仓库 | 用途 |
|------|------|
| `HP-jh/TarsWingGuard` | **本仓库** — 注册中心 + 套餐逻辑 |
| `HP-jh/TarsWingKnowledge` | 知识大脑模块 |
| `HP-jh/TarsWingVision` | 视觉雷达模块 |
| `HP-jh/TarsEnterprise` | 企业版模块 (B36) |
| `HP-jh/TarsOpenClawSandbox` | 智能体沙箱模块 (B37) |

## 路线图

- **v0.1.0** ✅ 仓库骨架 + 13 模块 catalog + tier 解析器 + MCP demo + 安装向导
- **v0.2.x** 接入 B33 完成的 30+ 版本能力（逐模块上线）
- **v0.3.0** 接入 B36 TarsEnterprise 企业版
- **v0.4.0** 接入 B37 OpenClawSandbox 智能体沙箱
- **v1.0.0** 收官：红蓝对抗 + 用户文档 + 营销站

## 价格锚定

| 版本 | 对应套餐 | 适合人群 |
|------|---------|---------|
| 社区版 | basic | 个人开发者、小型项目 |
| 专业版 | pro | 中小团队、生产环境 |
| 旗舰版 | ultimate | 大型企业、合规要求 |

## License

Apache-2.0
