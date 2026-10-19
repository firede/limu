// Requirement diagrams drawn end to end: what Pele's render draws, read back out of its SVG as
// one line per thing drawn, in the order drawn, with numbers as the SVG writes them. limu's
// test writes the same lines from its scene.
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { unescape, attr, capJoin } from '../lib/svg.mjs';
import { FRAGMENTS, WRITTEN, corpus, generated } from './requirement.mjs';

const box = (s) => `${attr(s, 'x')},${attr(s, 'y')},${attr(s, 'width')},${attr(s, 'height')}`;

// A text element as one line: its class, its font's size, weight and slant and its anchor,
// then each run of it: where it is placed, if it is, its own emphasis and its text.
function textLine(open, spans) {
  const head = `text ${attr(open, 'class') ?? '-'} fs=${attr(open, 'font-size') ?? '-'} w=${attr(open, 'font-weight') ?? '-'} i=${
    attr(open, 'font-style') === 'italic' ? 1 : 0
  } a=${attr(open, 'text-anchor')}`;
  return head + spans.map((s) => ' ' + s).join('');
}

const span = (open, text) => {
  const x = attr(open, 'x');
  const pos = x === undefined ? '' : `@${x},${attr(open, 'y')}`;
  const b = attr(open, 'font-weight') === undefined ? '' : '+b';
  const i = attr(open, 'font-style') === undefined ? '' : '+i';
  return pos + b + i + ':' + JSON.stringify(unescape(text));
};

