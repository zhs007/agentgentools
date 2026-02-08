# agentgentools

为 Agent 准备的 Go 多工具项目。每个工具通过命令行参数接收输入，结果输出到 `stdout`，并优先兼容 `tinygo` 编译到 WASM。

## 项目结构

```text
.
├── go.mod
└── src
    ├── host
    │   └── wasmrunner
    │       └── main.go
    └── tools
        └── dreamcard
            └── main.go
```

规则：

- `src/tools/<tool-name>/main.go`：一个目录一个工具。
- 工具输入：命令行参数（如 `--text=...`）。
- 数组参数：逗号分隔（如 `--tags=cat,horror,meta`）。
- 工具输出：写入 `stdout`（JSON 或文本）。
- 错误信息：写入 `stderr`，并返回非 0 退出码。
- 如需文件读写，优先由 Go 宿主控制 WASM 预打开目录与入参，避免工具自行开放任意路径访问。

## TinyGo 约束（必须遵守）

- 避免依赖反射的实现。
- 不要用标准库 `encoding/json`。
- 需要 JSON 时使用 `easyjson`（本项目示例直接使用 `github.com/mailru/easyjson/jwriter`）。

## 示例工具：dreamcard

工具路径：`src/tools/dreamcard/main.go`

输入命令行参数：

```bash
go run ./src/tools/dreamcard \
  --text="The stray cat you fed looks at you. 'Wake up, Jack,' it says in your father's voice. 'You're in a coma.'" \
  --type=weird \
  --phase=high \
  --outcome=win \
  --tags=cat,horror,meta \
  --mood=dark

# 或从文件读取文本，并将 JSON 写入指定文件
go run ./src/tools/dreamcard \
  --text-file=./input.txt \
  --out-file=./out.json \
  --type=weird \
  --phase=high \
  --outcome=win \
  --tags=cat,horror,meta \
  --mood=dark
```

输出 JSON（字段保持一致）：

```json
{
  "text": "The stray cat you fed looks at you. 'Wake up, Jack,' it says in your father's voice. 'You're in a coma.'",
  "type": "weird",
  "phase": "high",
  "outcome": "win",
  "tags": ["cat", "horror", "meta"],
  "mood": "dark"
}
```

## 运行与测试

```bash
go mod tidy

go run ./src/tools/dreamcard \
  --text="The stray cat you fed looks at you. 'Wake up, Jack,' it says in your father's voice. 'You're in a coma.'" \
  --type=weird \
  --phase=high \
  --outcome=win \
  --tags=cat,horror,meta \
  --mood=dark
```

## 编译为 WASM（TinyGo）

```bash
tinygo build -o dreamcard.wasm -target wasi ./src/tools/dreamcard
```

## Go 宿主执行 WASM（受控 HTTPS 读取 + 受控文件写入）

`src/host/wasmrunner` 提供一个示例宿主：

- 只允许读取 `--allow-urls` 中明确列出的 HTTPS URL（精确匹配）。
- 宿主下载远程文本后写入临时文件，并挂载到 WASM 的 `/in/input.txt`。
- 仅挂载 `--output-dir` 到 WASM 的 `/out`，工具只能写这个目录。

示例：

```bash
go run ./src/host/wasmrunner \
  --wasm=./dreamcard.wasm \
  --read-url=https://example.com/input.txt \
  --allow-urls=https://example.com/input.txt \
  --output-dir=./wasm-out \
  --output-file=result.json \
  --type=weird \
  --phase=high \
  --outcome=win \
  --tags=cat,horror,meta \
  --mood=dark
```

## 给 Agent 的新增工具模板

1. 新建目录 `src/tools/<new-tool>/`。
2. 编写 `main.go`，只做三件事：
   - 解析命令行参数；
   - 处理逗号分隔数组参数；
   - 将结果写到 `os.Stdout`。
3. 保持无状态、纯函数风格（输入决定输出），便于被其他 Agent 作为工具调用。
4. 编译命令统一：`tinygo build -target wasi ./src/tools/<new-tool>`。
