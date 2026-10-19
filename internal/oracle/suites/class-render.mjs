// Class diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// A scene has no groups to move what is in them, so it holds where everything is on the page,
// where Pele writes the inside of a class or a note about its middle, and everything about the
// title. Pele's SVG is drawn here with its numbers written out in full, as the oracle's hooks
// have num() write them while limuExactNumbers is set; each number is moved by the groups it is
// in, one move after the other from the innermost out, as the scene's are, and then written to
// two decimals as Pele writes them. The markers are drawn again where the title moves them,
// from their tips and directions, which classMarker writes before them while the flag is set.
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { unescape, attr, capJoin } from '../lib/svg.mjs';
import { readMarker } from '../lib/limu.mjs';
import { WRITTEN } from './class.mjs';

// The fragments mutations splice in, for text that parses and still draws something odd.
const FRAGMENTS = [
  '\n', ' <|-- ', ' *-- ', ' o-- ', ' --> ', ' -- ', ' ..> ', ' ..|> ', ' .. ', ' --* ', ' --o ', ' <--> ', ' ()-- ',
  ' --() ', ' "1" ', ' "0..*" ', ' "many" ', ' : label ', ' : a longer label that wraps past its width ', 'A', 'B', 'C',
  'Duck', '\nclass X {\n+int a\n-run() void\n}\n', '\nclass Y~T~\n', '\n<<interface>> A\n', '\nnote "a note"\n',
  '\nnote for A "for A"\n', '\nnamespace N {\nclass P\nclass Q\n}\n', '\nnamespace M.O {\nclass R\n}\n', '\nA --> A\n',
  '\nA "1" --> "2" A : self\n', '\ndirection LR\n', '\ndirection BT\n', '\ndirection RL\n', '\nclassDef k fill:#f96\n',
  '\nclass A:::k\n', '\nstyle A font-size:20px\n', '\nstyle A stroke:#333,stroke-linejoin:bevel,rx:0\n', '\nstyle B stroke:#333,stroke-linejoin:inherit\n',
  '\nclassDef k stroke:#a00,stroke-linecap:square,stroke-linejoin:ROUND\n', '\nstyle B font-style:italic\n', '\ncssClass "A,B" k\n',
  '\nlink A "https://x.y/z" "tip"\n', '\nclick B call f() "t"\n', '\nA : +x$\n', '\nA : +y()*\n', '\nA : <<enum>>\n',
  '中文', '😀', '#quot;', '&amp;', '<br>', '~T~', '$', '*', '{', '}', '"',
];

// Text that is drawn in more than one line or run: markup, breaks, icons with nothing to draw
// them, and the escapes Pele undoes before it measures.
const DRAWN = [
  'classDiagram\nclass A["fa:fa-car Car"]\nA --> B : go **b** *i*\nnote "fa:fa-y note **b** *i*"\nnamespace N["fa:fa-z n"] {\nclass C\n}',
  'classDiagram\nA --> B : one<br>two<br/>three\nnote for A "a<br>b<br>c"\nclass B["Two<br>lines"]',
  'classDiagram\nclass A {\n+&lt;b&gt; x\n+&amp;lt;y&amp;gt;\n+run(&quot;q&quot;) &#35;\n}\n<<&lt;i&gt;>> A\nA "&lt;1&gt;" --> "&amp;" B',
  'classDiagram\nA --> B : a label long enough that Pele wraps it at its two hundred units of width, more than once\nnote "a note long enough that Pele wraps it at its width of two hundred and sixty units, which this note passes by some way, twice"',
  'classDiagram\nclass `A b`\nclass C["  spaced  "]\n`A b` <|-- C : `back` ticks\nnote for C "  spaced  "',
  // Ends and corners an author sets, and some a browser does not take.
  'classDiagram\nclass A\nclass B\nclass C\nclass D\nclassDef k stroke:#a00,stroke-linecap:square,stroke-linejoin:ROUND\nclass B:::k\nstyle A stroke:#333,stroke-linejoin:bevel,rx:0,stroke-width:4px\nstyle C stroke:#333,stroke-linejoin:inherit,stroke-linecap:flat\nstyle D stroke:#333,stroke-linejoin:round,stroke-linejoin:arcs\nA *-- B\nC <|-- D\nnote for A "n"',
];

