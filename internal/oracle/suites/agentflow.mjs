// Agent flows parsed as Pele parses them, from text to AgentflowDb, with its element mappings,
// diagnostics and semantic model, and the graph buildAgentGraph makes of it: Pele's corpus and
// Mermaid's reference cases, the sources of Pele's tests, flowcharts turned into agent flows,
// cases written here, those sources mutated at random, and random runs of tokens.
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { enc, peleError } from '../lib/value.mjs';
import { fileLiterals, testFiles, docs, lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';

// Pele's fragments, tests/agentflow/oracle.test.ts, and more of what Pele reads beyond the
// grammar: metadata, shapes, flows, connectors.
export const FRAGMENTS = [
  ' ', '  ', '\n', ';', '-->', '--->', '--x', '---x', '--', '---', '-.-', '-..-', '-.', '.-', '-.->', '==>', '~~~',
  '--o', '<--', 'x--', '|', '[', ']', '(', ')', '{', '}', '((', '))', '[[', ']]', '([', '])', '[(', ')]', '(((',
  ')))', '{{', '}}', '[/', '/]', '[\\', '\\]', '>', '(-', '-)', '"', '`', '"`', '`"', '@', '@{', '@{ ', '}\n', 'e1@',
  ':::', ':', '&', ' & ', ',', '*', '#', '^', 'v', '-', '=', '.', '~', '<', '\\|', '[|', 'x', 'o', 'A', 'B', 'id1',
  '1', '23', 'text', 'é', 'ö', 'style ', 'classDef ', 'class ', 'click ', 'href ', 'call ', 'linkStyle ',
  'interpolate ', 'default', 'flow ', 'flow', 'flowx', 'connector ', 'connector', 'global', 'global ', 'global\n',
  'end', 'end\n', 'end ', 'direction TB', 'direction LR', 'direction', 'agentflow-beta ', 'agentflow-beta', 'TD',
  'LR', 'TB', 'BT', 'RL', '_blank', '_self', 'accTitle: ', 'accDescr: ', 'accDescr { ', 'fill:#f9f',
  'stroke-width:2px', 'shape: tool', 'label: "x"', 'view: collapsed', '%%', '%% c', ' %% c\n', '%', '%%{', '%%x@y',
  '\t', '\r\n', '\r', ' \n', ' ', ' ', 'a()', '()', 'foo', '["F"]', '@{ shape: tool }', '@{\n a: 1\n}',
  '@{ shape: decision }', '@{ shape: input }', '@{ shape: refdoc }', '@{ shape: action }', '@{ shape: task }',
  '@{ shape: circle }', '@{ shape: tri }', '@{ shape: Tool }', '@{ shape: lean_right }', '@{ shape: 5 }',
  '@{ view: collapsed }', '@{ label: "L", labelType: string }', '@{ curve: linear }', '@{ animate: true }',
  '@{\n  a: 1,\n  b: [1, 2],\n}', '@{ a: [ }', '😀', '中文', '#35;', '#quot;', 'connector c["C"]\n',
  'flow f["F"]\n', 'global\nA\nend\n', 'flow\nA\nend\n',
];

const SEEDS = [
  'agentflow-beta TB\n  a["A"] --> b{"B?"} -- yes --> c[["tool"]]\n  b -- "no" --x d\n  c -.- e[/in/]\n',
  'agentflow-beta LR\n  flow f["F"]@{ model: "m" }\n    a e1@--> b & c\n    e1@{ instruction: "go" }\n  end\n  f@{ view: collapsed }\n',
  'agentflow-beta\nglobal\n  A\n  B --> C\nend\nflow\n A --> D\nend\nflow x y\n direction LR\n q\nend\n',
  'agentflow-beta TB\n  connector gh["GitHub"]@{ protocol: "mcp" }\n  connector s\n  connector t@{\n    a: 1,\n    b: 2\n  }\n  gh --> s\n',
  '%% one\n\n  %% two\nagentflow-beta TB %% three\n  a --> b %% four\n  %% five\n  b[one %% six\n two] --> c(-el %% seven\n lipse-)\n  d[/tr %% eight\n ap/]\n',
  'agentflow-beta TB\n  a:::big --> b:::small\n  classDef big fill:#f9f,stroke:#333\n  class a,b big\n  style a fill:#bbf\n  linkStyle 0 stroke:red\n  linkStyle default interpolate basis stroke:blue\n  click a href "https://example.com" "tip" _blank\n  click b call cb("x") "tip"\n',
  'agentflow-beta TB\n  accTitle: A title\n  accDescr: A description\n  accDescr {\n   many\n   lines\n  }\n  a(round) --> b((circle)) --> c([stadium]) --> d[(db)] --> e>odd] --> f{{hex}} --> g(((dbl)))\n',
  'agentflow-beta TB\n  flow outer["Outer"]\n    flow inner["Inner"]\n      a --> inner\n    end\n    a --> outer\n  end\n  flow "quoted id"\n   b\n  end\n  flow end x\n  c\n  end\n',
  'agentflow-beta TB\n  a@{ shape: input, value: "x\n  y" }\n  b@{\n    label: "multi"\n    params:\n      - one\n  }\n  a --> b\n',
  'agentflow-beta TB;a-->b;b--xc;c-.-d;flow f;e;end;connector k;global;g;end\n',
];

const H = 'agentflow-beta TB\n';
const WRITTEN = [
  // Shapes, by alias, by flowchart name, removed and unsupported, and wrongly written.
  H + 'a@{ shape: task }\nb@{ shape: tool }\nc@{ shape: input }\nd@{ shape: decision }\ne@{ shape: refdoc }\nf@{ shape: action }\ng@{ shape: round }',
  H + 'a@{ shape: subprocess }\nb@{ shape: subproc }\nc@{ shape: framed-rectangle }\nd@{ shape: lined-document }\ne@{ shape: hex }\nf@{ shape: connector }',
  H + 'a@{ shape: circle }\nb@{ shape: cyl }\nc@{ shape: tri }\nd@{ shape: doc }\ne@{ shape: rect }\nf@{ shape: collapsedGroup }',
  H + 'a((circle)) --> b(((dbl))) --> c([st]) --> d[(cyl)] --> e>odd] --> f(-el-) --> g[/tr\\] --> h[\\it/] --> i[\\ll\\]',
  H + 'a[|field:value|text] --> b[/in/] --> c{dia} --> d{{hex}} --> e[[sub]] --> f(round) --> g[sq]',
  H + 'a@{ shape: Tool }',
  H + 'a@{ shape: lean_right }',
  H + 'a@{ shape: 5 }',
  H + 'a@{ shape: [a, b] }',
  H + 'a@{ shape: { x: 1 } }',
  H + 'a@{ shape: true }',
  H + 'a@{ shape: "" }',
  H + 'a@{ shape: nope }',
  H + 'a@{ shape: constructor }',
  H + 'a@{ shape: toString }',
  // Metadata of every kind of YAML, and of every thing.
  H + 'a@{ label: "L", labelType: string }\nb@{ label: 5 }\nc@{ label: [1, 2], labelType: weird }\nd@{ label: "" }',
  H + 'a@{\n  hello\n}',
  H + 'a@{\n  - x\n  - y\n}',
  H + 'a@{\n  5\n}',
  H + 'a@{\n  null\n}',
  H + 'a@{\n  "😀x"\n}',
  H + 'a@{\n  0\n}\nflow a\nb\nend',
  H + 'a@{ }',
  H + 'a@{ 2: x, 1: y, b: z, 0: w }\na@{ 10: q, a: r }',
  H + 'a@{ __proto__: { x: 1 }, constructor: 2, prototype: 3, ok: { __proto__: 1, y: 2 } }',
  H + 'a@{\n  k: "v",\n  m: [1, 2],\n  n: { a: 1, b: 2 },\n  # comment, here\n  s: \'q,\'\n  t: |\n    block,\n    lines,\n  u: 1,\n}',
  H + 'a@{\n  k: "v",\n  bad: [\n}',
  H + 'a@{ k: [ }',
  H + 'a@{ "x }',
  '---\ntitle: T\n---\n' + H + 'a@{ k: [ }',
  '---\ntitle: T\n---\n' + H + 'a@{ shape: nope }',
  '---\ntitle: T\n---\n' + H + 'a@{ shape: Nope }',
  '---\ntitle: T\n---\n' + H + 'a@{ shape: circle }\nflow f\na\nend\nflow g\nf\nend',
  H + 'a@{ k: "a\n   b" }',
  H + 'a e1@--> b\ne1@{ animate: true, animation: fast, curve: linear }\ne1@{ k: 1 }',
  H + 'a e1@--> b\ne1@{ curve: null, animate: null }\na e1@--> c\ne1@{ curve: 5 }',
  H + 'a e1@--> b\nlinkStyle 0 interpolate step\ne1@{ curve: 0 }',
  H + 'a e1@-.-> b & c\nd x@y@--> e',
  H + 'a e1@--> b\ne1 --> c\ne1@{ x: 1 }\nstyle e1 fill:red',
  H + 'connectors@{ a: 1 }\nconnectors --> b',
  // Flows, global blocks and connectors.
  H + 'flow f["F"]@{ k: 1 }\n  a --> b\nend\nf@{ view: collapsed }\nb --> c\nc --> a',
  H + 'flow o["O"]\n  flow i["I"]\n    a --> b\n  end\n  c\nend\ni@{ view: collapsed }\no@{ view: collapsed }\nb --> c\nz --> a\nc --> o\na --> a',
  H + 'flow o\n  flow i\n    a --> b\n  end\nend\ni@{ view: collapsed }\nb --> z\nz --> i',
  H + 'flow b\n  a\nend\nflow a\n  b\nend\na --> b',
  H + 'flow a\n  flow b\n    c\n  end\nend\nflow c\n  a\nend',
  H + 'flow f\n a\nend\nflow f["Second"]\n b\n direction LR\nend\nflow f\n c\nend',
  H + 'flow " "\n a\nend\nflow ""\n b\nend',
  H + 'flow f\nend\nflow g["T"]\nend',
  H + 'flow f\n flow f\n  a\n end\nend',
  H + 'flow a b c\n x\nend\nflow end x\n y\nend',
  H + 'flow f\n a; b; c\n;\nend',
  H + 'flow f\n direction LR\n a --> b\nend\nflow g\n direction BT\n c\nend',
  '%%{init: {"flowchart": {"inheritDir": true}}}%%\nagentflow-beta LR\nflow f\n a\nend\nflow g\n direction TB\n b\nend',
  '%%{init: {"flowchart": {"curve": "linear"}}}%%\nagentflow-beta LR\na --> b\nlinkStyle 0 interpolate basis\nb --> c',
  '---\nconfig:\n  flowchart:\n    curve: stepBefore\n---\nagentflow-beta LR\na --> b',
  H + 'global\n  a\n  b --> c\nend\nflow f\n  a --> d\n  b\nend',
  H + 'flow f\n  a\n  b\nend\nglobal\n  a\nend\nflow g\n  b\n  a\nend',
  H + 'global\nend\nglobal x\nend',
  H + 'global\n flow f\n  a\n end\nend',
  H + 'connector c["C"]@{ protocol: mcp }\nconnector c\nconnector c["D"]\nc --> a\na --> c\nc@{ x: 1 }',
  H + 'a --> c\nconnector c["C"]\nc --> b',
  H + 'connector " "\nconnector "q"["T"]',
  H + 'flow f\n connector k\n k --> a\nend\nf@{ view: collapsed }',
  H + 'connector',
  H + 'connector c x',
  H + 'connector c@{\n  a: 1,\n  b: 2\n}\nconnector d@{ e: [ }',
  // Edges.
  H + 'a --> b\na --x c\na --- d\na -.- e\na -.-> f\na ---> g\na -----> h\na -..- i',
  H + 'a -- text --> b\na -- text --x c\na -. text .- d\na -- "quoted" --> e\na -- text -.- f\na -. x --> g',
  H + 'a -->|label| b\na -->|"q"| c\na --x|l| d',
  H + 'a --> b\na --> b\na --> b\nb --> a\na --> a\na --> a',
  H + 'a & b --> c & d\na & a --> c',
  H + 'a ------------> b',
  // Styles, classes and clicks.
  H + 'classDef a fill:#f9f\\,stroke:red,color:blue\nclassDef b,c font-size:12px\nA:::a --> B:::b\nclass A,B c\nstyle A fill:#fff,stroke:#000\nstyle Z color:red',
  H + 'classDef default fill:#eee\nclassDef node stroke:#111\nA --> B\nclassDef a fill:red;\nclass A a',
  H + 'flow S\nA\nend\nclass S,A cls\nclassDef cls fill:red\nstyle S stroke:blue',
  H + 'A --> B --> C\nlinkStyle 0,1 stroke:red\nlinkStyle default stroke:blue\nlinkStyle 1 color:red',
  H + 'A --> B\nlinkStyle 5 stroke:red',
  H + 'A --> B\nlinkStyle 99999999999999999999 stroke:red',
  H + 'A --> B\nclick A "https://example.com/a b" "tip" _blank\nclick B href "javascript:alert(1)"',
  H + 'A --> B\nclick A call cb(1,2) "tip"\nclick B cb\nclick A callback "t2"',
  H + 'A,B --> C\nclick A,B "  " "both"\nclick C "&#106;avascript:x"',
  H + 'A[|__proto__:x|t]\nA[|__proto__:y|u]\nB --> C\nB[|f:v|t]\nB[|__proto__:z|w]',
  // Accessibility, titles, directions, comments.
  '---\ntitle: The title\n---\n' + H + 'accTitle: Hello  \naccDescr: World\nA',
  H + 'accDescr {\n  line one\n    line two\n}\nA',
  H + 'flow S\naccTitle: inside\naccDescr: also\nA\nend',
  'agentflow-beta\nA',
  'agentflow-beta LR\nA',
  'agentflow-beta RL;A',
  'agentflow-beta BT %% c\nA',
  'agentflow-beta TD  \nA',
  'agentflow-beta <\nA',
  'agentflow-beta >;A',
  'agentflow-beta ^\nA',
  'agentflow-beta v\nA',
  'agentflow-beta TB x',
  '%% c\n  %% d\nagentflow-beta TB\nA %% e\n%% f',
  H + 'a[x %% c\ny]\nb(-x %% c\ny-)\nc[/x %% c\n/]',
  H + 'a[x % y]',
  H + 'a --> b\n   direction LR and more\nflow s\ndirection RL\nend',
  H + 'a["direction TB"] --> b',
  H + 'a}   \n b}\t\r\n',
  'agentflow-beta TB\r\na --> b\r\nflow f\r\n c\r\nend\r\n',
  'agentflow-beta TB\na b --> c\n d',
  H + 'a["😀😀"] --> b -->|😀| c\na -- 😀 --> d\n中文 --> é',
  H + 'a　-->　b',
  // Errors of all kinds.
  H + 'a --> b --> ',
  H + 'a -->',
  H + '[A]',
  H + 'A[B',
  H + 'A{x]',
  H + 'A@{ shape: rect }}',
  H + 'flow f\na',
  H + 'end',
  H + 'flow f["T"] x\nend',
  H + 'global x',
  H + 'a ==> b',
  H + 'a ~~~ b',
  H + 'a -- x\nb',
  H + 'a ^ b',
  H + '\u0001',
  H + 'a@{ k: 1 }   b',
  'agentflow-beta',
  'agentflow-beta\n',
  'not an agent flow',
  '',
  { src: H + 'a --> b --> c --> d', maxEdges: 2 },
  { src: H + 'a & b --> c & d', maxEdges: 3 },
  { src: H + 'a --> b', maxEdges: -1 },
  // Columns after a token that spans lines count UTF-16 code units of its last line.
  H + 'a["one\ndeux été"] --> b\nc --> d',
  H + 'a["one\n😀中"] --> b@{ shape: tool }',
  H + 'a@{\n  label: "éé" }  --> b',
  // Metadata whose trailing commas are stripped around a block scalar, which keeps its own.
  H + 'a@{\n  note: |\n    keep, this,\n  b: [1, 2],\n  c: [3],\n}\na --> x',
  H + 'a@{\n  note: >\n    folded,\n\n    more,\n  shape: tool,\n  d: { e: 1, },\n}',
  // Flows that would contain themselves, or each other.
  H + 'flow f\n  f --> a\nend',
  H + 'flow a\n  flow b\n    a\n  end\nend',
  H + 'flow a\n  b\nend\nflow b\n  a\nend\nflow c\n  a\nend',
  H + 'flow p\n  x\nend\nflow q\n  y\nend\nflow r\n  z\nend\np --> q --> r',
  // Text given to the parser as it is, as Pele's tests give it, with the carriage returns
  // that preprocessing would take out.
  { src: 'agentflow-beta\r\na --> b\r\n\r\nc -->\r\n', forced: true },
  { src: 'agentflow-beta\r\na --> b\r\nb@{ shape: nope }\r\nc@{ shape: [ }\r\n', forced: true },
  { src: 'agentflow-beta\r\na["x\r\ny"] --> b\rc --> d\r\nflow f\r\n  e\r\nend\r\n', forced: true },
  { src: 'agentflow-beta\n\u2028a --> b\u2029c\n', forced: true },
];

// Pele's loadCorpus: the strings of the compat specs that hold an agent flow, and the docs.
function loadCorpus(pele) {
  const seeds = new Set();
  for (const file of testFiles(pele, 'tests/compat/agentflow', /\.spec\.[jt]s$/, true)) for (const s of fileLiterals(file, /agentflow-beta/)) seeds.add(s);
  for (const s of docs(pele, 'agentflow')) seeds.add(s);
  return [...seeds];
}

function dumpPosition(p) {
  return p === undefined ? undefined : [p.startLine, p.startColumn, p.endLine, p.endColumn, p.startIndex, p.endIndex];
}

function dumpDiagnostics(list) {
  return list.map((d) => ({
    id: d.id,
    severity: d.severity,
    message: d.message,
    nodeId: d.nodeId,
    edgeId: d.edgeId,
    position: dumpPosition(d.position),
  }));
}

const meta = (m) => (m === undefined ? undefined : enc(m));
const opt = (x) => (x === undefined || x === '' ? undefined : x);
const val = (x) => (x === undefined || x === null ? undefined : enc(x));

function dumpSemantic(s) {
  return {
    direction: s.direction,
    vertices: s.vertices.map((v) => ({ id: v.id, label: v.label, shape: v.shape, vertexKind: v.vertexKind, metadata: meta(v.metadata) })),
    edges: s.edges.map((e) => ({
      start: e.start,
      end: e.end,
      id: e.id,
      label: e.label,
      type: e.type,
      stroke: e.stroke,
      edgeSemantic: e.edgeSemantic,
      length: e.length,
      metadata: meta(e.metadata),
    })),
    subGraphs: s.subGraphs.map((g) => ({ id: g.id, type: g.type, title: g.title, nodes: g.nodes, direction: g.direction, metadata: meta(g.metadata) })),
    connectors: s.connectors.map((c) => ({ id: c.id, title: c.title, metadata: meta(c.metadata) })),
    diagnostics: dumpDiagnostics(s.diagnostics),
  };
}

function dumpDb(db) {
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
      metadata: meta(n.metadata),
      isConnector: n.isConnector === true,
    })),
    edges: db.edges.map((e) => ({
      start: e.start,
      end: e.end,
      type: e.type,
      text: e.text,
      labelType: e.labelType,
      stroke: e.stroke,
      edgeSemantic: e.edgeSemantic,
      length: e.length,
      style: e.style,
      classes: e.classes,
      id: e.id,
      isUserDefinedId: e.isUserDefinedId,
      interpolate: val(e.interpolate),
      animate: val(e.animate),
      animation: val(e.animation),
      metadata: meta(e.metadata),
    })),
    subgraphs: db.subgraphs.map((s) => ({
      id: s.id,
      nodes: s.nodes,
      title: s.title,
      classes: s.classes,
      dir: s.dir,
      labelType: s.labelType,
      type: s.type,
      metadata: meta(s.metadata),
    })),
    connectors: [...db.connectors.keys()],
    classes: [...db.classes.values()],
    tooltips: [...db.tooltips.entries()],
    mappings: db.mappings.map((m) => [m.id, m.type, ...dumpPosition(m.position)]),
    diagnostics: dumpDiagnostics(db.diagnostics),
    semantic: dumpSemantic(db.semanticModel()),
  };
}

