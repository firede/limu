// Entity relationship diagrams drawn end to end: what Pele's render draws, read back out of
// its SVG as one line per thing drawn, in the order drawn, with numbers as the SVG writes
// them. limu's test writes the same lines from its scene.
import { lone, mutator } from './parse.mjs';
import { FRAGMENTS as PARSE_FRAGMENTS, WRITTEN as PARSE_WRITTEN } from './er.mjs';

// Diagrams that reach each part of the drawing: tables, loops, subgraphs, ends spread along a
// side, styles that change sizes, links.
const WRITTEN = [
  'erDiagram\nA ||--|| A : self',
  'erDiagram\nA ||--|| A : one\nA |o--o{ A : two\nA }|..|{ A : "a longer label"\nA ||--|| A : ""',
  'erDiagram\ndirection LR\nA ||--|| A : one\nA |o--o{ A : two\nB { int id }\nB ||--o{ B : three\nA ||--o{ B : x',
  'erDiagram\ndirection LR\nsubgraph S\ndirection TB\nA ||--|| A : loop\nend\nsubgraph T\nB ||--|| B : loop\nend',
  'erDiagram\nsubgraph S\nA\nB\nend\nS ||--o{ S : "around the group"\nS ||--|| C : x',
  'erDiagram\nA ||--o{ B : one\nA ||--o{ B : two\nA ||--o{ B : three\nA ||--o{ C : four\nA ||--o{ D : five\nE ||--o{ B : six',
  'erDiagram\nA { int id }\nA ||--o{ B : a\nA ||--o{ C : b\nA ||--o{ D : c\nA ||--o{ E : d\nA ||--o{ F : e\nA ||--o{ G : f\nA ||--o{ H : g',
  'erDiagram\ndirection LR\nA ||--o{ B : a\nA ||--o{ C : b\nA }o--o{ D : c\nB ||--|{ D : d\nC u--o| D : e',
  'erDiagram\ndirection RL\nA ||--o{ B : a\nB ||--o{ C : b\nC ||--o{ A : c',
  'erDiagram\ndirection BT\nA ||--o{ B : a\nB ||..o{ C : b',
  'erDiagram\nCUSTOMER {\n  string name PK "The name of the customer, which is quite long and wraps onto a second line because it is long"\n  int age\n  string email UK\n}\nCUSTOMER ||--o{ ORDER : places',
  'erDiagram\nT {\n  Map~string,int~ m PK, FK\n  List~List~int~~ nested\n  string? opt "**bold** and _it_"\n  `my type` `my name`\n}',
  'erDiagram\nT {\n  a b\n}\nU {\n  a b PK\n}\nV {\n  a b "c"\n}\nW {\n  a b PK "c"\n  d e\n  f g\n  h i\n}',
  'erDiagram\np[Person] {\n  string firstName\n}\na["A very long alias that wraps onto more than one line because it is long"]\np ||--o| a : has',
  'erDiagram\nA\nB\nC\nstyle A fill:#f9f,stroke:#333,stroke-width:4px\nstyle B font-size:20px,font-weight:bold\nstyle C font-style:italic,color:red\nA ||--o{ B : x\nB ||--o{ C : y',
  'erDiagram\nA {\n  int id PK\n  string name\n}\nstyle A fill:#f96,font-size:11px\nclassDef big font-size:24px,font-weight:700\nB:::big {\n  int id PK\n}\nclass A big',
  'erDiagram\nA:::internal-link\nB["Note"]:::internal-link\nC["javascript:alert(1)"]:::internal-link\nA ||--o{ B : x',
  'erDiagram\nsubgraph S [A long title for the subgraph]\nA ||--o{ B : x\nend\nC ||--o{ A : y\nC ||--o{ B : z',
  'erDiagram\nsubgraph O\nsubgraph I\ndirection LR\nA ||--o{ B : x\nend\nC\nend\nI ||--o{ C : y\nO ||--o{ D : z',
  'erDiagram\nsubgraph S\nA\nend\nsubgraph T\nB\nend\nA ||--o{ B : a\nA ||--o{ B : b\nS ||--|| T : c',
  '---\ntitle: A title that is much wider than the diagram under it\n---\nerDiagram\nA ||--o{ B : x',
  '%%{init: {"er": {"nodeSpacing": 10, "rankSpacing": 20}}}%%\nerDiagram\nA ||--o{ B : x\nA ||--o{ C : y',
  '%%{init: {"er": {"nodeSpacing": 5000, "rankSpacing": -1}}}%%\nerDiagram\nA ||--o{ B : x\nA ||--o{ C : y',
  'erDiagram\ndirection LR\nA ||--o{ B : x\nB ||--o{ C : y\nC ||--o{ D : z\nD ||--o{ E : w\nE ||--o{ F : v',
  'erDiagram\nA ||--|| B : "<br>"\nA ||--|| C : "line one<br>line two"\nA ||--|| D : "fa:fa-car car"',
  'erDiagram\n中文 ||--o{ 日本語 : 関係\n中文 {\n  文字列 名前 PK "説明"\n}',
  // An entity laid out larger than it is drawn, for its loops, that edges from outside its
  // subgraph reach on the side it grew on.
  'erDiagram\nX ||--o{ A : in\nsubgraph S\ndirection LR\nA ||--|| A : self\nA ||--o{ B : x\nend\nB ||--o{ Y : out',
  'erDiagram\ndirection LR\nX ||--o{ A : in\nsubgraph S\ndirection TB\nA ||--|| A : self\nA ||--o{ B : x\nend\nB ||--o{ Y : out',
  'erDiagram\nX ||--o{ A : in\nZ ||--o{ A : in\nsubgraph S\ndirection RL\nA { int id }\nA ||--|| A : self\nA ||--|| A : again\nend',
  'erDiagram\ndirection RL\nX ||--o{ A : in\nsubgraph S\ndirection BT\nA ||--|| A : self\nend\nA ||--o{ Y : out',
  // Ranks close enough together that an edge between them is drawn straight.
  ...[10, 20, 30, 40, 50].map(
    (r) => `%%{init: {"er": {"rankSpacing": ${r}, "nodeSpacing": 10}}}%%\nerDiagram\nA ||--o{ B : ""\nA ||--o{ C : ""\nA ||--o{ D : ""\nB }o--o{ E : ""`
  ),
];