const TAG = /<(\/?)([\w:-]+)((?:[^>"]|"[^"]*")*?)(\/?)>|([^<]+)/g;

// How an element's stroke ends and turns, as "cap/join", as lib/svg.mjs's capJoin settles
// them from it and the elements it is in, given innermost first.
const pen = (tag, outer) => capJoin([...outer].reverse().concat([tag]), '/');

// The lines of a requirement diagram's SVG. A rect, path or circle ends with how it ends and
// turns its lines.
export function readSvg(svg) {
  const lines = [];
  // The open groups, innermost first.
  const groups = [];
  let text = null; // the open text element and its runs
  let tspan = null;
  let acc = null;
  for (const m of svg.matchAll(TAG)) {
    const [, close, name, attrs, , content] = m;
    if (content !== undefined) {
      if (acc) acc.text += content;
      else if (tspan) tspan.text += content;
      else if (text) text.plain += content;
      continue;
    }
    const open = ' ' + attrs;
    if (close) {
      if (name === 'tspan') {
        text.spans.push(span(tspan.open, tspan.text));
        tspan = null;
      } else if (name === 'text') {
        const spans = text.spans.length > 0 ? text.spans : [span(text.open, text.plain)];
        lines.push(textLine(text.open, spans));
        text = null;
      } else if (name === 'g') {
        groups.shift();
      } else if (name === 'title' || name === 'desc') {
        lines.push(`acc-${name} ${unescape(acc.text)}`);
        acc = null;
      }
      continue;
    }
    switch (name) {
      case 'svg':
        lines.push(`size ${attr(open, 'width')},${attr(open, 'height')}`);
        break;
      case 'title':
      case 'desc':
        acc = { text: '' };
        break;
      case 'text':
        text = { open, spans: [], plain: '' };
        break;
      case 'tspan':
        tspan = { open, text: '' };
        break;
      case 'g': {
        groups.unshift(open);
        const cls = attr(open, 'class');
        if (cls === 'pele-edge-labels') lines.push(`labels ${attr(open, 'font-size')}`);
        else if (cls === 'pele-fields') lines.push(`fields ${attr(open, 'font-size')}`);
        else if (cls === 'pele-edge-label') lines.push(`label ${attr(open, 'data-id')}`);
        else if (cls.startsWith('pele-edge ')) lines.push(`edge ${attr(open, 'data-id')} ${cls.slice('pele-edge pele-relationship-'.length)}`);
        else if (cls.startsWith('pele-node ')) {
          const [, kind, ...classes] = cls.split(' ');
          lines.push(`node ${attr(open, 'data-id')} ${kind.slice('pele-'.length)} [${classes.join(' ')}] ${attr(open, 'transform')}`);
        }
        break;
      }
      case 'rect':
        lines.push(`rect ${box(open)} ${pen(open, groups)}`);
        break;
      case 'circle':
        lines.push(`m o ${attr(open, 'cx')},${attr(open, 'cy')},${attr(open, 'r')} ${pen(open, groups)}`);
        break;
      case 'path': {
        const cls = attr(open, 'class');
        const p = pen(open, groups);
        if (cls === 'pele-marker') lines.push(`m ${attr(open, 'd')} ${p}`);
        else if (cls === 'pele-divider') lines.push(`divider ${attr(open, 'd')} ${p}`);
        else lines.push(`path ${attr(open, 'd')}${open.includes(' stroke-dasharray="5 4"') ? ' dashed' : ''} ${p}`);
        break;
      }
      default:
        lines.push(`unknown ${name}`);
    }
  }
  return lines;
}

// The fragments mutations splice in, for text that parses and still draws something odd.
const MORE = [
  '\nrequirement x {\nid: 1\ntext: "a long text that wraps past the width a value has in its box"\n}\n',
  '\nelement y {\ntype: t\ndocref: d\n}\n', '\nx - contains -> y\n', '\ny - traces -> x\n', '\nx - satisfies -> x\n',
  '\nclassDef c font-weight:bold\nclass x,y c\n', '\nstyle x font-size:22px\n', '\nstyle y font-style:italic\n',
  '\ndirection LR\n', '\ndirection BT\n', '\ndirection RL\n', '<br>', '**b**', '_i_',
  '\nstyle x stroke:#333,stroke-linejoin:bevel,rx:0\n', '\nclassDef c stroke:#a00,stroke-linecap:square,stroke-linejoin:round\n',
];

// Ends and corners an author sets, and some a browser does not take.
const STROKES = [
  'requirementDiagram\nrequirement r {\nid: 1\ntext: t\n}\nelement e {\ntype: x\n}\nelement f {\n}\ne - satisfies -> r\nf - contains -> r\n' +
    'style r stroke:#333,stroke-linejoin:bevel,rx:0\nclassDef k stroke:#a00,stroke-linecap:square,stroke-linejoin:ROUND\nclass e k\n' +
    'style f stroke:#333,stroke-linejoin:inherit,stroke-linecap:flat',
];

export default async function ({ src, cases, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { preprocess } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');

  const isRequirement = (s) => {
    try {
      return detect(preprocess(s).text) === 'requirement';
    } catch {
      return false;
    }
  };
  const seeds = [...WRITTEN, ...STROKES, ...corpus(cases)].filter(isRequirement);

  const out = [];
  const draw = (source, maxWidth) => {
    let r;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, outputLimit: Infinity });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return false;
    }
    if (r.type !== 'requirement') return false;
    out.push({ src: source, maxWidth, lines: readSvg(r.svg) });
    return true;
  };
  let drawn = 0;
  const seen = new Set();
  const each = (source) => {
    if (seen.has(source)) return;
    seen.add(source);
    // Every third one again in a narrow host, and every one that runs across, which turns
    // what runs across and does not fit.
    if (draw(source) && (drawn++ % 3 === 0 || /direction\s+(LR|RL)/i.test(source))) draw(source, 360);
  };
  seeds.forEach(each);
  const rnd = random(17);
  for (let k = 0; k < 1500; k++) each(generated(rnd));
  const next = mutator(seeds, [...FRAGMENTS, ...MORE], random(13));
  while (drawn < 4000) {
    const s = next();
    if (s.length <= 3000 && !lone(s)) each(s);
  }
  write('diagram/requirement/testdata/pele/render.json', { cases: out });
}
