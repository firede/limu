// Flowcharts parsed as Pele parses them, from text to FlowDb, and the graph buildFlowGraph
// makes of it: the sources of Pele's tests and Mermaid's specs and documentation, cases
// written here, and those sources mutated at random.
import { enc, peleError } from '../lib/value.mjs';
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';

// Pele's fragments, tests/flowchart/oracle.test.ts, and more of what Pele reads beyond the
// grammar: metadata, links, Unicode.
const FRAGMENTS = [
  ' ', '\n', ';', '-->', '---', '--', '==>', '===', '==', '-.->', '-.-', '-.', '.->', '~~~', '--x', '--o', 'x--',
  'o--', '<--', '<-->', '|', '[', ']', '(', ')', '{', '}', '((', '))', '[[', ']]', '([', '])', '[(', ')]', '(((',
  ')))', '{{', '}}', '[/', '/]', '[\\', '\\]', '>', '(-', '-)', '"', '`', '"`', '`"', '@', '@{', 'e1@', ':::', ':',
  '&', ' & ', ',', '*', '#', '^', 'v', '-', '=', '.', '~', '<', '\\|', '[|', 'x', 'o', 'A', 'B', 'id1', '1', '23',
  'text', 'é', 'ö', 'style ', 'classDef ', 'class ', 'click ', 'href ', 'call ', 'linkStyle ', 'interpolate ',
  'default', 'subgraph ', 'subgraph', 'end', 'end\n', 'direction TB', 'direction LR', 'graph ', 'flowchart ', 'TD',
  'LR', 'TB', 'BT', 'RL', '_blank', '_self', 'accTitle: ', 'accDescr: ', 'accDescr { ', 'fill:#f9f',
  'stroke-width:2px', 'shape: rect', 'label: "x"', '%%', '\t', '\r\n', ' \n', '}\n', 'a()', '()', 'foo',
  '@{ shape: circle }', '@{ shape: Rect }', '@{ shape: nope }', '@{ label: "x", icon: "fa:car", form: circle }',
  '@{ img: "https://x/y.png", w: 60, h: "0x20", pos: t, constraint: on }', '@{ view: collapsed }', 'e1@{ curve: linear }',
  'e1@{ animate: true, animation: fast }', '@{ shape: rect, label: [a, b] }', '@{ label: "a\n  b" }', '@{ "x }',
  'click A "https://x.y/z" "tip" _blank', 'click A href "javascript:alert(1)"', 'click A call cb(1)', 'click A cb "t"',
  'linkStyle 0 ', 'linkStyle default interpolate basis ', ':::cls', '　', '😀', ' ', '中文', '#35;', '#quot;',
  'A[|f:v|t]', '---\ntitle: T\n---\n', '%%{init: {"flowchart": {"curve": "linear", "inheritDir": true}}}%%\n',
];