// The fragments mutations splice in, for text that parses and still draws something odd.
const FRAGMENTS = [
  ...PARSE_FRAGMENTS,
  '\nA ||--o{ B : x\n', '\nA ||--|| A : self\n', '\nsubgraph S\n', '\nend\n', '\ndirection LR\n', '\ndirection TB\n',
  '\nX {\n  int id PK "comment"\n  string name\n}\n', ' "a long comment that wraps past the width a comment has, as it is long" ',
  '\nstyle A fill:#f96\n', '\nstyle A font-size:22px\n', ':::internal-link', '\nclassDef c font-weight:bold\nclass A c\n',
];

// Pele's escapes in attribute values, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
const tag = (s, name) => {
  const i = s.indexOf('<' + name + ' ');
  return i < 0 ? undefined : s.slice(i, s.indexOf('>', i));
};
const tags = (s, name) => [...s.matchAll(new RegExp(`<${name} [^>]*>`, 'g'))].map((m) => m[0]);

// A path's d with its horizontal and vertical lines written as lines to a point, as limu's
// paths have them.
function lines(d) {
  let out = '';
  let x = '';
  let y = '';
  for (const [, op, args] of d.matchAll(/([MLHVCQZ])([^MLHVCQZ]*)/g)) {
    if (op === 'H') {
      x = args;
      out += `L${x},${y}`;
    } else if (op === 'V') {
      y = args;
      out += `L${x},${y}`;
    } else {
      out += op + args;
      if (args !== '') [x, y] = args.split(' ').at(-1).split(',');
    }
  }
  return out;
}

const GROUP = /<g class="pele-(cluster|edge|edge-label|node)[ "]/g;

