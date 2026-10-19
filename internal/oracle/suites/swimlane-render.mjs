// Swimlane diagrams drawn end to end: what Pele's render draws, read back out of its SVG as
// render.mjs reads a flowchart's, lanes included. The sources are Mermaid's documentation
// examples, those of Pele's swimlane tests, flowcharts made swimlanes by their header,
// sources written here, and mutations of all of them.
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { mutator } from '../lib/mutate.mjs';
import { readSvg } from './flowchart-render.mjs';

const FRAGMENTS = [
  '\n', ' --> ', ' --- ', ' -.-> ', ' ==> ', ' ~~~ ', ' --x ', ' --o ', ' <--> ', ' -->|label| ', ' -- text --> ', ' ---> ',
  '[', ']', '(', ')', '{', '}', '((', '))', '{{', '}}', '>', 'A', 'B', 'C', 'a1', 'long label that wraps past the width a node has',
  '"', '`**bold** _it_`', '<br>', '\nsubgraph S\n', '\nsubgraph T [Title of a lane]\n', '\nsubgraph Inner\n', '\nend\n',
  '\ndirection LR\n', '\ndirection TB\n', ' & ', ':::cls', '\nclassDef cls fill:#f96\n', '\nstyle A stroke:#f00\n',
  '\nstyle S fill:#00f\n', '\nlinkStyle 0 stroke:#0f0\n', '\nclick A "https://x.y/z" "tip"\n', '@{ shape: circle }',
  '@{ shape: diam }', '@{ icon: "fa:user", form: circle, label: "U" }', 'e1@{ curve: linear }', '中文', '😀', '\nA --> A\n',
  '\nS --> A\n', '\nA --> S\n', ' LR', ' TD', ' BT', ' RL',
];

// Sources written as Pele's swimlane tests write them, and more.
const ORDER = `swimlane-beta LR
  subgraph Customer
    browse[Browse] --> pay[Pay]
  end
  subgraph Warehouse
    pick[Pick] --> ship[Ship]
  end
  pay -- paid --> pick`;

