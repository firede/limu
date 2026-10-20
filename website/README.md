# limu's website

The website at [limu.firede.com](https://limu.firede.com), built with Astro and served by
Cloudflare as a Worker of static assets. [docs/website.md](../docs/website.md) says what it
holds and how it looks.

It needs Node 22.13 or later, pnpm, and Go.

```sh
pnpm install
pnpm playground  # builds the playground for browsers, which dev and build need
pnpm dev         # writes what gen writes, then serves the site at localhost:4321
pnpm build       # the same, then builds the site into dist
pnpm check       # checks the site's types
pnpm run deploy  # builds the playground and the site, and deploys them with wrangler
```

Publishing a release of limu on GitHub deploys the site too, through
`.github/workflows/site.yml`.

## What is written, and from what

`gen`, a Go program in a module of its own, writes what the site shows of limu's own work
before Astro builds it:

- `public/mark.svg`, `mark-16.svg`, `icon.svg` and `icon-16.svg`: the mark and the icon,
  from the numbers in `gen/mark.go`, which docs/website.md describes.
- `src/generated/limu-theme.css`: `style.Kohu` and `style.KohuDark` as the `--limu-*`
  variables, for the site's light and dark themes.
- `src/generated/limu-theme.css` also has Kohu, KohuDark and Default under
  `[data-diagram-theme]`, for the examples' switch.
- `src/generated/examples.json`: every example of limu's gallery, `internal/gallery`, drawn as SVG at
  the three widths the pages show, 640, 480 and 320.
- `src/generated/home.json`: the home page's picture of where the pipeline branches, drawn
  the same way.

`scripts/playground.mjs`, which `pnpm playground` runs, compiles the playground,
`limumygo/examples/playground`, to WebAssembly, and writes:

- `public/playground/playground.<hash>.wasm` and `wasm_exec.<hash>.js`, Go's loader of it,
  named by their content, which `public/_headers` has browsers keep for good.
- `src/generated/playground.json`: their names and the WebAssembly's size, which the page
  that loads them, `src/pages/playground.astro`, reads.

It fails when the WebAssembly passes 24 MiB, as Cloudflare serves an asset of at most
25 MiB. Run it again when the playground or limu changes; `pnpm build` does not, so that a
change to the documents does not compile it again.

`src/generated` and `public/playground` are not committed; the rest of `public` is. `public/icon-180.png` and
`public/favicon.ico` are drawn from the SVGs by headless Chrome, as Go's standard library
has no rasterizer; run `node scripts/icons.mjs` after the mark changes.

## From Pele's website

The site's layout, its navigation, its code blocks and their highlighting, and the Markdown
each page is also served as, are adapted from [Pele's website](https://github.com/obsidianmd/pele/tree/main/website),
under the MIT license that [PELE-LICENSE](PELE-LICENSE) holds. limu changed its colors, its
type, its pages, and added highlighting for Go and a light theme.
