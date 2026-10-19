// State diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per element drawn, in the order drawn, with numbers as the SVG writes them. limu's test
// writes the same lines from its scene.
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { unescape, attr, capJoin } from '../lib/svg.mjs';
import { WRITTEN, stateCorpus } from './state.mjs';

// The fragments mutations splice in, for text that parses and still draws something odd.
const FRAGMENTS = [
  '\n', ' --> ', ' : label ', '\n[*] --> ', ' --> [*]\n', 'a', 'b', 'c', 'S1', 'long state name that wraps',
  '\nstate X {\n', '\nstate "Described" as Y {\n', '\n}\n', '\n--\n', '\ndirection LR\n', '\ndirection TB\n',
  '\ndirection BT\n', '\ndirection RL\n', '\nstate f <<fork>>\n', '\nstate j <<join>>\n', '\nstate c <<choice>>\n',
  '\nnote right of a : note\n', '\nnote left of b\n  two\n  lines\nend note\n', '\na : description\n',
  '\na --> a : loop\n', '\nb --> b\n', ':::cls', '\nclassDef cls fill:#f96,font-weight:bold\n',
  '\nstyle a stroke:#f00,font-size:22px\n', '\nclick a "https://x.y/z" "tip"\n', '中文', '😀', '#quot;', '<br>',
  '**bold**', '\nhide empty description\n',
];

const TAG = /<(\/?)(g|a|rect|circle|polygon|path|text|title)\b([^>]*)>/g;