const WRITTEN = [
  'graph TD\nA-->B',
  'flowchart LR\nsubgraph S\n  x\nend\nsubgraph S\n  y\nend',
  'flowchart TD\nsubgraph\n  a --> b\nend',
  'graph TD\nA-->B\nlinkStyle 1 stroke-width:1px;',
  'graph TD\nA-->B\nB --> [oops]',
  { src: 'graph TD\nA-->B-->C-->D', maxEdges: 2 },
  { src: 'graph TD\nA & B --> C & D', maxEdges: 4 },
  { src: 'graph TD\nA & B --> C & D', maxEdges: 3 },
  { src: 'graph TD\nA-->B', maxEdges: -1 },
  // Metadata of nodes, edges and subgraphs.
  'flowchart TD\nA@{ shape: rect, label: "x" }\nB@{ shape: diamond }\nC@{ shape: lean-r }',
  'flowchart TD\nA@{ shape: Rect }',
  'flowchart TD\nA@{ shape: lean_right }',
  'flowchart TD\nA@{ shape: nope }',
  'flowchart TD\nA@{ icon: "fa:user", form: circle, pos: t, w: 60, h: "0x20" }\nB@{ icon: "fa:car", label: "Car", form: rounded }',
  'flowchart TD\nA@{ img: "https://example.com/a.png", label: "", constraint: on, w: abc, h: [5] }',
  'flowchart TD\nA@{ img: "x.png", label: "  " }\nB@{ icon: [], form: square }\nC@{ icon: 0, img: false, w: 0 }',
  'flowchart TD\nA@{ label: [1, 2], labelType: string }\nB@{ label: 5, labelType: weird }\nC@{ label: { a: 1 } }',
  'flowchart TD\nA@{\n  shape: rect\n  label: |\n    two\n    lines\n}',
  'flowchart TD\nA@{ label: "a\n   b" } --> B@{ shape: circle }',
  'flowchart TD\nA@{ shape: [rect] }',
  'flowchart TD\nA@{ shape: rect',
  'flowchart TD\nA@{ "unclosed }',
  'flowchart TD\nA@{ label: "\\q" }',
  'flowchart TD\nA@{ label: "\\x-1" }',
  'flowchart TD\nA@{ a: 1 } & B@{ b: 2 } --> C',
  'flowchart TD\nA@{ a: 1 }   B',
  'flowchart TD\nA e1@--> B\ne1@{ animate: true, curve: linear }',
  'flowchart TD\nA e1@--> B\ne1@{ animation: fast, curve: 5 }\ne1@{ animate: null, curve: null }',
  'flowchart TD\nA e1@--> B\nA e1@--> C\ne1@{ curve: basis }',
  'flowchart TD\nA e1@-.-> B & C\nD x@y@==> E',
  'flowchart TD\nA e1@--> B\nlinkStyle 0 interpolate step\ne1@{ curve: 0 }',
  'flowchart TD\nsubgraph S\n a --> b\nend\nS@{ view: collapsed, 2: x, a: 1 }\nS@{ 1: y, a: 2 }\nb --> c',
  'flowchart TD\nsubgraph O\n subgraph I\n  a --> b\n end\n c\nend\nI@{ view: collapsed }\nO@{ other: 1 }\nb --> c\na --> z',
  'flowchart TD\nsubgraph O\n subgraph I\n  a --> b\n end\n c\nend\nO@{ view: collapsed }\nI@{ view: collapsed }\nb --> c\nz --> a\nc --> O',
  'flowchart TD\nsubgraph B\n A\nend\nsubgraph A\n B\nend\nA --> B',
  'flowchart TD\nsubgraph S\n a\nend\nS@{ shape: circle }',
  'flowchart TD\nsubgraph S\n a\nend\nS@{ 5 }\nS --> a',
  // Link styles.
  'flowchart TD\nA --> B --> C\nlinkStyle 0,1 stroke:red',
  'flowchart TD\nA --> B --> C\nlinkStyle 0,1 fill:blue,stroke:red',
  'flowchart TD\nA --> B --> C\nlinkStyle default stroke:blue\nlinkStyle 1 color:red',
  'flowchart TD\nA --> B --> C\nlinkStyle 0 interpolate basis stroke:red\nlinkStyle default interpolate step\nC --> D',
  'flowchart TD\nlinkStyle default interpolate monotoneX\nA --> B\nlinkStyle 00 interpolate linear',
  'flowchart TD\nA --> B\nlinkStyle 99999999999999999999 stroke:red',
  'flowchart TD\nlinkStyle 0 stroke:red',
  // Clicks and links.
  'flowchart TD\nA --> B\nclick A "https://example.com/a b" "tip" _blank\nclick B href "javascript:alert(1)"',
  'flowchart TD\nA --> B\nclick A call cb(1,2) "tip"\nclick B cb\nclick A callback "t2"',
  'flowchart TD\nA --> B\nclick A "http://EXAMPLE.com/Path?q=1#frag" _self\nclick B href "#35;quot;" "t"',
  'flowchart TD\nA,B --> C\nclick A,B "  " "both"\nclick C "&#106;avascript:x"',
  'flowchart TD\nA\nclick A href "mailto:a@b.c" "mail" _top',
  'flowchart TD\nA\nclick',
  'flowchart TD\nA\nclick A call cb()',
  'flowchart TD\nA\nclick A call cb(',
  // Props, classes, styles.
  'flowchart TD\nA[|field:value|text] --> B[|__proto__:x|t]\nA[|other:v2|u]\nB[|__proto__:y|w]',
  'flowchart TD\nA --> B\nA[|__proto__:x|t]',
  'flowchart TD\nclassDef a fill:#f9f\\,stroke:red,color:blue\nclassDef b,c font-size:12px\nA:::a --> B:::b\nclass A,B c\nstyle A fill:#fff,stroke:#000\nstyle Z color:red',
  'flowchart TD\nclassDef default fill:#eee\nclassDef node stroke:#111\nA --> B\nclassDef a fill:red;\nclass A a',
  'flowchart TD\nsubgraph S\nA\nend\nclass S,A cls\nclassDef cls fill:red\nstyle S stroke:blue',
  // Accessibility, titles, directions.
  '---\ntitle: The title\n---\nflowchart TD\naccTitle: Hello  \naccDescr: World\nA',
  'flowchart TD\naccDescr {\n  line one\n    line two\n}\nA',
  'flowchart TD\naccTitle:\nA',
  'flowchart TD\nsubgraph S\naccTitle: inside\naccDescr: also\nA\nend',
  'flowchart RL\nsubgraph A\ndirection TB\nx\nend\nsubgraph B\ndirection BT\ny\nend',
  '%%{init: {"flowchart": {"inheritDir": true}}}%%\nflowchart LR\nsubgraph A\nx\nend\nsubgraph B\ndirection TB\ny\nend',
  '---\nconfig:\n  flowchart:\n    curve: stepBefore\n---\nflowchart LR\nA-->B\nlinkStyle 0 interpolate basis\nB-->C',
  '---\nconfig:\n  flowchart:\n    curve: 5\n---\nflowchart LR\nA-->B',
  'graph <\nA',
  'graph >;A',
  'graph ^\nA',
  'graph v\nA',
  'graph TD  \nA',
  'graph TD x',
  'graph\nA',
  'flowchart-elk LR\nA',
  'swimlane-beta\nA',
  'flowchart TD\nA --> B\n   direction LR and more\nsubgraph s\ndirection RL\nend',
  'flowchart TD\nA["direction TB"] --> B',
  // Subgraph titles and ids.
  'flowchart TD\nsubgraph one two\nA\nend\nsubgraph "quoted title"\nB\nend\nsubgraph id [Title]\nC\nend\nsubgraph id2 ["`md`"]\nD\nend',
  'flowchart TD\nsubgraph ""\nA\nend',
  'flowchart TD\nsubgraph a\nend\nsubgraph b\na\nend',
  'flowchart TD\nsubgraph a\nb\nend\nsubgraph b\na\nc\nend',
  'flowchart TD\nsubgraph a\nb;c\nend',
  'flowchart TD\nsubgraph a\n',
  'flowchart TD\nend',
  // Edges.
  'flowchart TD\nA --- B\nA -.- C\nA ==> D\nA ~~~ E\nA --o F\nA --x G\nA <--> H\nA x--x I\nA o--o J\nA <-.-> K\nA <==> L',
  'flowchart TD\nA -- text --> B\nA == text ==> C\nA -. text .-> D\nA -- text --- E\nA -- "quoted" --> F\nA -- x -.-> G',
  'flowchart TD\nA <-- text --> B\nA x-- text --x C\nA o-- text --o D\nA <-- text --x E\nA -->|label| B\nA -->|"q"| B',
  'flowchart TD\nA ---------> B\nA ==========> C\nA -.........-> D\nA ~~~~~~ E',
  'flowchart TD\nA --> B\nA --> B\nA --> B\nB --> A',
  'flowchart TD\nA e1@--> B\nA e1@--> C\nA e1@--> D',
  // Shapes.
  'flowchart TD\nA[sq] --> B(round) --> C((circle)) --> D(((double))) --> E{diamond} --> F{{hex}}\nG>odd] --> H[/lean r/] --> I[\\lean l\\]\nJ[/trap\\] --> K[\\inv/] --> L[(cyl)] --> M([stadium]) --> N[[sub]] --> O(-ellipse-)',
  'flowchart TD\nA["`**bold** _it_`"] --> B["a #35; b #quot;c#quot;"]\nC[a < b > c]',
  'flowchart TD\nA[\'single\'] --> B["say \\"hi\\""]',
  // Unicode, columns counted in code units.
  'flowchart LR\nA[中文] --> B(é)\né --> ü\nÄ[Ö]',
  'flowchart LR\n😀 --> B',
  'flowchart LR\nA["😀😀"] --> B -->|😀| C\nA -- 😀 --> D',
  'flowchart LR\nA["😀"] --> [',
  'flowchart LR\nA　-->　B',
  'flowchart LR\nA --> B C --> D',
  'flowchart LR\nhref　x',
  'flowchart LR\nA[x] --> B',
  'flowchart TD\nA["ﬂ°°35¶ß"]',
  // Errors of all kinds.
  'flowchart TD\nA --> B --> ',
  'flowchart TD\nA -->',
  'flowchart TD\n[A]',
  'flowchart TD\nA[B',
  'flowchart TD\nA(-x)',
  'flowchart TD\nA[/x)]',
  'flowchart TD\nA{x]',
  'flowchart TD\nA["x"y]',
  'flowchart TD\nA["`x`y`"]',
  'flowchart TD\nA@{ shape: rect }}',
  'flowchart TD\nA --> B\nstyle',
  'flowchart TD\nlinkStyle x',
  'flowchart TD\nclassDef',
  'flowchart TD\nclass A',
  'flowchart TD\nsubgraph A[x] y\nend',
  'flowchart TD\nA -- x\nB',
  'flowchart TD\nA ^ B',
  'flowchart TD\n\u0001',
  'not a flowchart',
  'graph',
  '',
];

