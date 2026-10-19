// Requirement diagrams parsed as Pele parses them, from text to tokens and RequirementDb, and
// the graph buildRequirementGraph makes of it: the sources of Pele's tests and Mermaid's
// specs and documentation, cases written here, and those sources mutated at random.
import { peleError } from '../lib/value.mjs';
import { lone, picker } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';

// Pele's fragments, tests/requirement/oracle.test.ts, and more of what reaches past the
// grammar: Unicode, entities, front matter.
export const FRAGMENTS = [
  ' ', '  ', '\n', '\n\n', ' \n', '\r\n', '\r', '\t', ' ', ' ', ';', ':', ':::', '::', ',', ', ', '{', '}', '{\n', '}\n',
  '-', '->', '<-', '<', '>', '=', '"', '""', '"a b"', '#', '%', '%%', '# c', '.', '(', ')', '_', 'é', 'ö', '1', '1.2', '42',
  'requirementDiagram', 'requirementDiagram\n', 'RequirementDiagram', 'requirement ', 'functionalRequirement ',
  'interfaceRequirement ', 'performanceRequirement ', 'physicalRequirement ', 'designConstraint ', 'element ',
  'ELEMENT ', 'id', 'id: ', 'ID: ', 'text', 'text: ', 'risk', 'risk: ', 'verifymethod: ', 'verifyMethod', 'type',
  'type: ', 'docref: ', 'docRef', 'low', 'medium', 'high', 'High', 'analysis', 'demonstration', 'inspection', 'test',
  'Test', 'contains', 'copies', 'derives', 'satisfies', 'verifies', 'refines', 'traces', ' - contains -> ',
  ' <- copies - ', 'style ', 'style', 'classDef ', 'classdef', 'class ', 'class', 'fill:#f9f', 'stroke-width:2px',
  'color: red', 'font-weight:bold', '50%', 'direction TB', 'direction BT', 'direction RL', 'direction LR',
  'direction', 'direction\nTB', 'DIRECTION lr', 'direction  TB x', 'title x', 'title ', 'Title\nx', 'accTitle: ',
  'accTitle', 'accDescr: ', 'accDescr { ', 'accDescr{', 'acc', 'a', 'b', 'test_req', 'test_entity', 'foo bar',
  '__proto__', 'constructor', 'x:::y', 'a,b', '*x*', '**y**',
  '中文', '😀', '　', ' ', '#35;', '#quot;', '&amp;', '<br>', 'font-size:20px', 'font-style:italic',
  '---\ntitle: T\n---\n', '%% c\n', 'direction　LR', 'title　x',
];

// The payloads of Pele's tests/support/inert.ts, which the templates of its security tests fill in.
const PAYLOADS = [
  '<script>alert(1)</script>',
  '"><script>alert(1)</script>',
  '<img src=x onerror=alert(1)>',
  '<svg onload=alert(1)>',
  '<a href="javascript:alert(1)">x</a>',
  '<foreignObject><iframe src=javascript:alert(1)></iframe></foreignObject>',
  '</text></g></svg><script>alert(1)</script>',
  'javascript:alert(1)',
  'JaVaScRiPt:alert(1)',
  ' javascript:alert(1)',
  'jav&#x09;ascript:alert(1)',
  'data:text/html,<script>alert(1)</script>',
  'vbscript:msgbox(1)',
  "' onload='alert(1)",
  '" onmouseover="alert(1)',
  'x" onclick="alert(1)" y="',
  '#60;script#62;alert(1)#60;/script#62;',
  '#106;avascript:alert(1)',
  '#34; onload=#34;alert(1)',
  'url#40;javascript:alert#40;1#41;#41;',
  'red#59;background:url#40;//evil.example/x#41;',
  '&lt;script&gt;alert(1)&lt;/script&gt;',
  '&#60;script&#62;',
  'url(javascript:alert(1))',
  'expression(alert(1))',
  'red;background:url(//evil.example/x)',
  'red" onload="alert(1)',
  ']]><script>alert(1)</script>',
  '--><script>alert(1)</script>',
  '\u0000\u0001\u0008\u000b\u001f',
  '__proto__',
  'constructor',
  'toString',
  'hasOwnProperty',
  '{{7*7}}',
  '${alert(1)}',
  '%0ajavascript:alert(1)',
  'file:///etc/passwd',
  '//evil.example/share',
  '\\\\evil.example\\share',
  'smb://evil.example/share',
  'obsidian://open?vault=x',
  '\\"><script>',
];

