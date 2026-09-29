# Excel Agent

一个"能理解 Excel"的智能体，基于 [CloudWeGo Eino](https://github.com/cloudwego/eino) 的 ADK（Agent Development Kit）构建。

本项目从 [cloudwego/eino-examples](https://github.com/cloudwego/eino-examples) 的 `adk/multiagent/integration-excel-agent` 示例独立而来，并做了以下改进：

- Windows 适配：`read_file` / `tree` 工具改为 Go 原生实现，不再依赖 GNU 工具链和 `python3` 命令；命令执行默认走 PowerShell（优先 pwsh，正确透传退出码并强制 UTF-8 输出）
- OpenAI 协议兼容接口支持：自定义 `OPENAI_BASE_URL`，可选关闭 `json_schema` 严格响应，计划输出解析容错（代码块 / 前后杂讯 / 轻微坏 JSON）
- 中文 Windows 下强制 Python 以 UTF-8 输出，避免 GBK 乱码

## 工作流程

给出一段自然语言需求和若干数据文件，Excel Agent 会：

1. **规划**（Planner）：把需求拆解为清晰、可执行的分步计划；
2. **执行**（Executor）：逐步生成并运行 Python 代码（pandas / openpyxl / matplotlib）处理数据，需要时联网搜索（WebSearchAgent）；
3. **重规划**（Replanner）：根据每步的执行结果修改、扩展计划，或确认计划完成；
4. **报告**（ReportAgent）：汇总执行过程与产物，产出最终报告和交付文件。

典型任务示例：

- 统计附件文件中推荐的小说名称及推荐次数，形成表格写入文件
- 读取 csv 中的表格内容，规范格式整理后写出
- 将表格的第一列提取到新文件

## 环境要求

| 依赖 | 说明 |
|---|---|
| Go | 1.27 及以上（Windows 请安装 64 位版本） |
| Python | 3.10 及以上，需安装 pandas / numpy / matplotlib / openpyxl，推荐用 [uv](https://docs.astral.sh/uv/) 管理虚拟环境 |

## 配置模型（必需，二选一）

**方式一：火山方舟 Ark**

```bash
export ARK_API_KEY=""    # （必填）Ark Model API Key
export ARK_MODEL=""      # （必填）Ark Model 名称或接入点 ID
export ARK_BASE_URL=""   # （可选）Ark Model base_url
export ARK_REGION=""     # （可选）Ark Model region
```

**方式二：OpenAI 协议兼容接口**（OpenAI / 智谱 / DeepSeek 等均可）

```bash
export OPENAI_API_KEY=""   # （必填）API Key
export OPENAI_MODEL=""     # （必填）模型名称
export OPENAI_BASE_URL=""  # 兼容接口必填，例如智谱: https://open.bigmodel.cn/api/paas/v4
```

| 变量 | 说明 |
|---|---|
| `OPENAI_BY_AZURE` | 使用 Azure OpenAI 时设为 `true`，此时 `OPENAI_BASE_URL` 填 Azure 端点 |
| `OPENAI_DISABLE_JSON_SCHEMA` | 设为 `true` 时跳过 `response_format=json_schema` 参数，用于不支持该特性的兼容接口 |

## 其他配置（可选）

| 变量 | 说明 |
|---|---|
| `EXCEL_AGENT_PYTHON_EXECUTABLE_PATH` | 执行 Python 代码所用的解释器，默认 `python`。**强烈建议指向虚拟环境**，否则 Agent 自动 `pip install` 依赖时可能被系统 Python 阻断导致任务失败 |
| `EXCEL_AGENT_INPUT_DIR` | 附件输入目录（绝对路径），默认 `playground/input` |
| `EXCEL_AGENT_WORK_DIR` | 工作目录（绝对路径），默认 `playground/<任务id>` |
| `EXCEL_AGENT_WINDOWS_SHELL` | Windows 下命令执行使用的 shell，默认 `powershell`（pwsh 优先），设为 `cmd` 回退到 cmd.exe |
| `ARK_VISION_API_KEY` / `ARK_VISION_MODEL` / `ARK_VISION_BASE_URL` / `ARK_VISION_REGION` | 视觉模型配置，配置后 ReportAgent 的 `image_reader` 工具才会启用 |
| `COZELOOP_WORKSPACE_ID` / `COZELOOP_API_TOKEN` | 接入 [CozeLoop](https://loop.coze.cn) 全链路追踪 |

## 快速开始

**1. 构建项目**

```bash
git clone git@github.com:wusongzhou/excel-agent.git
cd excel-agent
go build ./...
```

**2. 准备 Python 环境（推荐 uv）**

```bash
uv venv --python 3.12
# Windows:
uv pip install --python .venv/Scripts/python.exe pandas numpy matplotlib openpyxl
# macOS / Linux:
uv pip install --python .venv/bin/python pandas numpy matplotlib openpyxl
```

Windows 下持久化生效：

```powershell
setx EXCEL_AGENT_PYTHON_EXECUTABLE_PATH "C:\path\to\excel-agent\.venv\Scripts\python.exe"
```

**3. 放入待处理文件**

将需要处理的文件放入 `playground/input/`（或用 `EXCEL_AGENT_INPUT_DIR` 指定其他目录）。`playground/test_data/` 中提供了几个示例文件可直接使用：

```text
playground/test_data
├── questions.csv
├── 推荐小说.txt
└── 模拟出题.csv
```

**4. 修改任务描述并运行**

编辑 `main.go` 中的 query（目前已内置几个注释掉的示例，换行即可）：

```go
query := schema.UserMessage("请帮我将 questions.csv 表格中的第一列提取到一个新的 csv 中")
```

```bash
go run .
```

**5. 查看结果**

运行过程中控制台会实时输出计划、每步生成的代码及执行结果。所有产物在 `playground/<任务id>/` 目录下：

- 输入文件的完整副本
- `plan.md`：任务计划及各步骤执行状态
- `final_report.json`：ReportAgent 提交的最终结果与交付文件列表
- 生成的目标文件、中间产物及 Python 脚本

## 目录结构

```text
excel-agent
├── main.go              # 任务描述、Agent 组装与运行入口
├── operator.go          # 本地文件/命令操作实现
├── agents/
│   ├── planner/         # 计划生成
│   ├── executor/        # 计划执行（CodeAgent + WebSearchAgent）
│   ├── replanner/       # 计划重排
│   ├── report/          # 报告生成
│   └── wrap_plan.go     # 计划落盘 plan.md 的包装器
├── tools/               # bash / read_file / edit_file / tree / python_runner / image_reader / submit_result
├── generic/             # 计划结构、Excel 预览、结果结构
├── params/              # 上下文参数传递
├── utils/               # 模型构建与通用工具
├── common/              # 输出打印、CozeLoop 追踪
└── internal/logs/       # 彩色日志
```

## 注意事项

- Agent 会**真实执行** shell 命令和 Python 代码，均限制在工作目录内进行，请只放入你允许其处理的文件；
- Windows 下命令默认通过 PowerShell 执行（优先 pwsh / PowerShell 7，其次系统自带的 Windows PowerShell 5.1），工具描述已告知模型使用 PowerShell 语法；设置 `EXCEL_AGENT_WINDOWS_SHELL=cmd` 可回退；
- 若兼容接口上报格式相关错误（如 planner 解析失败），先尝试设置 `OPENAI_DISABLE_JSON_SCHEMA=true`。

## 许可

本项目以 [Apache-2.0](LICENSE-APACHE) 许可发布，原始代码 Copyright [CloudWeGo Authors](https://github.com/cloudwego/eino-examples)。