function dumpDb(db) {
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  const val = (x) => (x === undefined || x === null ? undefined : enc(x));
  const num = (x) => (x === undefined ? undefined : enc(x));
  return {
    direction: db.direction,
    title: opt(db.title),
    accTitle: opt(db.accTitle),
    accDescr: opt(db.accDescr),
    defaultEdgeStyle: db.defaultEdgeStyle,
    defaultInterpolate: db.defaultInterpolate,
    nodes: [...db.nodes.values()].map((n) => ({
      id: n.id,
      text: n.text,
      labelType: n.labelType,
      type: n.type,
      styles: n.styles,
      classes: n.classes,
      dir: n.dir,
      props: Object.entries(n.props ?? {}),
      link: opt(n.link),
      linkTarget: opt(n.linkTarget),
      icon: opt(n.icon),
      form: opt(n.form),
      pos: opt(n.pos),
      img: opt(n.img),
      constraint: opt(n.constraint),
      assetWidth: num(n.assetWidth),
      assetHeight: num(n.assetHeight),
    })),
    edges: db.edges.map((e) => ({
      start: e.start,
      end: e.end,
      type: e.type,
      text: e.text,
      labelType: e.labelType,
      stroke: e.stroke,
      length: e.length,
      style: e.style,
      classes: e.classes,
      id: e.id,
      isUserDefinedId: e.isUserDefinedId,
      interpolate: val(e.interpolate),
      animate: val(e.animate),
      animation: val(e.animation),
    })),
    subgraphs: db.subgraphs.map((s) => ({
      id: s.id,
      nodes: s.nodes,
      title: s.title,
      classes: s.classes,
      dir: s.dir,
      labelType: s.labelType,
      metadata: s.metadata === undefined ? undefined : enc(s.metadata),
    })),
    classes: [...db.classes.values()],
    tooltips: [...db.tooltips.entries()],
  };
}

