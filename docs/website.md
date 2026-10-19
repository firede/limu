# Website

limu's website, at `limu.firede.com`, is to be made before 0.1.0 is released, so that
writing it shows what in the API and the packages' structure should change first. This
document holds what has been decided about it; the parts still open say so.

## Build and hosting

The site is built with Astro, as Pele's is, from a directory `website/` in this repository,
and is served by Cloudflare as the static assets of a Worker, as mygo-platinum's site is.
It is in English only. The import path stays `github.com/firede/limu`: the site's domain is
not a vanity import path.

A Go program, `website/gen`, run before Astro builds, writes what the site shows of limu's
own work: the mark and the icon; the SVG of every example, drawn by the svg package, and of
each Mermaid code block a document marks `draw` (` ```mermaid draw title="limu"
mark="scene" `), which the page shows in the code block's place while its Markdown keeps
the source; and `style.Kohu` and `style.KohuDark` as the `--limu-*` variables, so that the
diagrams follow the site's theme and the theme's colors are written in one place. It is in a module of its own, as the
`./...` of limu's would reach into `website/node_modules`. The PNG icons are drawn from
its SVGs by headless Chrome and committed (`website/scripts/icons.mjs`). No page but the
playground loads WebAssembly.

The diagrams' text is set in Arial, or a font as wide, as layout measures it with
`text.Arial`. Setting it in Hanken Grotesk, as the site's text is, would take a
`text.Measurer` reading that font's advances; that is left open.

The site's layout, navigation, code blocks and Markdown pages are adapted from Pele's
website, under its MIT license, which `website/PELE-LICENSE` holds.

## Pages

As on Pele's site, the pages are listed in a sidebar, from the home page, Introduction,
to the documents; on a narrow screen they are in a menu. The mark and the name head the
sidebar, with the playground's button below them; the link to GitHub and the switch
between light and dark are in the window's top right corner. The footer lists the pages
again in three groups, Start, Guides and Project, with the playground and GitHub among
them. There is no search, and no page comparing limu with Pele or with Mermaid: limu
matches Pele number for number, so a comparison would show nothing.

### Home

The home page says what limu is for, which is drawing diagrams natively, so its first code
example draws a diagram with limumygo; the SVG backend comes second. It credits Pele, which
most of limu is ported from.

Its section on how limu works is two pictures, which the Architecture page shows at more
length:

1. Where the pipeline branches. A renderer that writes SVG goes from text to a layout to an
   SVG string to a browser; limu goes from text to a layout to a scene, and branches there,
   to a backend: MyGo's GPU, SVG, or one of the host's own. This picture is drawn by limu.
2. One scene, laid out once and drawn four ways, one over another so that its geometry
   lines up: the scene as data, each item's box, parts, route and label's box with no
   style; the scene in Kohu; in KohuDark, which changes only the colors; and as Liquid
   Glass, an `Options.Draw` of its own that draws each node and edge label as a pane of
   MyGo's glass, which bends what is under it at its rim, as SVG in a browser hardly can.
   It shows that a diagram is laid out once, that a change of style only redraws, and how
   far the third level of style reaches. The look of mygo-platinum, which the plan first
   named, measures text in its own bitmap faces and so lays a diagram out again, and it
   depends on limu; the glass is MyGo's own, and draws the same scene.

A third picture may show `ItemAt` finding the item under the pointer in a MyGo window, as
what a scene being data gives a native app.

The first picture is drawn by `gen` with the svg package, as two diagrams, the renderer of
SVG's and limu's, with limu's scene in the accent. The second is drawn by
`limumygo/examples/pictures` into `website/public/pictures`: the scene as SVG in
currentColor, which takes the page's color, and the other three as PNG at twice their size.
`lib/pictures.mjs` writes them as a figure, which the home page shows, and which a
document shows where a paragraph holds nothing but their images, as Architecture does.
After the pictures come the code, a switch between MyGo and SVG with MyGo first, cards to
the examples, the playground, Getting started and Theming, and a paragraph on limu and Pele.

Given the website's font, Hanken Grotesk, as a TrueType file, the same program draws the
picture a link shows, under the sea that limu is named for: water drawn by a shader of
MyGo's, light near the surface, with shafts of light and the caustics the surface throws;
limu kohu, drawn with limu's mark, growing from the floor; on the left limu's name, the
lede's first sentence, a line on drawing with MyGo or as SVG, and the address; and on the
right how limu works, Mermaid, layout, the scene, MyGo and SVG, laid out by limu from the
bottom up and drawn as bubbles of clear glass, which rise from the limu and bend the light
behind them. `og.png`, 1200 by 630, is every page's `og:image` and `twitter:image`, with a
large card; `github.png`, 1280 by 640, is the repository's social preview, which GitHub
takes only through its settings. The font is not in the repository: the program reads the
file it is given, as Google Fonts serves it.

The pictures drawn natively are rendered with `ui.Render`, as limumygo's tests render
theirs, on a Mac, and committed: MyGo sets text with Pango on Linux, so CI would draw them
otherwise.

### Examples

`/examples` shows the featured example of each type, and `/examples/<type>` every example
of a type, from limu's gallery, `internal/gallery`, which the playground shows too. The
examples are limu's own, three to seven of each of the 32 types, one Markdown file for
each: the first one or two, featured, draw a scene a reader could meet, and the others each
show another feature of the type in a scene of its own. They are mostly scenes developers
know, and now and then one of the sea; none stands in for content with "Label", "A" or
"Alice". Four are written in Chinese, Japanese and Korean, a flowchart, a sequence diagram,
a mindmap and an entity relationship diagram, so that text in those scripts is laid out and
drawn whenever the examples are.
They replace the examples of Mermaid's documentation that the playground showed before,
which stay in limu's tests. Each diagram is SVG written at
build time, with "Copy source" and "Open in playground" beside it. A switch draws the
examples in Kohu, KohuDark or Default; as the SVG's colors are the `--limu-*` variables,
switching lays nothing out again. Until a theme is chosen, the examples follow the page's,
Kohu in the light and KohuDark in the dark, and a choice is kept as the page's own is.

As on Pele's site, each diagram is laid out at most 640, 480 and 320 wide, and the page
shows the widest its figure has room for, so that on a phone a diagram that can draw itself
narrower does rather than shrinking its text. A gantt chart takes 2026-01-15 in UTC as its
present, so that the build writes the same SVG whenever it runs.

"Open in playground" links to `/playground#source=…`, the source in the fragment encoded as
a URL's query, as Pele's playground takes it, and as the playground reads it. A source
whose link would pass 32,000 characters is not offered.

### Playground

`/playground` is the playground, compiled to WebAssembly, as an application of its own: it
fills the window under a narrow bar of its own, with the wordmark, links to the examples
and the documents, and the header's icons, and does not use the documents' layout.
`pnpm playground` (`scripts/playground.mjs`) builds it into `public/playground`, which is
not committed, and writes the files' names and size to `src/generated/playground.json`;
`pnpm build` does not, so that a change to the documents does not compile it again, and
fails when it has not been built. `pnpm deploy` builds both.

The web playground and the desktop one are one program, so that the two do not drift
apart, and both are made simpler: the playground is a place for a user to try limu, not an
application complete in features. It keeps the editor, the diagram and its SVG, light and
dark, a link to share, and the SVG and PNG to download, and an Examples menu of limu's
gallery. It drops opening and saving files and its page of examples, which the site's
examples replace, and its appearances in GitHub's colors, which Kohu and KohuDark replace,
the colors of its window, its code and its controls the site's, from `base.scss`, which a
test holds them to. A diagram comes in through the URL, as Pele's playground takes one,
and Copy link writes one; without the page of examples the playground needs no router,
which in a browser would own the URL's fragment. The shell of the browser and the desktop
differ by build tags (`desktop.go`, `web.go`): a desktop downloads through a dialog, offers
the appearance in a menu, remembers the text, and links to the site; a browser downloads
as a link with `download` does, and is light or dark as the page's switch sets it, through
`data-mygo-theme`. MyGo draws its menus in a browser as HTML in macOS's colors, as the
system's appearance has them, which the switch does not reach, so the page restyles them
in the site's. Under 640 pixels, the text is above the diagram.

Its WebAssembly is heavy, and its first load is the cost: compiling it and starting limu
take tens of milliseconds, downloading it takes seconds. So the page that loads it is plain
HTML that shows the diagram's source from the URL at once, where the editor will have it,
and the download's progress, and the files are named by their content and cached for good
(`public/_headers`), so that a second visit starts at once. Measured on 2026-10-20, built
with `GOOS=js GOARCH=wasm go build -trimpath -tags mygo_noinspector -ldflags="-s -w"`:

| Build | Size | gzip | brotli |
|---|---|---|---|
| the playground before it was made simpler | 24.4 MB (23.3 MiB) | 6.2 MB | 4.4 MB |
| the playground | 23.9 MB (22.8 MiB) | 6.1 MB | 4.4 MB |
| limu and svg alone | 12.2 MB | 3.3 MB | 2.3 MB |

Of the playground's code (16.7 MB, with 7.7 MB of data, before it was made simpler),
limu's diagrams are 6.1 MB, Go's runtime and standard library 5.5 MB, MyGo 4.0 MB (its `ui`
and `editor` 2.3 MB of it), and limu's other packages 0.7 MB. Cloudflare serves an asset
of at most 25 MiB, and compresses `application/wasm`. The playground's build and the
site's fail when the WebAssembly passes 24 MiB; past 25 MiB, it would be served from R2.

### Documents

| Page | What it covers |
|---|---|
| Getting started | Installing; `limu.Layout` and `svg.Render`, first, as they need only the standard library; `limu.Parse` once and `Layout` many times, as when the width changes; `Types` and `Supports`. |
| API | The `limu` package; `diagram.Options`, with `Config`, and `Limits` and what to set them to for text that is not trusted; the error codes; `layout.Options`; the scene, enough to walk it; measuring text, `Measurer` and `text.Arial`. Each section links to pkg.go.dev rather than repeating it. |
| Theming | The three levels of style: the theme's tokens, with the values of Kohu, KohuDark and Default; overrides by role; a backend's own drawing. Styles an author writes in the diagram, with `classDef` and `style`. |
| MyGo | Installing limumygo; `View`, and finding the item under the pointer with `ItemAt`; `Paint` on a `Painter`; `Fonts`, which measure and draw in the same fonts; `DrawIcon` and `DrawImage`; the editor's language in `limumygo/mermaid`; running in a browser. A section says that limumygo loads no image and follows no link: the host decides, through `DrawImage` and the pointer's events. |
| SVG | `svg.Render` and its `Options`: `Literal`, the fonts, `IconMarkup`, `Draw` and the `Writer`; setting the `--limu-*` variables in a page. A section on diagrams that are not trusted: the schemes links and images may use (`LinkSchemes`, `ImageSchemes`), and where icons' markup comes from. |
| Architecture | The stages, and what belongs to layout and what to style; that a backend of one's own walks the scene, in a sentence or two, as an agent can write one from there; the two pictures of the home page at more length; how limu is ported from Pele and checked against it. |
| Compatibility | The types limu reads; where it differs from Pele, as in what it does not port (`mount`, zoom) and the differences of the platform; where it differs from Mermaid, mostly as Pele does; how to report a difference. |

Two things about text in Chinese, Japanese and Korean belong in the documents. The API
page says that `text.Arial` only estimates such text, at an em a character, and wraps
Chinese and Japanese between characters as Pele does; the MyGo page says that `Fonts`
measure it in the fonts it is drawn in, and that the system's text engine puts a little
room where Han characters meet Latin letters and digits, which `Fonts` measure too.

MyGo comes before SVG, as drawing natively is what limu is for. `docs/design.md` stays as
it is, for those who work on limu; the Architecture page is its short form for those who
use it. The README may then be cut down to point to the site.

Every page is also served as Markdown, at its path with `.md`, and `/llms.txt` lists them,
for agents. The site has a sitemap, a `robots.txt` and a page for 404.

## Look

The site's colors are Limu kohu's, the red limu of Hawaiʻi in a tide pool; its type is
Hanken Grotesk. Both are chosen to stand apart from Pele's site, which is dark only, in
Flexoki's warm near-black with a neutral accent.

### Colors

The site has a light and a dark theme, which follow the system and can be switched. Its
own colors:

| | Dark | Light |
|---|---|---|
| page | `#0c1618` | `#f1f6f5` |
| panel (code, cards) | `#132125` | `#e3edeb` |
| rule | `#213439` | `#cddcda` |
| text | `#d0dddc` | `#0f262a` |
| muted text | `#7f989b` | `#536f72` |
| accent (links, the mark) | `#ee8494` | `#a92e4a` |

The same colors make a pair of themes in the style package, `style.Kohu()` and
`style.KohuDark()`; `style.Default()`, Pele's light theme, does not change. Their tokens:

| Token | `KohuDark` | `Kohu` |
|---|---|---|
| `Bg` | `#0c1618` | `#f1f6f5` |
| `Fg` | `#d0dddc` | `#0f262a` |
| `Muted` | `#7f989b` | `#536f72` |
| `Line` | `#b4c7c7` | `#23393d` |
| `Surface` | `#142428` | `#e5eeec` |
| `SurfaceAlt` | `#1b2f34` | `#d6e4e2` |
| `Border` | `#557176` | `#76939a` |
| `Accent` | `#ee8494` | `#a92e4a` |
| `Critical` | `#e3a040` | `#b0611a` |
| `Series` | `#8cbf6a` `#e37d90` `#5cbcb3` `#d9a748` `#78a3d9` `#ad92d1` `#e693b3` `#b9a272` | `#4f8a3a` `#b5435c` `#23817d` `#a8761c` `#3d6aa3` `#7b5aa0` `#c0668a` `#7d6a3f` |

`Radius` and `StrokeWidth` are Default's, 4 and 1. `Critical` is amber, not red, so that it
is not taken for the accent. The eight series are colors of algae: sea lettuce, limu kohu,
the tide pool's teal, kelp's gold, deep water, dulse, coralline and olive. In both themes,
text is at least 7:1 against the page and surfaces, muted text, links and lines at least
4.5:1, the series at least 3:1 against `Surface`, and outlines at least 2.5:1 against it
(WCAG 2).

### Mark and icon

The mark is the plume of limu kohu, in one color: the accent of the ground it is on, as
`currentColor`. On a box of 64, its stem is the cubic from (31, 60) through (28.5, 45) and
(36.5, 29) to (33, 5), 4.6 wide at its foot and 2.6 at its tip. Seven branchlets leave it,
alternating, at t = 0.18, 0.31, 0.44, 0.57, 0.69, 0.80 and 0.89, first to the left, of
lengths 17.5, 16.5, 14.5, 12.5, 10, 7.5 and 5. A branchlet of length L leaving the stem at
(x, y) to the side s (−1 left, 1 right) is the cubic from (x, y) through (x + 0.36sL,
y − 0.14L) and (x + 0.64sL, y − 0.34L) to (x + 0.78sL, y − 0.68L), 3.4 − 1.2t wide where it
leaves and 2 at its tip. Every stroke tapers evenly and ends round. The plume is centered
on its box by its bounds, then moved 0.6 to the right.

At 16 pixels the mark has a cut of its own, on a box of 16: the stem from (8, 15) through
(7.4, 11) and (9.1, 7) to (8.4, 1.2), 1.9 wide, and three branchlets of 1.7, at t = 0.3,
0.52 and 0.74, to the left, the right and the left, of lengths 6.6, 5.6 and 4, shaped as
above.

The icon, for the browser's tab, the playground's window and MyGo apps, is the plume in
the accent of the dark theme, `#ee8494`, scaled to 0.74 about the center and set 0.5 lower,
on a superellipse of exponent 5 filled with the dark page's `#0c1618`, as MyGo and Apple
round an icon's corners. A rim of `#2a4247`, 2 wide, inside its edge keeps its shape on
dark grounds. The favicon is the icon with the cut for 16 pixels, scaled to 0.8 about the
center, and a rim 1 wide.

The SVGs, and the script that draws them from these numbers, go into `website/` with the
site's skeleton.

### Type

The name and the site are set in Hanken Grotesk; the name in its weight 650, with a letter
spacing of −0.025em, the plume as tall as the l and a fifth of an em before it. Code is set
in the system's monospaced font.

The font is preloaded and declared with `font-display: optional`, not fontsource's `swap`,
so that a page is drawn once in it rather than first in the fallback font; `_headers` lets
browsers keep the files under `/_astro`, which are named by their content, for a year.

## Order of work

1. Add `style.Kohu` and `style.KohuDark`, with their tests. Every diagram on the site needs
   them, and they are the likeliest to show a problem in the API first.
2. The site's skeleton: Astro, the colors, the mark, the header and the footer, the layout
   of the documents, and the Go program that writes the examples' SVG.
3. The home page and the examples.
4. The playground made simpler, and its build for the web.
5. The documents.
6. The pictures of how limu works, rendered on a Mac; then deploying.
