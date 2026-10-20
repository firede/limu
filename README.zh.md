# limu

[English](README.md) | 简体中文

[![verify](https://github.com/firede/limu/actions/workflows/verify.yml/badge.svg?branch=main)](https://github.com/firede/limu/actions/workflows/verify.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/firede/limu.svg)](https://pkg.go.dev/github.com/firede/limu)

limu 是一个用 Go 写的图表布局引擎。它读取 Mermaid，绘制交给你的程序。程序可以用
[MyGo](https://mygo.egoist.dev) 原生绘制，可以输出 SVG，也可以画成自己的风格。

limu 的大部分代码移植自 Obsidian 团队的 Mermaid 渲染器
[Pele](https://github.com/obsidianmd/pele)。Pele 直接输出 SVG，limu 则早一步停下，
停在场景（scene），也就是以数据表示的排布结果。limu 的测试把它的结果和 Pele 的逐个
数值比对。

> [!WARNING]
> limu 是一个早期的个人项目，0.x 版本之间 API 随时可能变化。

## 快速开始

```sh
go get github.com/firede/limu
```

```go
sc, err := limu.Layout("flowchart LR\n  A[Text] --> B[Scene] --> C{Backend}", nil)
if err != nil {
	return err
}
out := svg.Render(sc, nil)
```

Pele 能画的图表类型 limu 都支持，从流程图、时序图到桑基图、C4 图和 Wardley 图。
要用 MyGo 绘制，请使用独立模块 `github.com/firede/limu/limumygo`。

## 文档

文档、每种图表的示例，以及可以试画自己图表的
[playground](https://limu.firede.com/playground) 都在
[limu.firede.com](https://limu.firede.com)（英文）。

- [Getting started](https://limu.firede.com/getting-started)
- [MyGo](https://limu.firede.com/mygo)、[SVG](https://limu.firede.com/svg) 和
  [Theming](https://limu.firede.com/theming)
- [API](https://limu.firede.com/api)
- [Architecture](https://limu.firede.com/architecture) 和
  [Compatibility](https://limu.firede.com/compatibility)

## 开发

```sh
./scripts/verify.sh
```

它会对两个模块运行 `go vet`、`gofmt` 和全部测试。在 Linux 上，limumygo 的测试需要
Pango（`libpangocairo-1.0-0`）。limu 的构建方式以及它如何与 Pele 比对，见
[docs/design.md](docs/design.md)，修改代码的规则见 [AGENTS.md](AGENTS.md)。

## 许可

MIT，见 [LICENSE](LICENSE)，其中也包含 Pele 的许可。