// A diagram of a few to a dozen classes, some in namespaces, some with members, notes and
// labels, related at random.
const NAMES = ['Animal', 'Duck', 'Fish', 'Zebra', 'Order', 'Customer', 'LineItem', 'Product', 'Invoice', 'Payment', 'Ledger', 'Account'];
const ARROWS = ['<|--', '--|>', '*--', '--*', 'o--', '--o', '-->', '<--', '--', '..>', '<..', '..|>', '<|..', '..', '()--', '--()', '<-->'];
const CARDS = ['', '', '"1"', '"0..*"', '"many"', '"1..n"'];
function wide(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const n = 3 + Math.floor(rnd() * 10);
  const names = NAMES.slice(0, n);
  const lines = ['classDiagram'];
  if (rnd() < 0.6) lines.push('direction ' + pick(['TB', 'LR', 'BT', 'RL']));
  const spaces = Math.floor(rnd() * 4);
  const placed = new Set();
  for (let k = 0; k < spaces; k++) {
    const inner = names.filter((x) => !placed.has(x) && rnd() < 0.35);
    if (inner.length === 0) continue;
    inner.forEach((x) => placed.add(x));
    const id = (k > 0 && rnd() < 0.4 ? 'N0.' : '') + 'N' + k;
    lines.push(`namespace ${id}${rnd() < 0.3 ? '["Space ' + k + '"]' : ''} {`, ...inner.map((x) => `class ${x}`), '}');
  }
  for (const x of names) {
    if (rnd() < 0.3) lines.push(`class ${x} {`, `+int ${x.toLowerCase()}Count`, `+${x.toLowerCase()}Run(int a)$ bool`, '}');
  }
  const edges = Math.floor(rnd() * (n + 3));
  for (let k = 0; k < edges; k++) {
    const a = pick(names);
    const b = rnd() < 0.08 ? a : pick(names);
    const label = rnd() < 0.4 ? ' : ' + pick(['has', 'owns many', 'belongs to', 'pays']) : '';
    lines.push(`${a} ${pick(CARDS)} ${pick(ARROWS)} ${pick(CARDS)} ${b}${label}`);
  }
  if (rnd() < 0.3) lines.push(`note for ${pick(names)} "a note"`);
  if (rnd() < 0.2) lines.push('note "free"');
  return lines.join('\n');
}

// The ways a diagram is laid out besides its own: a title, and Pele's options for classes.
const HEADS = [
  '---\ntitle: Class diagram\n---\n',
  '---\ntitle: A title wider than most of the diagrams it sits above, by some way\n---\n',
  '---\nconfig:\n  class:\n    hideEmptyMembersBox: true\n---\n',
  '---\nconfig:\n  class:\n    hierarchicalNamespaces: false\n---\n',
  '---\nconfig:\n  class:\n    nodeSpacing: 90\n    rankSpacing: 20\n---\n',
];

const TAG = /<(\/?)([A-Za-z][\w-]*)([^>]*?)(\/?)>/g;

// How an element's stroke ends and turns, as "cap/join", as lib/svg.mjs's capJoin settles
// them from it and the elements it is in, given innermost first.
const pen = (tag, outer) => capJoin([...outer].reverse().concat([tag]), '/');
const NUMBER = /[MLHVCQZ]|-?(?:\d+\.?\d*|\.\d+)(?:e[-+]?\d+)?/gi;

