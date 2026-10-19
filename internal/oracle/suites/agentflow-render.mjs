// Agent flows drawn end to end: what Pele's render draws, read back out of its SVG as one line
// per thing drawn, in the order drawn, with every number of its geometry as the SVG writes it.
// limu's test writes the same lines from its scene.
import { mutator } from '../lib/mutate.mjs';
import { unescape, attr, tag, capJoin } from '../lib/svg.mjs';
import { FRAGMENTS, agentflowCorpus } from './agentflow.mjs';

const H = 'agentflow-beta\n';

// Cases written for what the corpus has little of: collapsed flows, edges between a node and
// a flow around it, self-loops, nested flows that run another way, links, styles, options.
const WRITTEN = [
  H + 'flow f["F"]\n  a --> b\nend\nf@{ view: collapsed }\nc --> f\n',
  'agentflow-beta LR\n  flow f["F"]@{ model: "m" }\n    a e1@--> b & c\n    e1@{ instruction: "go" }\n  end\n  f@{ view: collapsed }\n',
  H + 'flow f["A long title for a collapsed flow that wraps"]\n  a\nend\nf@{ view: collapsed }\n',
  H + 'flow f\n  a --> b\nend\na --> f\nf --> b\n',
  H + 'flow f\n  a\n  b\nend\nf --> a\nb --> f\n',
  'agentflow-beta LR\nflow f\n  a --> b\nend\na --> f\nf --> b\nb -- "back out" --> f\n',
  H + 'flow outer\n  flow inner\n    direction LR\n    a --> b\n  end\n  c\nend\na --> outer\ninner --> outer\nouter --> c\n',
  H + 'flow f\n  direction RL\n  a --> b\n  c\nend\nb --> f\nf --> c\n',
  H + 'flow f\n  direction BT\n  a\n  b --> c\nend\na --> f\n',
  H + 'a --> a\na -- "again" --> a\na -.- a\n',
  'agentflow-beta LR\na --> a\na -- "label" --> a\nb --x b\n',
  H + 'flow f\n  direction LR\n  a --> a\n  a -- "x" --> a\nend\n',
  H + 'a["Tool"]@{ shape: tool } --> b{"Choose?"} -- yes --> c[/In/]\nb -- no --x d@{ shape: refdoc }\nc -.- e@{ shape: action }\n',
  H + 'a@{ shape: decision, label: "A rather long question that wraps over lines?" } --> b\n',
  H + 'connector c["Conn"]\nc --> a\na --> c\n',
  H + 'a --> b\nclick a "https://example.com" "tip"\nclick b href "javascript:alert(1)"\n',
  H + 'a["Note"]:::internal-link --> b\nclassDef internal-link fill:#ffd\n',
  H + 'a --> b\nstyle a fill:#f9f,stroke:#333,stroke-width:4px,font-size:24px,font-weight:bold\nlinkStyle 0 stroke:#f00,color:#00f\n',
  H + 'a --> b --> c --> a\nlinkStyle 0 stroke-linecap:butt,stroke-linejoin:bevel\nlinkStyle 1 stroke-linecap:SQUARE\nlinkStyle 2 stroke-linecap:inherit,stroke-linejoin:round\n',
  'agentflow-beta LR\na --> b & c\nlinkStyle default stroke-linecap:initial,stroke-linejoin:miter\n',
  '---\ntitle: A titled flow with a long name\n---\nagentflow-beta\na --> b\n',
  '---\nconfig:\n  agentflow:\n    wrappingWidth: 60\n    minNodeWidth: 40\n    nodeSpacing: 10\n    rankSpacing: 100\n---\nagentflow-beta\na["A label that wraps"] --> b & c --> d\n',
  '---\nconfig:\n  flowchart:\n    curve: linear\n---\nagentflow-beta\na --> b & c --> d\na --> d\n',
  '---\nconfig:\n  flowchart:\n    curve: step\n---\nagentflow-beta LR\na --> b & c --> d\n',
  H + 'a e1@--> b\ne1@{ curve: stepBefore }\nb e2@--> c\ne2@{ curve: linear }\n',
  H + 'a -- "**bold** and _it_" --> b\nb["`**md** label`"]\n',
  'agentflow-beta LR\n' + Array.from({ length: 12 }, (_, i) => `n${i} --> n${i + 1}`).join('\n') + '\n',
  'agentflow-beta RL\nflow f["Flow"]\n  a --> b --> c\nend\nflow g["Gee"]\n  d --> e\nend\nf --> g\nc --> d\n',
  H + 'global\n  a --> b\nend\n',
  H + 'flow f\nend\nflow g["G"]\nend\nf --> g\n',
  H + 'flow f\n  a --> b --> c\nend\nf --> b\na --> f\n',
  H + 'flow f\n  a\nend\na -- "out" --> f\nf -- "in" --> a\n',
  H + 'flow outer["Outer flow title"]\n  a\n  b\nend\nouter --> a\n',
  H + 'flow outer["Outer flow title"]\n  a\n  b\nend\nouter --> b\nouter -- "into a" --> a\n',
  'agentflow-beta BT\nflow outer["A rather long outer title"]\n  a\nend\na --> outer\n',
  'agentflow-beta LR\nflow f\n  a --> b --> c\nend\nf --> b\nb --> f\n',
  H + 'flow f\n  d{"D?"}\n  r@{ shape: refdoc }\n  i@{ shape: input }\nend\nf --> d\nd --> f\nr --> f\nf --> r\ni --> f\n',
  'agentflow-beta LR\nflow f\n  d{"Decide?"}\n  h@{ shape: action }\nend\nf --> d\nd --> f\nh --> f\n',
  H + 'flow f\n  a --> a\n  a -- "loop" --> a\nend\na --> f\nf --> a\n',
  H + 'flow outer["Outer"]\n  flow inner["Inner flow title"]\n    a\n  end\nend\nouter --> a\n',
  H + 'flow outer["Outer title"]\n  flow inner["Inner"]\n    a --> b\n  end\n  c\nend\nb --> outer\nouter --> a\n',
  '---\nconfig:\n  agentflow:\n    rankSpacing: 5000\n    nodeSpacing: 3000\n    minNodeWidth: 9000\n---\nagentflow-beta\na --> b & c\n',
];

