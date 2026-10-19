// Flowcharts drawn end to end, some with a padding other than Pele's 8: what Pele's render
// draws, read back out of its SVG as one line per thing drawn, in the order drawn, with
// numbers as the SVG writes them. limu's test writes the same lines from its scene.
import { mutator } from './parse.mjs';

// The fragments mutations splice in, for text that parses and still draws something odd.
const FRAGMENTS = [
  '\n', ' --> ', ' --- ', ' -.-> ', ' ==> ', ' ~~~ ', ' --x ', ' --o ', ' <--> ', ' -->|label| ', ' -- text --> ',
  '[', ']', '(', ')', '{', '}', '((', '))', '[[', ']]', '([', '])', '[(', ')]', '{{', '}}', '[/', '/]', '>',
  'A', 'B', 'C', 'id1', 'long label that wraps past the width a node has', '"', '`**bold** _it_`', '<br>',
  '\nsubgraph S\n', '\nsubgraph T [Title]\n', '\nend\n', '\ndirection LR\n', '\ndirection TB\n', ' & ',
  ':::cls', '\nclassDef cls fill:#f96\n', '\nstyle A stroke:#f00\n', '\nlinkStyle 0 stroke:#0f0\n',
  '\nlinkStyle 0 stroke-linecap:butt,stroke-linejoin:ROUND\n', '\nlinkStyle default stroke-linecap:square\n',
  '\nlinkStyle 1 stroke-linecap:inherit,stroke-linejoin:bevel\n', '\nclassDef cls stroke-linecap:initial\n',
  '\nclick A "https://x.y/z" "tip"\n', '@{ shape: circle }', '@{ shape: hex }', '@{ shape: doc }',
  '@{ icon: "fa:user", form: circle, label: "U" }', '@{ img: "https://x/y.png", w: 60, h: 40 }',
  'e1@{ curve: linear }', 'e1@{ curve: step }', '中文', '😀', '#quot;', '\nA --> A\n',
];

// Pele's escapes in attribute values, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
const box = (s) => `${attr(s, 'x') ?? 0},${attr(s, 'y') ?? 0},${attr(s, 'width')},${attr(s, 'height')}`;
// Where a label's first line sits: the first x and y in its text.
const textAt = (s) => {
  const i = s.indexOf('<text');
  if (i < 0) return undefined;
  const m = / x="([^"]*)" y="([^"]*)"/.exec(s.slice(i));
  return `t ${m[1]},${m[2]}`;
};
const tag = (s, name) => {
  const i = s.indexOf('<' + name + ' ');
  return i < 0 ? undefined : s.slice(i, s.indexOf('>', i));
};

// The cap and the join a stroke is drawn with, as the browser settles them from the opening
// tags of the element and of the groups it is in, outermost first: each takes them from its
// parent, then from its own attributes, then from the declarations of its style attribute,
// in turn, of which the browser takes a keyword in either case, initial for SVG's initial
// value, and inherit or unset for its parent's, and drops any other.
const KEYWORDS = { 'stroke-linecap': ['butt', 'round', 'square'], 'stroke-linejoin': ['miter', 'round', 'bevel'] };
export function capJoin(chain) {
  const settle = (prop) => {
    let v = KEYWORDS[prop][0];
    for (const open of chain) {
      const parent = v;
      const a = attr(open, prop);
      if (a !== undefined && KEYWORDS[prop].includes(a.toLowerCase())) v = a.toLowerCase();
      for (const decl of (attr(open, 'style') ?? '').split(';')) {
        const at = decl.indexOf(':');
        if (at < 0 || decl.slice(0, at).trim().toLowerCase() !== prop) continue;
        const value = decl.slice(at + 1).trim().toLowerCase();
        if (KEYWORDS[prop].includes(value)) v = value;
        else if (value === 'initial') v = KEYWORDS[prop][0];
        else if (value === 'inherit' || value === 'unset') v = parent;
      }
    }
    return v;
  };
  return `${settle('stroke-linecap')} ${settle('stroke-linejoin')}`;
}

const GROUP = /<g class="pele-(cluster|edge|cluster-title|edge-label|node)[ "]/g;