const BODY = 'requirement a {\nid: 1\n}\nelement b {\ntype: t\n}\n';
const KINDS = ['requirement', 'functionalRequirement', 'interfaceRequirement', 'performanceRequirement', 'physicalRequirement', 'designConstraint'];
const RELATIONS = ['contains', 'copies', 'derives', 'satisfies', 'verifies', 'refines', 'traces'];

export const WRITTEN = [
  // Every kind of requirement, with every field, quoted and not.
  ...KINDS.map(
    (kind, k) =>
      `requirementDiagram\n${kind} r${k} {\nid: ${k}.${k}\ntext: "the text of ${kind}"\nrisk: ${['low', 'medium', 'high'][k % 3]}\nverifymethod: ${
        ['analysis', 'demonstration', 'inspection', 'test'][k % 4]
      }\n}\n`
  ),
  'requirementDiagram\nrequirement "quoted name" {\nid: "1.2.3"\ntext: plain text with words\nrisk: Medium\nverifyMethod: Inspection\n}\n',
  'requirementDiagram\nREQUIREMENT a {\nID: x\nTEXT: y\nRISK: LOW\nVERIFYMETHOD: ANALYSIS\n}\n',
  'requirementDiagram\nrequirement a {\nid: first\nid: second\ntext: one\ntext: two\n}\n',
  'requirementDiagram\nrequirement a {\n\n\nid: 1\n\n}\nrequirement a {\nid: 2\n}\n',
  'requirementDiagram\nrequirement a {\n}\nelement a {\ntype: same name\n}\na - satisfies -> a\n',
  // Elements.
  'requirementDiagram\nelement e {\ntype: simulation\ndocref: reqs/e\n}\nelement "f g" {\ntype: "a type"\ndocRef: "http://x.y/z"\n}\n',
  'requirementDiagram\nelement e {\ndocref: only\n}\nelement f {\ntype: only\n}\nelement g {\n}\n',
  'requirementDiagram\nelement e {\ntype: one\ntype: two\n}\n',
  // Every relationship, both ways, and to names that are not there.
  ...RELATIONS.flatMap((r) => [`requirementDiagram\n${BODY}a - ${r} -> b\n`, `requirementDiagram\n${BODY}a <- ${r} - b\n`, `requirementDiagram\n${BODY}b - ${r} -> a\nb - ${r} -> b\n`]),
  `requirementDiagram\n${BODY}a - contains -> nowhere\nnowhere - traces -> b\n"a" - copies -> "b"\n`,
  `requirementDiagram\n${BODY}a - CONTAINS -> b\na <- Traces - b\n`,
  // Classes and styles.
  `requirementDiagram\n${BODY}classDef c1,c2 fill:#f9f,stroke:#333,stroke-width:4px\nclass a,b c1,c2\nstyle a,b fill:#f9f,stroke:#333;stroke-width:4px\n`,
  `requirementDiagram\nrequirement a:::c1 {\n}\nelement b:::c1,c2 {\n}\nclassDef c1 fill:#f9f\nclassDef c2 color:blue\na:::c1,c2\n`,
  `requirementDiagram\n${BODY}classDef c color:red,fill:green,fill-color:x\nclassDef c font-weight:bold\nclass a c\n`,
  `requirementDiagram\n${BODY}classDef c fill-color:fill,color:fillfill\nclass b c\n`,
  `requirementDiagram\n${BODY}style zz fill:red\nstyle a,zz,b stroke:blue\n`,
  `requirementDiagram\n${BODY}class zz,a c\nclassDef c font-size:20px,font-style:italic\n`,
  `requirementDiagram\n${BODY}style a font-size:9px\nstyle b font-size:300px\nstyle b font-weight:700\n`,
  `requirementDiagram\n${BODY}style a fill:#f9f;stroke:#333\nclassDef "q c" fill:red\nclass "a" "q c"\n`,
  `requirementDiagram\n${BODY}classDef c fill:red\nclass a c\nclassDef c stroke:blue\n`,
  `requirementDiagram\n${BODY}style a fill:#f9f,  stroke:#333\n`,
  `requirementDiagram\n${BODY}style a fill:red -x\n`,
  'requirementDiagram\nrequirement a {\n}\nclassDef c ' + 'a:b,'.repeat(2000) + 'a:b\n' + 'class a c\n'.repeat(600),
  'requirementDiagram\nrequirement a {\n}\nclassDef c ' + 'a:b,'.repeat(1000) + 'a:b\n' + 'class a c\n'.repeat(400),
  // Directions.
  ...['TB', 'BT', 'RL', 'LR', 'tb', 'Lr'].map((d) => `requirementDiagram\ndirection ${d}\n${BODY}a - contains -> b\n`),
  `requirementDiagram\ndirection LR direction TB\n${BODY}`,
  `requirementDiagram\nx direction RL y\n${BODY}`,
  `requirementDiagram\ndirection TBX\n${BODY}`,
  `requirementDiagram\ndirection\tBT\n${BODY}`,
  `requirementDiagram\ndirection　LR\n${BODY}`,
  `requirementDiagram\ndirection\nLR\n${BODY}`,
  `requirementDiagram\nrequirement "direction LR" {\n}\n`,
  `requirementDiagram\ndirection LR\r\n${BODY}`,
  `requirementDiagram\ndirection LR ${BODY}`,
  `requirementDiagram\ndirectiondirection TB\n${BODY}`,
  // Markdown and markup in names and text.
  'requirementDiagram\nrequirement "__name__" {\ntext: "*slanted* and **heavy** words"\n}\n',
  'requirementDiagram\nrequirement "**bold** _it_" {\ntext: "a<br>b<br/>c"\nid: "<b>x</b>"\n}\n',
  'requirementDiagram\nelement e {\ntype: "*a* b"\ndocref: "_c_"\n}\n',
  `requirementDiagram\nrequirement a {\ntext: "${'many words in a row '.repeat(12)}"\n}\n`,
  `requirementDiagram\nrequirement ${'longname'.repeat(10)} {\ntext: ${'x'.repeat(80)}\n}\n`,
  // Accessibility and titles.
  'accTitle: t\naccDescr: d\nrequirementDiagram\naccDescr {\n multi\n line\n}\nrequirement "a b":::c1, c2 {\n}\n',
  '---\ntitle: Checkout\n---\nrequirementDiagram\naccTitle: Short name\naccDescr: Longer text\nrequirement a {\n}\n',
  'requirementDiagram\naccTitle:   spaced  \naccDescr {\n  one\n\n   two\n}\n',
  'requirementDiagram\naccDescr {\n  unclosed',
  'requirementDiagram\naccTitle:',
  'requirementDiagram\naccTitle',
  'ACCTITLE: x\nrequirementDiagram\n',
  'requirementDiagram\ntitle My diagram\n',
  'requirementDiagram\ntitle\n',
  'requirementDiagram\ntitle #x\n',
  'requirementDiagram\ntitle　x\n',
  // Comments.
  'requirementDiagram\n# a comment\n% another\nrequirement a {\n# inside\n}\n',
  'requirementDiagram\n%% comment\nrequirement a {\n}\n',
  // Unicode and entities.
  'requirementDiagram\nrequirement 中文 {\ntext: 中文的文字很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长\n}\n',
  'requirementDiagram\nrequirement "😀 face" {\nid: "😀"\n}\n"😀 face" - contains -> "😀 face"\n',
  'requirementDiagram\nrequirement é {\n}\n',
  'requirementDiagram\nrequirement a　b {\n}\n',
  'requirementDiagram\nrequirement "a #35; b #quot;c#quot;" {\ntext: "x &amp; y &lt; z"\n}\n',
  'requirementDiagram\nrequirement a {\ntext: "#9829; and #infin;"\n}\n',
  // Errors of all kinds.
  'requirementDiagram\nrequirement a {\nid: REQ-1\n}\n',
  'requirementDiagram\nrequirement a { \n}\n',
  'requirementDiagram\nrequirement a {\ntext: test the thing\n}\n',
  'requirementDiagram\na - contains - b\n',
  'requirementDiagram\nstyle a stroke-width:1.5px\n',
  'requirement a {\n}\n',
  'requirementDiagram\nrequirement a {\n}\nclassDef c ',
  'requirementDiagram\nrequirement a {\nrisk: severe\n}\n',
  'requirementDiagram\nrequirement a {\nverifymethod: guess\n}\n',
  'requirementDiagram\nelement a {\nid: 1\n}\n',
  'requirementDiagram\nrequirement a {\ntype: 1\n}\n',
  'requirementDiagram\nrequirement {\n}\n',
  'requirementDiagram\nrequirement a\n',
  'requirementDiagram\nrequirement a {\nid: 1',
  'requirementDiagram\na < b\n',
  'requirementDiagram\na - nope -> b\n',
  'requirementDiagram\na -> b\n',
  'requirementDiagram\na\n',
  'requirementDiagram\né\n',
  'requirementDiagram\nstyle a fill:red\tx\n',
  'requirementDiagram\nstyle\n',
  'requirementDiagram\nclass a\n',
  'requirementDiagram x\n',
  'requirementDiagram',
  'requirementDiagram\n"unclosed',
  'requirementDiagram\nrequirement "" {\n}\n',
  '',
  'not a requirement diagram',
];