function dumpGraph(graph) {
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  const val = (x) => (x === undefined || x === null ? undefined : enc(x));
  const num = (x) => (x === undefined ? undefined : enc(x));
  return {
    nodes: graph.nodes.map((n) => ({
      id: n.id,
      label: n.label,
      labelType: n.labelType,
      shape: n.shape,
      isGroup: n.isGroup,
      parentId: n.parentId,
      cssStyles: n.cssStyles,
      cssCompiledStyles: n.cssCompiledStyles,
      cssClasses: n.cssClasses,
      dir: n.dir,
      link: opt(n.link),
      linkTarget: opt(n.linkTarget),
      tooltip: opt(n.tooltip),
      icon: opt(n.icon),
      pos: opt(n.pos),
      img: opt(n.img),
      assetWidth: num(n.assetWidth),
      assetHeight: num(n.assetHeight),
      constraint: opt(n.constraint),
      colorIndex: n.colorIndex,
      metadata: n.metadata === undefined ? undefined : enc(n.metadata),
    })),
    edges: graph.edges.map((e) => ({
      id: e.id,
      isUserDefinedId: e.isUserDefinedId,
      start: e.start,
      end: e.end,
      type: e.type,
      label: e.label,
      labelType: e.labelType,
      thickness: e.thickness,
      pattern: e.pattern,
      minlen: e.minlen,
      arrowTypeStart: e.arrowTypeStart,
      arrowTypeEnd: e.arrowTypeEnd,
      cssCompiledStyles: e.cssCompiledStyles,
      style: e.style,
      classes: e.classes,
      animate: val(e.animate),
      animation: val(e.animation),
      curve: e.curve === '' ? undefined : val(e.curve),
    })),
  };
}

