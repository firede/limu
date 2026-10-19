# limu

[English](README.md) | 简体中文

[![verify](https://github.com/firede/limu/actions/workflows/verify.yml/badge.svg?branch=main)](https://github.com/firede/limu/actions/workflows/verify.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/firede/limu.svg)](https://pkg.go.dev/github.com/firede/limu)

用 Go 解析并排布 Mermaid 图表，供自己负责绘制的程序使用：可以用
[MyGo](https://mygo.egoist.dev) 的原生 UI 在 GPU 上绘制，可以输出 SVG，也可以
画成程序自己的外观。

大部分代码移植自 Obsidian 团队的 Mermaid 渲染器
[Pele](https://github.com/obsidianmd/pele)，它小巧、快速。Pele 直接输出 SVG，
limu 则少走一步：它把图表排布成一个场景（scene），记录每个部件是什么、在哪里，
但不记录长什么样，绘制交给后端。测试会把 limu 的结果和 Pele 的结果逐个数值
比对。

> [!WARNING]
> **这是一个早期的个人项目。** 目前能读取流程图；状态图、类图、实体关系图、
> 需求图、用例图、agent flow 和泳道图；以及时序图、甘特图、时间线、用户旅程图、
> git 图、packet 图和看板；以及饼图、XY 图、象限图、雷达图、桑基图、树图和韦恩图。
> 其余图表会陆续加入。0.x 版本之间 API 随时可能变化。

```go
sc, err := limu.Layout("flowchart LR\n  A[Text] --> B[Scene] --> C{Backend}", nil)
if err != nil {
	return err
}
out := svg.Render(sc, nil)
```

## 结构

```
文本 ──解析──▶ 模型 ──布局──▶ 场景 ──后端──▶ SVG、GPU、像素网格
```

| 包 | 作用 |
|---|---|
| `limu` | `Layout`：把图表文本变成场景。 |
| `diagram` | 读取图表的 front matter 和指令，判断类型；每种类型有自己的包负责解析和排布。 |
| `text` | 通过宿主提供的 `Measurer` 测量文本，并排版标签：分行、换行、强调、图标、实体。`text.Arial` 用一张 Arial 字宽表测量，不需要字体。 |
| `scene` | 排布好的图表：各个部件、它们的角色和形状、盒子、路径、箭头、标签。 |
| `style` | 场景的外观，分三级：按 token 取色的主题、按角色的覆盖、后端自己的绘制。 |
| `layout` | 会改变位置的参数：测量器、字号、对齐网格、边的走线方式，以及甘特图的当前时间和时区。还有分层布局和复合布局。 |
| `draw` | 各类图表共用的几何：形状轮廓和边的路径。 |
| `svg` | 把场景画成 SVG，主题写成 CSS 变量。 |
| `mygoui` | 用 MyGo 的原生 UI 绘制场景，并找出指针下的部件。它是独立的 module，其余部分因此只依赖标准库。 |

会改变位置的参数交给布局；其余都属于样式，改动后只需重画。细节和路线图见
[docs/design.md](docs/design.md)。

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

一个部件的样式从下到上依次叠加：主题、图表自身的样式、该角色的覆盖、作者用
`classDef` 和 `style` 写下的样式。要用别的方式画某个部件，后端接受一个函数来
代替默认画法：

```go
svg.Render(sc, &svg.Options{Draw: func(w *svg.Writer, it *scene.Item, s style.Style, next func()) {
	if it.Shape != "diam" {
		next()
		return
	}
	// 用别的方式写出菱形
}})
```

## 测试

```sh
./scripts/verify.sh
```

运行 `go vet`、`gofmt` 和全部测试，无需联网。测试比对的 fixture 由 Pele 自己的
源码生成：

```sh
./scripts/oracle.sh ../pele
```

用 Node 24 直接运行 Pele 的 TypeScript，无需安装任何依赖，跑完
`internal/oracle/cases` 里的用例，把结果写到各个包的 `testdata/pele`。添加用例
后，或要跟进 Pele 的新版本时运行它。

## 许可

MIT，见 [LICENSE](LICENSE)，其中也附有 Pele 的许可。