function written() {
  const out = [ORDER, ORDER.replace('LR', 'TD'), ORDER.replace('LR', 'BT'), ORDER.replace('LR', 'RL')];
  for (const dir of ['TB', 'BT', 'LR', 'RL']) out.push(`swimlane-beta ${dir}\n  subgraph L\n    a --> b\n  end`);
  out.push(
    'swimlane-beta TD\n  subgraph A\n    a1 --> a2\n  end\n  subgraph B\n    b1 --> b2\n  end\n  a1 --> b2',
    'swimlane-beta TD\n  start --> a\n  subgraph Team\n    a --> b\n  end',
    'swimlane-beta TD\n  a --> b',
    'swimlane-beta\n  a --> b --> c',
    'swimlane-beta TD\n  subgraph One\n    subgraph Inner\n      a --> b\n    end\n    c\n  end\n  subgraph Two\n  end\n  b --> c',
    'swimlane-beta TD\n  subgraph L\n    direction LR\n    a --> b\n  end',
    // A direction inside a lane would turn its self-loops; Pele's swimlane drops it.
    'swimlane-beta TD\n  subgraph L\n    direction LR\n    a --> a\n    a -- again --> a\n    a --> b\n  end\n  subgraph M\n    direction RL\n    subgraph G\n      direction BT\n      c --> c\n    end\n  end\n  b --> c',
    '---\ntitle: Orders\n---\nswimlane-beta LR\n  accTitle: Order flow\n  accDescr: Who does what\n  subgraph L[Lane **one**]\n    a{Choice} --> b[(Store)]\n    b --> b\n  end\n  click a "https://example.com" "tip"\n  style a fill:#f00\n  classDef hot stroke:#0f0\n  class b hot\n  style L fill:#00f',
    'swimlane-beta LR\n  a\n  click a "https://example.com"',
    '---\ntitle: Delivery\n---\nswimlane-beta LR\naccTitle: Accessible delivery\naccDescr: Delivery process\nstart --> a\nsubgraph Team\n subgraph Inner\n a{Ready?} --> b[(Store)]\n b --> b\n end\n c --> d --> e\nend\nsubgraph Empty\nend\nb --> c\nclick a "https://example.com"\nstyle a fill:#f00\nclassDef hot stroke:#0f0\nclass b hot',
    '---\nconfig:\n  swimlane:\n    rankSpacing: 200\n---\nswimlane-beta TD\n  subgraph L\n    a --> b\n  end',
    '---\nconfig:\n  flowchart:\n    rankSpacing: 200\n    nodeSpacing: 90\n---\nswimlane-beta TD\n  subgraph L\n    a --> b\n    a --> c\n  end',
    '---\nconfig:\n  swimlane:\n    nodeSpacing: 10\n  flowchart:\n    rankSpacing: 200\n---\nswimlane-beta LR\n  subgraph L\n    a --> b\n    a --> c\n  end\n  subgraph M\n    d\n  end\n  c --> d',
    '---\nconfig:\n  swimlane: "x"\n  flowchart:\n    curve: linear\n---\nswimlane-beta TD\n  subgraph L\n    a --> b --> c\n  end\n  subgraph M\n    d\n  end\n  a --> d --> c',
    '---\nconfig:\n  swimlane:\n    curve: step\n    wrappingWidth: 60\n---\nswimlane-beta LR\n  subgraph L[A lane with a long title that runs on]\n    a[A step with a long label to wrap] -- a label to wrap as well --> b\n  end\n  subgraph M\n    c\n  end\n  b --> c',
    'swimlane-beta LR\n  subgraph sales [Sales team]\n    lead[Qualify lead]\n    quote[Prepare quote]\n  end\n  subgraph Empty1\n  end\n  subgraph finance [Finance team]\n    review[Review terms]\n  end\n  lead --> quote --> review\n  sales --> finance\n  review --> sales',
    'swimlane-beta TD\n  subgraph A\n    a1 --> a2 --> a3 --> a4\n  end\n  subgraph B\n    b1 --> b2\n  end\n  subgraph C\n    c1\n  end\n  a4 -- back --> a1\n  a1 --> c1\n  c1 -- long way --> a4\n  b2 --> a1\n  a3 ---> b1',
    'swimlane-beta TD\n  subgraph A\n    a1 -->|yes| a2\n    a1 -->|no| a3\n    a1 --> a1\n    a2 -- loop --> a2\n  end\n  subgraph B\n    b1@{ shape: circle } --> b2@{ icon: "fa:user", form: circle, label: "U" }\n  end\n  a2 --> b1\n  b2 -.-> a3\n  a3 ==> b2',
    'swimlane-beta RL\n  subgraph L["`**bold** lane`"]\n    a["`a *markdown* label`"]\n  end\n  subgraph M:::hot\n    b\n  end\n  classDef hot fill:#f96,stroke:#333,stroke-width:4px\n  a --> b',
    'swimlane-beta TD\n  subgraph L1\n    subgraph G1 [Group one]\n      subgraph G2 [Group two]\n        a --> b\n      end\n      c\n    end\n    d\n  end\n  subgraph L2\n    subgraph G3\n    end\n    e\n  end\n  a --> e\n  G1 --> d\n  e --> G3',
  );
  // Smaller cases of Pele's stress tests.
  const lanes = (count, body) => Array.from({ length: count }, (_, k) => `  subgraph L${k}\n${body(k)}\n  end`).join('\n');
  {
    let src = 'swimlane-beta TD\n' + lanes(6, (k) => Array.from({ length: 10 }, (_, i) => `    n${k}_${i}`).join('\n')) + '\n';
    let seed = 7;
    const next = (n) => (seed = (seed * 1103515245 + 12345) & 0x7fffffff) % n;
    for (let e = 0; e < 80; e++) src += `  n${next(6)}_${next(10)} --> n${next(6)}_${next(10)}\n`;
    out.push(src);
  }
  {
    let src = 'swimlane-beta LR\n' + lanes(4, (k) => Array.from({ length: 10 }, (_, i) => `    n${i * 4 + k}`).join('\n')) + '\n';
    for (let i = 0; i < 39; i++) src += `  n${i} --> n${i + 1}\n`;
    out.push(src);
  }
  out.push('swimlane-beta TD\n' + lanes(30, (k) => `    a${k} --> b${k}`) + '\n');
  {
    let src = 'swimlane-beta TD\n  subgraph A\n';
    for (let i = 0; i < 60; i++) src += `    a${i} --> a${i + 1}\n`;
    src += '  end\n  subgraph B\n    b\n  end\n';
    for (let i = 0; i < 10; i++) src += `  a0 --> a${60 - i}\n  b --> a${50 - i}\n`;
    out.push(src);
  }
  out.push('swimlane-beta TD\n  subgraph A\n    a\n  end\n  subgraph B\n    b\n  end\n' + '  a -- label --> b\n'.repeat(30));
  out.push('swimlane-beta TD\n  subgraph A\n    a\n  end\n' + '  a --> a\n'.repeat(12));
  out.push('swimlane-beta TD\n  subgraph L\n' + Array.from({ length: 30 }, (_, i) => `subgraph g${i}\n`).join('') + 'a --> b\n' + 'end\n'.repeat(31));
  out.push('swimlane-beta TD\n' + lanes(8, (k) => `    a${k}`) + '\n' + Array.from({ length: 40 }, (_, i) => `  L${i % 8} --> L${(i * 7 + 3) % 8}`).join('\n') + '\n');
  {
    let src = 'swimlane-beta RL\n' + lanes(5, (k) => Array.from({ length: 8 }, (_, i) => `    n${k * 8 + i}`).join('\n')) + '\n';
    for (let i = 0; i < 40; i++) src += `  n${i} --> n${(i * 37 + 11) % 40}\n  n${(i * 37 + 11) % 40} --> n${i}\n`;
    out.push(src);
  }
  for (const name of ['__proto__', 'constructor', 'toString', 'hasOwnProperty']) {
    out.push(`swimlane-beta TD\n  subgraph ${name}\n    a --> b\n  end\n  subgraph other\n    c\n  end\n  b --> c`);
  }
  return out;
}