const box = (s) => `${attr(s, 'x') ?? 0},${attr(s, 'y') ?? 0},${attr(s, 'width')},${attr(s, 'height')}`;
// Where a label's first line sits: the first x and y in its text.
const textAt = (s) => {
  const i = s.indexOf('<text');
  if (i < 0) return undefined;
  const m = / x="([^"]*)" y="([^"]*)"/.exec(s.slice(i));
  return `t ${m[1]},${m[2]}`;
};
// The elements of a node's shape, before its label, each with its geometry.
const elements = (s) => {
  const end = s.indexOf('<text');
  const lines = [];
  for (const m of (end < 0 ? s : s.slice(0, end)).matchAll(/<(rect|path|polygon|circle|ellipse)\s[^>]*>/g)) {
    const e = m[0];
    switch (m[1]) {
      case 'rect':
        lines.push(`rect ${box(e)} ${attr(e, 'rx') ?? ''}`);
        break;
      case 'path':
        lines.push(`path ${attr(e, 'd')}${e.includes('stroke-dasharray') ? ' dashed' : ''}`);
        break;
      case 'polygon':
        lines.push(`polygon ${attr(e, 'points')}`);
        break;
      default:
        lines.push(`${m[1]} ${e}`);
    }
  }
  return lines;
};

// The palette slot a flow or a node is drawn in, from the first series color it uses.
const slot = (s) => 's' + (Number(/var\(--pele-series-(\d)/.exec(s)[1]) - 1);

const GROUP = /<g class="pele-(cluster|edge|edge-label|node)[ "]/g;

// The lines of an agent flow's SVG.
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
    const classes = attr(open, 'class').split(' ').slice(kind === 'node' || kind === 'cluster' ? 2 : 1).join(' ');
    switch (kind) {
      case 'cluster': {
        const rect = tag(s, 'rect');
        lines.push(`flow ${id} [${classes}] ${box(rect)} ${slot(s)}`);
        const t = textAt(s);
        if (t) lines.push(t);
        break;
      }
      case 'edge': {
        const path = tag(s, 'path');
        const flags = path.includes(' stroke-dasharray="3 4"') ? ' dotted' : '';
        lines.push(`edge ${id} [${classes}]${flags} ${attr(path, 'd')}`);
        // The edge's line, in the group of all of them and its own.
        lines.push(`stroke ${capJoin([tag(svg, 'g class="pele-edges"'), open, path])}`);
        for (const m of s.matchAll(/<(path|circle) class="pele-marker"[^>]*>/g)) {
          lines.push(m[1] === 'path' ? `m ${attr(m[0], 'd')}` : `m o ${attr(m[0], 'cx')},${attr(m[0], 'cy')}`);
        }
        break;
      }
      case 'edge-label':
        lines.push(`label ${id} ${box(tag(s, 'rect'))}`, textAt(s));
        break;
      case 'node': {
        const shape = /pele-shape-([^\s"]+)/.exec(open)[1];
        lines.push(`node ${id} ${shape} [${classes}] ${attr(open, 'transform')} ${slot(s)}`);
        const a = /<a\s[^>]*?(?<![\w-])href="([^"]*)"/.exec(s);
        if (a) lines.push(`link ${unescape(a[1])}`);
        lines.push(...elements(s));
        const t = textAt(s);
        if (t) lines.push(t);
        break;
      }
    }
  }
  return lines;
}

export default async function ({ pele, src, cases, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { preprocess } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');

  const agentflows = [...WRITTEN, ...agentflowCorpus(pele, cases).map((x) => (typeof x === 'string' ? x : x.src))].filter((s) => {
    try {
      return detect(preprocess(s).text) === 'agentflow';
    } catch {
      return false;
    }
  });

  const out = [];
  const seen = new Set();
  const draw = (source, maxWidth) => {
    let r;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, outputLimit: Infinity });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      // A case written for this suite is written to be drawn.
      if (WRITTEN.includes(source)) console.warn(`not drawn: ${JSON.stringify(source)}: ${e.message}`);
      return false;
    }
    if (r.type !== 'agentflow') return false;
    out.push({ src: source, maxWidth, lines: readSvg(r.svg) });
    return true;
  };
  let drawn = 0;
  const each = (source) => {
    if (seen.has(source)) return;
    seen.add(source);
    // Every third one again in a narrow host, which turns what runs across and does not fit.
    if (draw(source) && drawn++ % 3 === 0) draw(source, 360);
  };
  agentflows.forEach(each);
  const next = mutator(agentflows, FRAGMENTS, random(12));
  while (drawn < 2500) {
    const s = next();
    if (s.length <= 3000) each(s);
  }
  // The test reads a line with a tag among its classes with the tag moved last, so the lines
  // that name one are kept in full.
  write('diagram/agentflow/testdata/pele/render.json', { cases: out }, { keep: (line) => line.includes('clickable') });
}