// The lines of an ER diagram's SVG, drawn with its numbers whole. Pele draws an entity's
// parts in a frame about its center, and limu in the diagram's: the reader moves them into
// the diagram's frame, adding the center as limu does, and then writes every number as
// Pele's num does.
export function readSvg(svg, num) {
  const n = (s) => num(Number(s));
  // A number of an entity's frame in the diagram's, the entity's center being c.
  const from = (c, s) => num(Number(c) + Number(s));
  const box = (s, cx = '0', cy = '0') =>
    `${from(cx, attr(s, 'x') ?? 0)},${from(cy, attr(s, 'y') ?? 0)},${n(attr(s, 'width'))},${n(attr(s, 'height'))}`;
  // Where a label's first line sits: the first x and y in its text.
  const textAt = (s, cx = '0', cy = '0') => {
    const i = s.indexOf('<text');
    if (i < 0) return undefined;
    const m = / x="([^"]*)" y="([^"]*)"/.exec(s.slice(i));
    return m ? `t ${from(cx, m[1])},${from(cy, m[2])}` : undefined;
  };
  const path = (d, cx = '0', cy = '0') => lines(d).replace(/([^MLCQZ ,]+),([^MLCQZ ,]+)/g, (_, x, y) => `${from(cx, x)},${from(cy, y)}`);

  const out = [];
  const head = tag(svg, 'svg');
  out.push(`size ${n(attr(head, 'width'))},${n(attr(head, 'height'))}`);
  const title = svg.indexOf('<text class="pele-title"');
  if (title >= 0) out.push('title ' + textAt(svg.slice(title)));
  const starts = [...svg.matchAll(GROUP)];
  for (let k = 0; k < starts.length; k++) {
    const kind = starts[k][1];
    const s = svg.slice(starts[k].index, k + 1 < starts.length ? starts[k + 1].index : svg.length);
    const open = s.slice(0, s.indexOf('>'));
    const id = attr(open, 'data-id');
    const classes = attr(open, 'class').split(' ');
    switch (kind) {
      case 'cluster': {
        out.push(`cluster ${id} [${classes.slice(1).join(' ')}] ${box(tag(s, 'rect'))}`);
        const t = textAt(s);
        if (t) out.push(t);
        break;
      }
      case 'edge': {
        const line = tag(s, 'path');
        const dashed = line.includes(' stroke-dasharray="4 3"') ? ' dashed' : '';
        out.push(`edge ${id} ${classes[1].slice('pele-er-'.length)}${dashed} ${path(attr(line, 'd'))}`);
        for (const m of s.matchAll(/<(path|circle) class="pele-marker"[^>]*>/g)) {
          if (m[1] === 'circle') out.push(`m o ${n(attr(m[0], 'cx'))},${n(attr(m[0], 'cy'))} ${n(attr(m[0], 'r'))}`);
          else out.push(`m ${path(attr(m[0], 'd'))}${attr(m[0], 'fill') === 'var(--_bg)' ? ' bg' : ''}`);
        }
        break;
      }
      case 'edge-label':
        out.push(`label ${id} ${box(tag(s, 'rect'))}`, textAt(s));
        break;
      case 'node': {
        const [, cx, cy] = /translate\(([^,]+),([^)]+)\)/.exec(attr(open, 'transform'));
        out.push(`node ${id} [${classes.slice(2).join(' ')}]`);
        const rects = tags(s, 'rect');
        out.push(`box ${box(rects[0], cx, cy)}`);
        const stripes = s.indexOf('<g class="pele-er-stripes"');
        if (stripes >= 0) {
          const group = s.slice(stripes, s.indexOf('</g>', stripes));
          for (const r of tags(group, 'rect')) out.push(`stripe ${box(r, cx, cy)}${/rx/.test(r) ? ' rounded' : ''}`);
          out.push(`divider ${path(attr(tag(s, 'path class="pele-er-divider"'), 'd'), cx, cy)}`);
          out.push(`outline ${box(rects.at(-1), cx, cy)}`);
        }
        const a = /<a [^>]*\shref="([^"]*)"/.exec(s);
        if (a) out.push(`link ${unescape(a[1])}`);
        const name = s.indexOf('<text class="pele-label"');
        if (name >= 0) out.push(textAt(s.slice(name), cx, cy));
        const cells = s.indexOf('<g class="pele-er-attributes"');
        if (cells >= 0) {
          const size = attr(tag(s, 'g class="pele-er-attributes"'), 'font-size');
          // A cell's text puts its position on its first span with text, if it has one.
          for (const m of s.slice(cells).matchAll(/<text class="pele-er-(type|name|keys|comment)"[^>]*>/g)) {
            const at = cells + m.index;
            const t = textAt(s.slice(at, s.indexOf('</text>', at)), cx, cy) ?? 't none';
            out.push(`${m[1]} ${n(attr(m[0], 'font-size') ?? size)} ${t}`);
          }
        }
        break;
      }
    }
  }
  return out;
}

export default async function ({ src, cases, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { preprocess } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');
  const { num } = await src('svg/builder.ts');

  const corpus = [...new Set([...WRITTEN, ...PARSE_WRITTEN, ...cases('er').extracted])].filter((s) => {
    try {
      return detect(preprocess(s).text) === 'er';
    } catch {
      return false;
    }
  });

  const out = [];
  const draw = (source, maxWidth) => {
    let r;
    globalThis.limuExactNumbers = true;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, outputLimit: Infinity });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'er') return false;
    out.push({ src: source, maxWidth, lines: readSvg(r.svg, num) });
    return true;
  };
  let drawn = 0;
  const each = (source) => {
    // Every third one again in a narrow host, which turns what runs across and does not fit,
    // and every one that may run across.
    if (draw(source) && (drawn++ % 3 === 0 || /direction\s+(LR|RL)/i.test(source))) draw(source, 360);
  };
  corpus.forEach(each);
  const next = mutator(corpus, FRAGMENTS, random(13));
  while (drawn < 2500) {
    const s = next();
    if (!lone(s) && s.length <= 3000) each(s);
  }
  write('diagram/er/testdata/pele/render.json', { cases: out });
}
