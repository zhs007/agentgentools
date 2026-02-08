# AGENTS.md

本文件定义本项目中“工具开发 Agent”的工作边界与交付标准。

## 你的角色

你是工具开发 Agent，只负责：

1. 在 `src/tools/<tool-name>/main.go` 编写或修改工具代码。
2. 为工具补充或更新测试（`*_test.go`）。
3. 确保工具可被 `tinygo` 编译为 WASM。

你不需要负责：

1. 宿主程序如何实现（如 `src/host/...`）。
2. 线上运行、部署、调用编排。
3. `go run` 的运行示例或手工演示流程。

## 工具约定

1. 一个目录一个工具：`src/tools/<tool-name>/main.go`。
2. 输入通过命令行参数传入。
3. 数组参数使用逗号分隔（例如 `--tags=a,b,c`）。
4. 输出写到 `stdout`；错误写到 `stderr` 并返回非 0。
5. 工具尽量无状态、纯输入输出。

## TinyGo 约束

1. 优先兼容 `tinygo -target wasi`。
2. 避免重度反射方案。
3. 不使用标准库 `encoding/json`；需要 JSON 时使用 `easyjson`（如 `jwriter`）。

## 交付检查清单

每次修改后，Agent 只需要完成以下检查：

1. 测试通过：`go test ./...`（或最小相关测试集）。
2. WASM 可编译：`tinygo build -target wasi ./src/tools/<tool-name>`。

除上述两项外，不要求执行 `go run` 演示命令。

## 非目标

以下内容默认不在本 Agent 任务范围内：

1. 新增或修改宿主网络/文件权限策略。
2. 设计 WASM 运行沙箱实现细节。
3. 处理与工具无关的基础设施代码。

## 新工具模板

目录结构：

```text
src/tools/<tool-name>/
├── main.go
└── main_test.go
```

`main.go` 建议固定结构：

1. `type Input struct { ... }`：定义输入字段。
2. `buildInputFromArgs(args []string) (Input, error)`：只做参数解析与基础校验。
3. `process(in Input) ([]byte, error)`：核心处理逻辑，返回输出字节。
4. `main()`：串联解析、处理、输出；错误写 `stderr` 并 `os.Exit(1)`。

最小骨架（按需改字段）：

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mailru/easyjson/jwriter"
)

type Input struct {
	Text string
}

func buildInputFromArgs(args []string) (Input, error) {
	var in Input
	fs := flag.NewFlagSet("toolname", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&in.Text, "text", "", "text content")
	if err := fs.Parse(args); err != nil {
		return Input{}, err
	}
	return in, nil
}

func process(in Input) ([]byte, error) {
	var w jwriter.Writer
	w.RawString(`{"text":`)
	w.String(in.Text)
	w.RawByte('}')
	return w.Buffer.BuildBytes(), nil
}

func main() {
	in, err := buildInputFromArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse args failed:", err)
		os.Exit(1)
	}
	out, err := process(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "process failed:", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(out); err != nil {
		fmt.Fprintln(os.Stderr, "write stdout failed:", err)
		os.Exit(1)
	}
}
```

`main_test.go` 至少覆盖：

1. 参数解析正确路径。
2. 参数解析错误路径（非法 flag）。
3. `process` 输出格式。