// A diagram of a few requirements and elements joined at random, for the routes of edges,
// their ends set apart on the sides of boxes, self-loops and markers.
export function generated(rnd) {
  const pick = picker(rnd);
  const kinds = ['requirement', 'functionalRequirement', 'interfaceRequirement', 'performanceRequirement', 'physicalRequirement', 'designConstraint'];
  const words = ['load', 'the system shall respond', 'x', 'within two seconds of a request', '**fast**', '_safe_', 'a<br>b', 'REQ'];
  const relations = ['contains', 'copies', 'derives', 'satisfies', 'verifies', 'refines', 'traces'];
  let s = 'requirementDiagram\n';
  if (rnd() < 0.5) s += `direction ${pick(['TB', 'BT', 'LR', 'RL'])}\n`;
  const names = [];
  const n = 1 + Math.floor(rnd() * 7);
  for (let i = 0; i < n; i++) {
    const name = rnd() < 0.8 ? `n${i}` : `"node ${i} ${pick(words)}"`;
    names.push(name);
    if (rnd() < 0.6) {
      s += `${pick(kinds)} ${name} {\n`;
      if (rnd() < 0.7) s += `id: ${i}.${Math.floor(rnd() * 9)}\n`;
      if (rnd() < 0.7) s += `text: "${pick(words)} ${pick(words)}"\n`;
      if (rnd() < 0.5) s += `risk: ${pick(['low', 'medium', 'high'])}\n`;
      if (rnd() < 0.5) s += `verifymethod: ${pick(['analysis', 'demonstration', 'inspection', 'test'])}\n`;
    } else {
      s += `element ${name} {\n`;
      if (rnd() < 0.6) s += `type: "${pick(words)}"\n`;
      if (rnd() < 0.4) s += `docref: ${pick(['a/b', 'doc', 'http://x.y'])}\n`;
    }
    s += '}\n';
  }
  const m = Math.floor(rnd() * 12);
  for (let k = 0; k < m; k++) {
    const a = pick(names);
    const b = rnd() < 0.1 ? a : pick(names);
    s += rnd() < 0.5 ? `${a} - ${pick(relations)} -> ${b}\n` : `${b} <- ${pick(relations)} - ${a}\n`;
  }
  if (rnd() < 0.2) s += `style ${names[0]} ${pick(['font-size:20px', 'font-weight:bold', 'font-style:italic', 'stroke:red', 'font-size:10px'])}\n`;
  return s;
}

