# limu

[English](README.md) | 简体中文

[![verify](https://github.com/firede/limu/actions/workflows/verify.yml/badge.svg?branch=main)](https://github.com/firede/limu/actions/workflows/verify.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/firede/limu.svg)](https://pkg.go.dev/github.com/firede/limu)

limu 是一个用 Go 写的图表布局引擎。它读取 Mermaid，绘制交给你的程序。程序可以用
[MyGo](https://mygo.egoist.dev) 原生绘制，可以输出 SVG，也可以画成自己的风格。
文档、示例和 playground 都在 [limu.firede.com](https://limu.firede.com)。

limu 的大部分代码移植自 Obsidian 团队的 Mermaid 渲染器
[Pele](https://github.com/obsidianmd/pele)，它小巧、快速。Pele 直接输出 SVG，limu
则早一步停下，停在场景（scene）。场景是以数据表示的排布结果，记录每个部件是什么、
在哪里，但不记录长什么样，绘制交给后端。limu 的测试把它的结果和 Pele 的结果逐个数值
比对。

> [!WARNING]
> **limu 是一个早期的个人项目**，0.x 版本之间 API 随时可能变化。它支持 Pele 能画的
> 所有图表类型，从流程图、时序图到桑基图、C4 图和 Wardley 图，完整列表见
> [Compatibility](https://limu.firede.com/compatibility)。

```go
sc, err := limu.Layout("flowchart LR\n  A[Text] --> B[Scene] --> C{Backend}", nil)
if err != nil {
	return err
}
out := svg.Render(sc, nil)
```

如果同一张图要排布多次，比如可用宽度变了，就只解析一次，再按需排布任意多次，也可以在
多个 goroutine 里同时排布。`Parse` 接受的限制和配置与 Pele 的 `parse` 相同。

```go
d, err := limu.Parse(src, diagram.Options{Config: diagram.Config{"seed": 7.0}})
if err != nil {
	return err
}
sc, err := d.Layout(&layout.Options{MaxWidth: 480, Padding: new(0.0)})
```

## 结构

```
文本 ──解析──▶ 模型 ──布局──▶ 场景 ──后端──▶ SVG、GPU、像素网格
```

| 包 | 作用 |
|---|---|
| `limu` | 解析并排布图表。`Parse` 让一张图解析一次、排布多次，`Types` 和 `Supports` 说明 limu 支持哪些类型。 |
| `diagram` | 读取所有图表共有的 front matter、指令和类型。每种类型在它下面有自己的包，负责解析和排布。各类型共用的代码在 `diagram/internal` 下，分层、复合和泳道布局在 `graphs`，形状和边的几何在 `draw`，作者用 `classDef` 和 `style` 写下的样式在 `css`，其余在 `common`。 |
| `text` | 用你提供的 `Measurer` 测量文本，并排版标签，处理分行、换行、强调、图标和实体。`text.Arial` 按一张 Arial 字宽表测量，不需要加载字体。 |
| `scene` | 描述排布好的图表，由带有角色、形状、盒子、路径、箭头和标签的部件组成。 |
| `style` | 存放按 token 取色的主题，以及按角色的覆盖。 |
| `layout` | 存放会改变位置的选项，比如测量器、字号、可用宽度、留白、网格对齐、边的走线方式，以及甘特图的当前时间和时区。 |
| `svg` | 把场景渲染成 SVG，主题颜色写成 CSS 变量。 |
| `limumygo` | 用 MyGo 的原生 UI 绘制场景，并找出指针下的部件。它是独立的 module，所以 limu 的其余部分只依赖标准库。其中的 `limumygo/mermaid` 包让 MyGo 的代码编辑器支持 Mermaid，按图表类型高亮关键字。 |

会改变位置的都属于布局，其余都是样式，改动样式只需重画。细节和路线图见
[docs/design.md](docs/design.md)。

想用自己的图表试试 limu，可以打开 [playground](https://limu.firede.com/playground)。
它把文本、limumygo 画出的图和 SVG 并排显示，Examples 菜单能打开 limu 的任一示例。
同一个程序也能在桌面上运行，命令是 `cd limumygo && go run ./examples/playground`。

## 样式

```go
look := style.Look{
	Theme: style.Default(), // Bg、Fg、Surface、Border、Accent、八个系列色……
	Overrides: style.Overrides{
		"flowchart.node": {Fill: style.Accent, StrokeWidth: style.N(2)},
		"flowchart.edge": {Dash: style.Dashes(3, 4)},
	},
}
out := svg.Render(sc, &svg.Options{Look: look})
```

limu 带有三个主题。`style.Default()` 是 Pele 的亮色主题，`style.Kohu()` 和
`style.KohuDark()` 是 limu 自己的亮色和暗色主题。部件的样式从主题开始逐层叠加，往上是
图表自身的样式和你为它的角色设的覆盖，最上面是作者用 `classDef` 和 `style` 写下的样式。
要用别的方式画某个部件，可以给后端一个函数来代替默认画法，函数里仍然可以画出默认的各个部分。

```go
svg.Render(sc, &svg.Options{Draw: func(w *svg.Writer, it *scene.Item, s style.Style, next func()) {
	if it.Shape != "diam" {
		next()
		return
	}
	// 用自己的方式写出菱形，再按默认方式写出它的标签。
	w.Label(it, s)
}})
```

## 测试

```sh
./scripts/verify.sh
```

这会在 limu 的 module 和 limumygo 的 module 里运行 `go vet`、`gofmt` 和全部测试。
下载好 limumygo 的依赖后无需联网。CI 在 amd64（`GOAMD64` 为 v1 和 v3）和 arm64 上运行它。
在 Linux 上，limumygo 的测试需要 Pango（`libpangocairo-1.0-0`）。

测试比对的 fixture 由 Pele 自己的源码生成。

```sh
./scripts/oracle.sh ../pele
```

这会用 Node 24 直接运行 Pele 的 TypeScript，无需安装任何依赖，跑完
`internal/oracle/suites` 里各个套件的用例，把结果写到各个包的 `testdata/pele`。
它只在 arm64 上运行。添加用例后，或要跟进 Pele 的新版本时运行它。

## 许可

MIT 许可，详见 [LICENSE](LICENSE)，其中也附有 Pele 的许可。