function dumpGraph(graph) {
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
      colorIndex: n.colorIndex,
      kind: n.kind,
      metadata: meta(n.metadata),
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
      metadata: meta(e.metadata),
    })),
  };
}

// Runs of the fragments after a start in each state of the lexer, as Pele's oracle test makes
// them: mutation rarely joins two tokens without a space, which is where keyword boundaries
// and lookahead matter.
function sequences(rnd, count) {
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const pick = () => pieces[Math.floor(rnd() * pieces.length)];
  const openers = [
    'agentflow-beta\n', H + 'a', H + 'a --> ', H + 'a -- ', H + 'flow f\n', H + 'flow ', H + 'connector ', H + 'a[',
    H + 'a(-', H + 'a[/', H + 'a@{', H + 'style a ', H + 'click a ', 'agentflow-beta', '',
  ];
  const out = [];
  for (let i = 0; i < count; i++) {
    let src = openers[Math.floor(rnd() * openers.length)];
    for (let k = 1 + Math.floor(rnd() * 10); k > 0; k--) src += pick() + (rnd() < 0.3 ? ' ' : '');
    out.push(src);
  }
  return out;
}

// The agent flows the suites start from: Pele's corpus, Mermaid's reference cases and
// upstream fixtures, the seeds above, and flowcharts turned into agent flows.
export function agentflowCorpus(pele, cases) {
  const reference = JSON.parse(readFileSync(join(pele, 'tests/corpus/agentflow-reference.json'), 'utf8')).map((r) => r.src);
  const upstream = join(pele, 'tests/compat/agentflow/upstream');
  const fixtures = readdirSync(upstream)
    .filter((name) => name.endsWith('.mmd'))
    .sort()
    .map((name) => readFileSync(join(upstream, name), 'utf8'));
  // The agent flow specs are thin on the syntax it shares with flowcharts, so flowcharts are
  // reused with their keywords swapped, as Pele's oracle test does.
  const fromFlowchart = cases('flowchart')
    .extracted.filter((s) => /graph|flowchart/.test(s))
    .map((s) => s.replace(/\b(?:flowchart-elk|flowchart|graph)\b/g, 'agentflow-beta').replace(/\bsubgraph\b/g, 'flow'));
  return [...new Set([...loadCorpus(pele), ...fixtures, ...SEEDS, ...reference, ...fromFlowchart])];
}