function dumpDb(db) {
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  return {
    direction: db.direction,
    title: opt(db.title),
    accTitle: opt(db.accTitle),
    accDescr: opt(db.accDescr),
    requirements: [...db.requirements.entries()].map(([k, r]) => [
      k,
      {
        name: r.name,
        type: r.type,
        requirementId: r.requirementId,
        text: r.text,
        risk: r.risk,
        verifyMethod: r.verifyMethod,
        cssStyles: r.cssStyles,
        classes: r.classes,
      },
    ]),
    elements: [...db.elements.entries()].map(([k, e]) => [
      k,
      { name: e.name, type: e.type, docRef: e.docRef, cssStyles: e.cssStyles, classes: e.classes },
    ]),
    relations: db.relations.map((r) => ({ type: r.type, src: r.src, dst: r.dst })),
    classes: [...db.classes.entries()].map(([k, c]) => [k, { id: c.id, styles: c.styles, textStyles: c.textStyles }]),
  };
}

function dumpGraph(g) {
  return {
    direction: g.direction,
    nodes: g.nodes.map((n) => ({
      id: n.id,
      requirement: n.requirement === undefined ? undefined : n.requirement.name,
      element: n.element === undefined ? undefined : n.element.name,
      cssStyles: n.cssStyles,
      cssClasses: n.cssClasses,
    })),
    edges: g.edges.map((e) => ({ id: e.id, start: e.start, end: e.end, type: e.type, contains: e.contains })),
  };
}