// The lines of a class diagram's SVG, drawn with numbers in full; num writes them as Pele does,
// and redraw draws a marker again from what the hooks write before it, at a tip moved by dx,
// dy.
export function readSvg(svg, num, redraw) {
  const lines = [];
  // The open elements, each with the group it makes, if any, and where that group moves what
  // is in it.
  const open = [];
  const kind = () => {
    for (let k = open.length - 1; k >= 0; k--) if (open[k].kind) return open[k];
    return undefined;
  };
  const move = (v, axis) => {
    for (let k = open.length - 1; k >= 0; k--) if (open[k].by) v = v + open[k].by[axis];
    return v;
  };
  const pt = (x, y) => `${num(move(Number(x), 0))},${num(move(Number(y), 1))}`;
  const box = (tag) => `${pt(attr(tag, 'x') ?? 0, attr(tag, 'y') ?? 0)},${num(Number(attr(tag, 'width')))},${num(Number(attr(tag, 'height')))}`;
  // A path's d, each point moved, H and V written as L.
  const d = (raw) => {
    let out = '';
    let x = 0;
    let y = 0;
    let op = '';
    let args = [];
    const flush = () => {
      if (op === '') return;
      if (op === 'Z') {
        out += 'Z';
      } else if (op === 'H' || op === 'V') {
        for (const a of args) {
          if (op === 'H') x = a;
          else y = a;
          out += `L${pt(x, y)}`;
        }
      } else {
        const per = op === 'C' ? 6 : op === 'Q' ? 4 : 2;
        for (let i = 0; i < args.length; i += per) {
          const pts = [];
          for (let j = i; j < i + per; j += 2) pts.push(pt(args[j], args[j + 1]));
          out += (i === 0 || op !== 'M' ? op : 'L') + pts.join(' ');
          x = args[i + per - 2];
          y = args[i + per - 1];
        }
      }
      args = [];
    };
    for (const m of raw.matchAll(NUMBER)) {
      if (/[A-Z]/i.test(m[0])) {
        flush();
        op = m[0].toUpperCase();
      } else {
        args.push(Number(m[0]));
      }
    }
    flush();
    return out;
  };
  // The text of an element, its tags taken out and its escapes undone.
  const content = (s) => unescape(s.replace(/<[^>]*>/g, ''));
  // How an element in the open ones strokes its ends and corners.
  const penOf = (tag) => pen(tag, open.map((e) => e.tag ?? '').reverse());
  // Where a text's lines, and the runs in them, are drawn: each x and y in it.
  const at = (s) => [...s.matchAll(/ x="([^"]*)" y="([^"]*)"/g)].map((m) => pt(m[1], m[2])).join(' ');
  // The marker the hooks wrote last, which the next element drawn as a marker is.
  let written;
  const marker = () => {
    const { type, x, y, dx, dy } = written;
    const again = redraw(type, move(x, 0), move(y, 1), dx, dy);
    const t = /<(path|circle)([^>]*)>/.exec(again);
    const fill = attr(t[2], 'fill');
    const paint = fill === undefined ? 'stroke' : attr(t[2], 'stroke') === 'none' ? 'fill' : 'fill+stroke';
    if (t[1] === 'circle') return `m ${paint} o ${num(Number(attr(t[2], 'cx')))},${num(Number(attr(t[2], 'cy')))}`;
    const saved = open.slice();
    open.length = 0;
    const line = `m ${paint} ${d(attr(t[2], 'd'))}`;
    open.push(...saved);
    return line;
  };

  TAG.lastIndex = 0;
  for (let m = TAG.exec(svg); m; m = TAG.exec(svg)) {
    const [whole, closing, name, rest] = m;
    const selfClosing = m[4] === '/';
    if (closing) {
      while (open.length > 0 && open.pop().name !== name);
      continue;
    }
    if (name === 'limu-marker') {
      written = readMarker(whole);
      continue;
    }
    const cls = attr(rest, 'class') ?? '';
    const k = kind();
    if (name === 'text' || name === 'title') {
      const endAt = svg.indexOf(`</${name}>`, TAG.lastIndex);
      const inner = svg.slice(m.index, endAt);
      TAG.lastIndex = endAt + name.length + 3;
      if (name === 'title') {
        if (k?.kind === 'class') lines.push(`tip ${content(inner.slice(whole.length))}`);
        continue;
      }
      if (cls === 'pele-title') {
        lines.push(`title ${at(inner)}`);
      } else if (cls === 'pele-cardinality') {
        lines.push(`card ${content(inner.slice(whole.length))} ${at(inner)}`);
      } else if (cls === 'pele-annotation') {
        lines.push(`annotation ${content(inner.slice(whole.length))} ${at(inner)}`);
      } else if (cls.startsWith('pele-member')) {
        const parts = cls.split(' ');
        const flag = parts[2] === 'pele-static' ? ' static' : parts[2] === 'pele-abstract' ? ' abstract' : '';
        lines.push(`member ${parts[1].slice(5)}${flag} ${at(inner)} ${content(inner.slice(whole.length))}`);
      } else if (k && !(k.kind === 'struck' && k.seen)) {
        lines.push(`t ${at(inner)}`);
        k.seen = true;
      }
      continue;
    }
    const element = { name, tag: rest };
    if (name === 'svg' && cls === 'pele-icon') {
      lines.push(`icon ${attr(rest, 'data-icon')} ${box(rest)}`);
    } else if (name === 'svg') {
      lines.push(`size ${attr(rest, 'width')},${attr(rest, 'height')}`);
    } else if (name === 'g') {
      const id = attr(rest, 'data-id');
      const transform = attr(rest, 'transform');
      if (transform) {
        const t = /translate\(([^,]*),([^)]*)\)/.exec(transform);
        element.by = [Number(t[1]), Number(t[2])];
      }
      const classes = cls.split(' ');
      if (cls === 'pele-cluster pele-namespace') {
        element.kind = 'ns';
      } else if (classes[0] === 'pele-edge' && id !== undefined) {
        element.kind = 'edge';
      } else if (cls === 'pele-cluster-title') {
        element.kind = 'struck';
        lines.push(`struck ${id}`);
      } else if (cls === 'pele-edge-label') {
        element.kind = 'label';
      } else if (classes[0] === 'pele-node') {
        element.kind = classes[1].slice(5);
        if (element.kind === 'class') lines.push(`class ${id} [${classes.slice(2).join(' ')}]`);
        else lines.push(`${element.kind} ${id}`);
      }
      element.id = id;
      element.classes = classes;
    } else if (name === 'a' && k?.kind === 'class') {
      lines.push(`link ${attr(rest, 'href')}`);
    } else if (name === 'rect' && k) {
      if (k.kind === 'ns') lines.push(`ns ${k.id} ${box(rest)}`);
      else if (k.kind === 'label') lines.push(`label ${k.id} ${box(rest)}`);
      else if (k.kind === 'class') lines.push(`rect ${box(rest)} ${penOf(rest)}`);
    } else if ((name === 'path' || name === 'circle') && k) {
      if (cls === 'pele-marker') {
        lines.push(`${marker()} ${penOf(rest)}`);
      } else if (k.kind === 'edge') {
        const dash = attr(rest, 'stroke-dasharray');
        const flag = dash === '6 4' ? ' dashed' : dash === '2 4' ? ' dotted' : '';
        lines.push(`edge ${k.id} ${k.classes[1] === 'pele-note-edge' ? 'note' : 'relation'}${flag} ${d(attr(rest, 'd'))} ${penOf(rest)}`);
      } else if (k.kind === 'class') {
        lines.push(`divider ${d(attr(rest, 'd'))}`);
      } else if (k.kind === 'note') {
        lines.push(`${k.paths ? 'fold' : 'path'} ${d(attr(rest, 'd'))} ${penOf(rest)}`);
        k.paths = true;
      }
    }
    if (!selfClosing && name !== 'rect' && name !== 'path' && name !== 'circle') open.push(element);
  }
  return lines;
}

