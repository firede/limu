# limu's website

The website at [limu.firede.com](https://limu.firede.com), built with Astro and served by
Cloudflare as a Worker of static assets. [docs/website.md](../docs/website.md) says what it
holds and how it looks.

It needs Node 22.13 or later, pnpm, and Go.

```sh
pnpm install
pnpm dev      # writes what gen writes, then serves the site at localhost:4321
pnpm build    # the same, then builds the site into dist
pnpm check    # checks the site's types
pnpm deploy   # builds it, and deploys it with wrangler
```

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

`src/generated` is not committed; `public` is. `public/icon-180.png` and
`public/favicon.ico` are drawn from the SVGs by headless Chrome, as Go's standard library
has no rasterizer; run `node scripts/icons.mjs` after the mark changes.

## From Pele's website

The site's layout, its navigation, its code blocks and their highlighting, and the Markdown
each page is also served as, are adapted from [Pele's website](https://github.com/obsidianmd/pele/tree/main/website),
under the MIT license that [PELE-LICENSE](PELE-LICENSE) holds. limu changed its colors, its
type, its pages, and added highlighting for Go and a light theme.