export default async function ({ pele, src, cases, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { agentflow } = await src('diagrams/agentflow/index.ts');
  const { agentGraph } = await src('diagrams/agentflow/render.ts');
  const { buildAgentGraph } = await src('diagrams/agentflow/graph.ts');
  const { AgentflowDb } = await src('diagrams/agentflow/db.ts');
  const { parseAgentflow } = await src('diagrams/agentflow/parser.ts');

  const caught = (e) => {
    if (e instanceof RangeError) return { error: { code: 'limit', message: 'Diagram is nested too deeply.' } };
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  // Text that is no agent flow goes to the parser as it is, as Pele's tests give it.
  const direct = (source) => {
    try {
      const db = new AgentflowDb();
      parseAgentflow(source, db);
      // Built twice, as a model drawn twice is: the diagnostics of the second are not reported again.
      buildAgentGraph(db, 8);
      const graph = buildAgentGraph(db, 8);
      return { db: dumpDb(db), graph: dumpGraph(graph) };
    } catch (e) {
      return caught(e);
    }
  };

  // Pele's parseSource for an agent flow, and the graph its render builds.
  const run = (source, maxEdges = 5000) => {
    let pre;
    let type;
    try {
      pre = preprocess(source);
      type = detect(pre.text);
    } catch (e) {
      return caught(e);
    }
    if (type !== 'agentflow') return { type, direct: direct(source) };
    try {
      const config = pre.config;
      const db = agentflow.parse(encodeEntities(pre.withComments) + '\n', config, pre.title, pre.lineOffset, {
        maxEdges: maxEdges < 0 ? undefined : maxEdges,
      });
      const graph = agentGraph(db, config);
      return { type, db: dumpDb(db), graph: dumpGraph(graph) };
    } catch (e) {
      return { type, ...caught(e) };
    }
  };

  const corpus = agentflowCorpus(pele, cases);

  const next = mutator(corpus, FRAGMENTS, random(7));
  const fuzz = [];
  while (fuzz.length < 3000) {
    const s = next();
    if (!lone(s) && s.length <= 4000) fuzz.push(s);
  }
  const runs = sequences(random(8), 1000).filter((s) => !lone(s));
  const all = [...WRITTEN, ...corpus, ...fuzz, ...runs].map((x) => (typeof x === 'string' ? { src: x } : x));
  write('diagram/agentflow/testdata/pele/agentflow.json', { agentflows: all.map((x) => ({ ...x, ...(x.forced ? { direct: direct(x.src) } : run(x.src, x.maxEdges)) })) });
}