export default async function ({ src, cases, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { preprocess } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');
  const { classMarker } = await src('diagrams/class/markers.ts');
  const { num } = await src('svg/builder.ts');

  const classes = [...DRAWN, ...WRITTEN, ...cases('class').extracted].filter((s) => {
    try {
      return detect(preprocess(s).text) === 'class';
    } catch {
      return false;
    }
  });

  const redraw = (type, x, y, dx, dy) => classMarker(type, x, y, dx, dy, 'var(--_l)');
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
    if (r.type !== 'class') return false;
    out.push({ src: source, maxWidth, lines: readSvg(r.svg, num, redraw) });
    return true;
  };
  let drawn = 0;
  const each = (source) => {
    if (!draw(source)) return;
    // Every third one again in a narrow host, which turns or tightens what does not fit, and
    // every seventh with a title or options of its own.
    if (drawn % 3 === 0) draw(source, 360);
    if (drawn % 7 === 0 && !source.startsWith('---') && !source.startsWith('%%')) draw(HEADS[(drawn / 7) % HEADS.length] + source);
    drawn++;
  };
  classes.forEach(each);
  const next = mutator(classes, FRAGMENTS, random(13));
  while (drawn < 2200) {
    const s = next();
    if (!lone(s) && s.length <= 3000) each(s);
  }
  // Diagrams too wide for a narrow host, each drawn as it is and in two narrow ones, which
  // tighten it or turn it down.
  const rnd = random(17);
  for (let i = 0; i < 400; i++) {
    const s = wide(rnd);
    if (draw(s)) {
      draw(s, 360);
      draw(s, i % 2 === 0 ? 520 : 200);
    }
  }
  // The test reads a line with a tag among its classes with the tag moved last, so the lines
  // that name one are kept in full.
  write('diagram/class/testdata/pele/render.json', { cases: out }, { keep: (line) => line.includes('clickable') });
}