// The sources of the corpus that are templates in Pele's tests, filled in.
export function corpus(cases) {
  const fill = {
    '${BODY}': [BODY],
    '${p}': PAYLOADS,
    '${type}': RELATIONS,
    '${kind}': RELATIONS,
    '${keyword}': KINDS,
    '${property}': ['__proto__', 'constructor', 'toString'],
    '${dir}': ['TB', 'BT', 'LR', 'RL'],
    '${body}': ['requirement a {\n}\nrequirement b {\n}\nrequirement c {\n}\na - contains -> b\nb - contains -> c\n'],
    '${expectedTitle}': ['Accessible title'],
    '${expectedAccDescription}': ['Accessible description'],
  };
  const out = [];
  for (const s of cases('requirement').extracted) {
    let forms = [s];
    for (const [key, list] of Object.entries(fill)) {
      if (!s.includes(key)) continue;
      forms = forms.flatMap((f) => list.map((v) => f.split(key).join(v)));
    }
    out.push(...forms);
  }
  return out;
}

export default async function ({ src, cases, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { requirement } = await src('diagrams/requirement/index.ts');
  const { buildRequirementGraph } = await src('diagrams/requirement/graph.ts');
  const { RequirementDb } = await src('diagrams/requirement/db.ts');
  const { parseRequirement } = await src('diagrams/requirement/parser.ts');
  const { tokenize } = await src('diagrams/requirement/lexer.ts');
  const { TOKEN_NAMES } = await src('diagrams/requirement/tokens.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const tokens = (source) => {
    const { types, texts, starts } = tokenize(source);
    return types.map((t, k) => [TOKEN_NAMES[t], texts[k], starts[k]]);
  };
  // Text that is no requirement diagram goes to the parser as it is, as Pele's tests give it.
  const direct = (source) => {
    try {
      const db = new RequirementDb();
      parseRequirement(source, db);
      return { tokens: tokens(source), db: dumpDb(db), graph: dumpGraph(buildRequirementGraph(db)) };
    } catch (e) {
      return { tokens: tokens(source), ...caught(e) };
    }
  };

  // Pele's parseSource for a requirement diagram, and the graph its render builds.
  const run = (source) => {
    let pre;
    let type;
    try {
      pre = preprocess(source);
      type = detect(pre.text);
    } catch (e) {
      return caught(e);
    }
    if (type !== 'requirement') return { type, direct: direct(source) };
    const text = encodeEntities(pre.text) + '\n';
    try {
      const db = requirement.parse(text, pre.config, pre.title);
      return { type, tokens: tokens(text), db: dumpDb(db), graph: dumpGraph(buildRequirementGraph(db)) };
    } catch (e) {
      return { type, tokens: tokens(text), ...caught(e) };
    }
  };

  const seeds = corpus(cases);
  const all = [...WRITTEN, ...seeds];
  const next = mutator(all, FRAGMENTS, random(7));
  const fuzz = [];
  while (fuzz.length < 3000) {
    const s = next();
    if (!lone(s) && s.length <= 4000) fuzz.push(s);
  }
  // Inputs assembled from fragments, as Pele's oracle test assembles them.
  const rnd = random(8);
  const assembled = [];
  while (assembled.length < 1500) {
    let s = rnd() < 0.7 ? 'requirementDiagram\n' : '';
    for (let k = 1 + Math.floor(rnd() * 14); k > 0; k--) s += FRAGMENTS[Math.floor(rnd() * FRAGMENTS.length)];
    if (!lone(s)) assembled.push(s);
  }
  const rndGraph = random(9);
  const graphs = Array.from({ length: 1000 }, () => generated(rndGraph));
  const sources = [...new Set([...all, ...fuzz, ...assembled, ...graphs])];
  write('diagram/requirement/testdata/pele/requirement.json', { cases: sources.map((s) => ({ src: s, ...run(s) })) });
}