// Pele's stress tests at full size: past the budgets of the lane layout's routing.
function stress() {
  const lanes = (count, body) => Array.from({ length: count }, (_, k) => `  subgraph L${k}\n${body(k)}\n  end`).join('\n');
  const out = [];
  {
    let src = 'swimlane-beta TD\n' + lanes(20, (k) => Array.from({ length: 100 }, (_, i) => `    n${k}_${i}`).join('\n')) + '\n';
    let seed = 7;
    const next = (n) => (seed = (seed * 1103515245 + 12345) & 0x7fffffff) % n;
    for (let e = 0; e < 3000; e++) src += `  n${next(20)}_${next(100)} --> n${next(20)}_${next(100)}\n`;
    out.push(src);
  }
  {
    let src = 'swimlane-beta LR\n' + lanes(8, (k) => Array.from({ length: 250 }, (_, i) => `    n${i * 8 + k}`).join('\n')) + '\n';
    for (let i = 0; i < 1999; i++) src += `  n${i} --> n${i + 1}\n`;
    out.push(src);
  }
  {
    let src = 'swimlane-beta TD\n  subgraph A\n';
    for (let i = 0; i < 1500; i++) src += `    a${i} --> a${i + 1}\n`;
    src += '  end\n  subgraph B\n    b\n  end\n';
    for (let i = 0; i < 40; i++) src += `  a0 --> a${1500 - i}\n  b --> a${1400 - i}\n`;
    out.push(src);
  }
  return out;
}