export default async function ({ src, cases, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { flowchart } = await src('diagrams/flowchart/index.ts');
  const { buildFlowGraph } = await src('diagrams/flowchart/graph.ts');
  const { FlowDb } = await src('diagrams/flowchart/db.ts');
  const { parseFlowchart } = await src('diagrams/flowchart/parser.ts');

  const caught = (e) => {
    if (e instanceof RangeError) return { error: { code: 'limit', message: 'Diagram is nested too deeply.' } };
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  // Text that is no flowchart goes to the parser as it is, as Pele's tests give it.
  const direct = (source) => {
    try {
      const db = new FlowDb();
      parseFlowchart(source, db);
      return { db: dumpDb(db), graph: dumpGraph(buildFlowGraph(db)) };
    } catch (e) {
      return caught(e);
    }
  };

  // Pele's parseSource for a flowchart, and the graph its render builds.
  const run = (source, maxEdges = 5000) => {
    let pre;
    let type;
    try {
      pre = preprocess(source);
      type = detect(pre.text);
    } catch (e) {
      return caught(e);
    }
    if (type !== 'flowchart') return { type, direct: direct(source) };
    try {
      const config = pre.config;
      const db = flowchart.parse(encodeEntities(pre.text) + '\n', config, pre.title, pre.lineOffset, {
        maxEdges: maxEdges < 0 ? undefined : maxEdges,
      });
      const flow = config.flowchart ?? {};
      const graph = buildFlowGraph(db, typeof flow.curve === 'string' ? flow.curve : undefined);
      return { type, db: dumpDb(db), graph: dumpGraph(graph) };
    } catch (e) {
      return { type, ...caught(e) };
    }
  };

  const c = cases('flowchart');
  const flowcharts = c.extracted.filter((s) => {
    try {
      return detect(preprocess(s).text) === 'flowchart';
    } catch {
      return false;
    }
  });
  const next = mutator(flowcharts, FRAGMENTS, random(7));
  const fuzz = [];
  while (fuzz.length < 2500) {
    const s = next();
    if (!lone(s) && s.length <= 4000) fuzz.push(s);
  }
  const all = [...WRITTEN, ...c.extracted, ...fuzz].map((x) => (typeof x === 'string' ? { src: x } : x));
  write('diagram/flowchart/testdata/pele/flowchart.json', { flowcharts: all.map((x) => ({ ...x, ...run(x.src, x.maxEdges) })) });
}