// The lines of a flowchart's SVG.
export function readSvg(svg) {
  const lines = [];
  const head = tag(svg, 'svg');
  lines.push(`size ${attr(head, 'width')},${attr(head, 'height')}`);
  const title = svg.indexOf('<text class="pele-title"');
  if (title >= 0) lines.push('title ' + textAt(svg.slice(title)));
  const starts = [...svg.matchAll(GROUP)];
  for (let k = 0; k < starts.length; k++) {
    const kind = starts[k][1];
    const s = svg.slice(starts[k].index, k + 1 < starts.length ? starts[k + 1].index : svg.length);
    const open = s.slice(0, s.indexOf('>'));
    const id = attr(open, 'data-id');
    const classes = attr(open, 'class').split(' ').slice(kind === 'node' ? 2 : 1).join(' ');
    switch (kind) {
      case 'cluster': {
        if (classes.split(' ')[0] === 'pele-lane') {
          // A swimlane's lane: its outline, its title band, its title and how the title is turned.
          const rects = [...s.matchAll(/<rect [^>]*>/g)].map((m) => m[0]);
          lines.push(`lane ${id} [${classes.split(' ').slice(1).join(' ')}] ${box(rects[0])} band ${box(rects[1])}`);
          const t = textAt(s);
          if (t) lines.push(t);
          const turn = /<g transform="rotate\(([^)]*)\)">/.exec(s);
          if (turn) lines.push(`rotate ${turn[1]}`);
          break;
        }
        lines.push(`cluster ${id} [${classes}] ${box(tag(s, 'rect'))}`);
        const t = textAt(s);
        if (t) lines.push(t);
        break;
      }
      case 'edge': {
        const path = tag(s, 'path');
        let flags = '';
        if (path.includes(' stroke-width="2.5"')) flags += ' thick';
        if (path.includes(' stroke-dasharray="3 4"')) flags += ' dotted';
        lines.push(`edge ${id} [${classes}]${flags} ${attr(path, 'd')}`);
        // The edge's line, in the group of all of them and its own.
        lines.push(`stroke ${capJoin([tag(svg, 'g class="pele-edges"'), open, path])}`);
        for (const m of s.matchAll(/<(path|circle) class="pele-marker"[^>]*>/g)) {
          lines.push(m[1] === 'path' ? `m ${attr(m[0], 'd')}` : `m o ${attr(m[0], 'cx')},${attr(m[0], 'cy')}`);
        }
        break;
      }
      case 'cluster-title':
        lines.push(`struck ${id}`, textAt(s));
        break;
      case 'edge-label':
        lines.push(`label ${id} ${box(tag(s, 'rect'))}`, textAt(s));
        break;
      case 'node': {
        const shape = /pele-shape-([^\s"]+)/.exec(open)[1];
        lines.push(`node ${id} ${shape} [${classes}] ${attr(open, 'transform')}`);
        const image = tag(s, 'image');
        if (image) lines.push(`image ${attr(image, 'href')} ${box(image)}`);
        const icon = tag(s, 'svg class="pele-icon"');
        if (icon) lines.push(`icon ${attr(icon, 'data-icon')} ${box(icon)}`);
        const a = /<a [^>]*href="([^"]*)"/.exec(s);
        if (a) lines.push(`link ${unescape(a[1])}`);
        const t = textAt(s);
        if (t) lines.push(t);
        break;
      }
    }
  }
  return lines;
}

export default async function ({ src, cases, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { preprocess } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');

  const flowcharts = cases('flowchart').extracted.filter((s) => {
    try {
      return detect(preprocess(s).text) === 'flowchart';
    } catch {
      return false;
    }
  });

  const out = [];
  const draw = (source, maxWidth, padding) => {
    let r;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, padding, outputLimit: Infinity });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return false;
    }
    if (r.type !== 'flowchart') return false;
    out.push({ src: source, maxWidth, padding, lines: readSvg(r.svg) });
    return true;
  };
  const PADS = [0, 20, 2.5];
  let drawn = 0;
  const each = (source) => {
    if (!draw(source)) return;
    // Every third one again in a narrow host, which turns or tightens what does not fit, and
    // every twentieth with a padding other than Pele's 8, some in the narrow host, which
    // takes the padding from the room for the title and the groups.
    if (drawn % 3 === 0) draw(source, 360);
    if (drawn % 20 === 0) draw(source, drawn % 40 === 0 ? undefined : 360, PADS[(drawn / 20) % PADS.length]);
    drawn++;
  };
  flowcharts.forEach(each);
  const next = mutator(flowcharts, FRAGMENTS, random(11));
  while (drawn < 2000) {
    const s = next();
    if (s.length <= 3000) each(s);
  }
  write('diagram/flowchart/testdata/pele/render.json', { cases: out });
}