// The templates of Pele's inert-output tests, filled with its payloads.
const TEMPLATES = [
  (p) => `swimlane-beta TD\n  subgraph L["${p}"]\n    a\n  end`,
  (p) => `swimlane-beta LR\n  subgraph L["${p}"]\n    a["${p}"] -- "${p}" --> b\n  end`,
  (p) => `swimlane-beta TD\n  subgraph ${p}\n    a\n  end`,
  (p) => `swimlane-beta ${p}\n  a --> b`,
  (p) => `swimlane-beta TD\n  subgraph L\n    a\n  end\n  style L ${p}\n  class L ${p}`,
  (p) => `swimlane-beta LR\n  subgraph L\n    a\n  end\n  click a "${p}" "${p}"\n  L:::${p}`,
  (p) => `---\ntitle: "${p}"\nconfig:\n  swimlane: "${p}"\n---\nswimlane-beta TD\n  a`,
  (p) => `swimlane-beta RL\n  subgraph L["\`**${p}**\`"]\n    a\n  end\n  accTitle: ${p}\n  accDescr: ${p}`,
];

function payloads(pele) {
  const text = readFileSync(join(pele, 'tests/support/inert.ts'), 'utf8');
  const start = text.indexOf('export const PAYLOADS = [');
  const end = text.indexOf('\n];', start);
  return new Function('return ' + text.slice(text.indexOf('[', start), end + 2))();
}

// A flowchart made a swimlane diagram: its header replaced.
const toSwimlane = (s) => s.replace(/^([ \t]*)(?:flowchart-elk|flowchart|graph)(?=[\s;]|$)/m, '$1swimlane-beta');

export default async function ({ pele, src, cases, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { preprocess } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');
  const isSwimlane = (s) => {
    try {
      return detect(preprocess(s).text) === 'swimlane';
    } catch {
      return false;
    }
  };

  const out = [];
  const draw = (source, maxWidth, more = {}) => {
    let r;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, outputLimit: Infinity, ...more });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return false;
    }
    if (r.type !== 'swimlane') return false;
    out.push({
      src: source, maxWidth, keep: more.autoDirection === false, breakpoint: more.directionBreakpoint, unlimited: more.limit === Infinity,
      lines: readSvg(r.svg),
    });
    return true;
  };
  // Each in hosts of every width, wide and narrow, turning or not.
  const everyWidth = (source) => {
    if (!draw(source)) return;
    for (const maxWidth of [360, 480, 320, 160, 100]) draw(source, maxWidth);
    draw(source, 480, { autoDirection: false });
    draw(source, 700, { directionBreakpoint: 800 });
    draw(source, 480, { directionBreakpoint: 480 });
  };
  let drawn = 0;
  const each = (source) => {
    // Every third one again in a narrow host, which turns or tightens what does not fit.
    if (draw(source) && drawn++ % 3 === 0) draw(source, 360);
  };

  const docs = JSON.parse(readFileSync(join(pele, 'tests/corpus/swimlane-docs.json'), 'utf8'));
  docs.forEach(everyWidth);
  const own = written();
  own.forEach(everyWidth);
  for (const s of stress()) {
    draw(s, undefined, { limit: Infinity });
    draw(s, 360, { limit: Infinity });
  }
  for (const template of TEMPLATES) for (const p of payloads(pele)) each(template(p));
  const flowcharts = cases('flowchart').extracted.map(toSwimlane).filter(isSwimlane);
  flowcharts.forEach(each);
  const corpus = [...docs, ...own, ...flowcharts];
  const next = mutator(corpus, FRAGMENTS, random(29));
  while (drawn < 2200) {
    const s = next();
    // A mutation may split a pair of surrogates, which JSON cannot carry to Go as it is.
    if (s.length <= 3000 && s.isWellFormed()) each(s);
  }
  console.log(`swimlane: ${docs.length} docs, ${own.length} written, ${flowcharts.length} flowcharts, ${out.length} drawn`);
  write('diagram/swimlane/testdata/pele/render.json', { cases: out });
}