// The lines of a state diagram's SVG. Everything the title moves down is moved here too, so
// that the lines give where things are on the page; what a node's transform places is given
// as written, from the node's middle.
export function readSvg(svg, num) {
  const lines = [];
  const head = /<svg [^>]*>/.exec(svg)[0];
  lines.push(`size ${attr(head, 'width')},${attr(head, 'height')}`);
  let tx = 0;
  let ty = 0;
  const sx = (v) => (tx === 0 ? v : num(parseFloat(v) + tx));
  const sy = (v) => (ty === 0 ? v : num(parseFloat(v) + ty));
  // A path written with absolute commands only, H and V as L.
  const path = (d, moved) => {
    const X = moved ? (v) => v : sx;
    const Y = moved ? (v) => v : sy;
    let out = '';
    let cx = '';
    let cy = '';
    let mx = '';
    let my = '';
    for (const m of d.matchAll(/([MLHVCQZ])([^MLHVCQZ]*)/g)) {
      const nums = m[2].split(/[ ,]+/).filter((s) => s !== '');
      switch (m[1]) {
        case 'H':
          cx = X(nums[0]);
          out += `L${cx},${cy}`;
          break;
        case 'V':
          cy = Y(nums[0]);
          out += `L${cx},${cy}`;
          break;
        case 'Z':
          out += 'Z';
          cx = mx;
          cy = my;
          break;
        default: {
          const pairs = [];
          for (let k = 0; k < nums.length; k += 2) pairs.push(`${X(nums[k])},${Y(nums[k + 1])}`);
          out += m[1] + pairs.join(' ');
          [cx, cy] = pairs[pairs.length - 1].split(',');
          if (m[1] === 'M') {
            mx = cx;
            my = cy;
          }
        }
      }
    }
    return out;
  };
  const stack = [];
  // The opening tags of the groups open, for the caps and joins they give what is in them.
  const groups = [];
  let inNode = false;
  for (const m of svg.matchAll(TAG)) {
    const [whole, close, name, rest] = m;
    if (name === 'g') {
      if (close) {
        inNode = stack.pop() ?? false;
        groups.pop();
        continue;
      }
      stack.push(inNode);
      groups.push(whole);
      const cls = attr(rest, 'class') ?? '';
      const id = attr(rest, 'data-id');
      const t = attr(rest, 'transform');
      const names = cls.split(' ');
      if (cls === '' && t) {
        const [, x, y] = /translate\(([^,]*),([^)]*)\)/.exec(t);
        tx = parseFloat(x);
        ty = parseFloat(y);
      } else if (names[0] === 'pele-cluster') {
        lines.push(`composite ${id} [${names.slice(2).join(' ')}]`);
      } else if (names[0] === 'pele-node' || names[0] === 'pele-note') {
        const [, x, y] = /translate\(([^,]*),([^)]*)\)/.exec(t);
        const kind = names[0] === 'pele-note' ? 'note' : names[1].slice(5);
        const extra = names.slice(names[0] === 'pele-note' ? 1 : 2);
        lines.push(`node ${id} ${kind} [${extra.join(' ')}] ${sx(x)},${sy(y)}`);
        inNode = true;
      } else if (names[0] === 'pele-edge') {
        lines.push(`edge ${id} ${names[1].slice(5)}`);
      } else if (names[0] === 'pele-edge-label') {
        lines.push(`label ${id}`);
      }
      continue;
    }
    if (close) continue;
    const X = inNode ? (v) => v : sx;
    const Y = inNode ? (v) => v : sy;
    switch (name) {
      case 'a':
        lines.push(`link ${attr(rest, 'href')}`);
        break;
      case 'title': {
        // The document's own title is its accessible one.
        if (stack.length === 0) break;
        const start = m.index + whole.length;
        lines.push(`tip ${unescape(svg.slice(start, svg.indexOf('</title>', start)))}`);
        break;
      }
      case 'rect': {
        const kind = attr(rest, 'class') === 'pele-divider' ? 'divider-rect' : 'rect';
        lines.push(`${kind} ${X(attr(rest, 'x'))},${Y(attr(rest, 'y'))},${attr(rest, 'width')},${attr(rest, 'height')} ${attr(rest, 'rx')}`);
        break;
      }
      case 'circle':
        lines.push(`circle ${attr(rest, 'r')}`);
        break;
      case 'polygon':
        lines.push(`polygon ${attr(rest, 'points')}`);
        break;
      case 'path': {
        const cls = attr(rest, 'class');
        const kind = cls === 'pele-marker' ? 'm' : cls === 'pele-divider' ? 'divider' : 'path';
        lines.push(`${kind} ${path(attr(rest, 'd'), inNode)}`);
        // An edge's line, with the cap and join it is stroked with.
        if (kind === 'path' && (attr(groups[groups.length - 1] ?? '', 'class') ?? '').startsWith('pele-edge ')) {
          lines.push(`stroke ${capJoin([...groups, whole])}`);
        }
        break;
      }
      case 'text': {
        const p = / x="([^"]*)" y="([^"]*)"/.exec(svg.slice(m.index));
        if (attr(rest, 'class') === 'pele-title') lines.push(`title t ${p[1]},${p[2]}`);
        else lines.push(`t ${X(p[1])},${Y(p[2])}`);
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
  const { num } = await src('svg/builder.ts');

  const seeds = [...WRITTEN.filter((s) => s.startsWith('stateDiagram') || s.startsWith('---') || s.startsWith('%%')), ...stateCorpus(cases, detect, preprocess)];

  const out = [];
  const draw = (source, maxWidth) => {
    let r;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, outputLimit: Infinity });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return false;
    }
    if (r.type !== 'state') return false;
    out.push({ src: source, maxWidth, lines: readSvg(r.svg, num) });
    return true;
  };
  let drawn = 0;
  const each = (source) => {
    // Every third one again in a narrow host, which turns what runs across and does not fit,
    // and every one that runs across.
    if (draw(source) && (drawn++ % 3 === 0 || /direction\s+(LR|RL)/i.test(source))) draw(source, 360);
  };
  seeds.forEach(each);
  // The same diagrams run across, which a narrow host turns to run down.
  for (const s of seeds) {
    const i = s.indexOf('stateDiagram');
    const eol = s.indexOf('\n', i);
    if (i >= 0 && eol >= 0) each(s.slice(0, eol + 1) + 'direction LR\n' + s.slice(eol + 1));
  }
  const next = mutator(seeds, FRAGMENTS, random(11));
  while (drawn < 2500) {
    const s = next();
    if (s.length <= 3000 && !lone(s)) each(s);
  }
  write('diagram/state/testdata/pele/render.json', { cases: out });
}
