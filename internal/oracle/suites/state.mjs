// State diagrams parsed as Pele parses them, from text to tokens, StateDb and the graph
// buildStateGraph makes of it: the sources of Pele's tests and Mermaid's specs and
// documentation, cases written here, and those sources mutated at random.
import { peleError } from '../lib/value.mjs';
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';

// The fragments of Pele's tests/state/oracle.test.ts, and more of what Pele reads beyond the
// grammar.
const FRAGMENTS = [
  ' ', '\n', '\n\n', '\t', '\r\n', '\r', ' ', ' ', ' \n', ';', ':', '::', ':::', ' : ', '-->', '--', '-', '->',
  '[*]', '[*] --> ', ' --> [*]', '{', '}', ' {\n', '\n}\n', '{ ', ' }', '"', '""', '"a b"', '%%', '%%{', '%% c\n', '#', '# c',
  'state ', 'state', 'State ', 'STATE ', 'state "', '" as ', ' as ', 'as ', 'as', 'AS ', 'note ', 'note', 'Note ',
  'note left of ', 'note right of ', 'left of', 'right of ', 'end note', '\nend note', '\n  end note\n', 'End Note',
  'note "', '<<fork>>', '<<join>>', '<<choice>>', '[[fork]]', '[[join]]', '[[choice]]', ' <<FORK>>', '<<', '>>',
  'direction ', 'direction TB', 'direction BT', 'direction RL', 'direction LR', 'Direction lr', 'direction\nTB',
  'direction TD', 'direction', 'TB', 'LR', 'scale ', 'scale 350 width', ' width', 'width', '350', 'accTitle: ',
  'accDescr: ', 'accDescr { ', 'accDescr {\n', 'acctitle : ', 'classDef ', 'classDef c fill:#f00', 'classdef ',
  'class ', 'class a,b c', 'Class ', 'style ', 'style a fill:#f00', 'Style ', 'default', 'default ', 'DEFAULT ',
  'click ', 'click a "u" "t"', 'click a href "u"', 'href ', 'href', 'hide empty description',
  'Hide Empty Description', 'hide empty', 'stateDiagram', 'stateDiagram ', 'stateDiagram-v2', 'stateDiagram-v2\n',
  'STATEDIAGRAM\n', 'a', 'b', 'S1', 'id_1', 'nl', 'é', 'ö', '1', ',', ', ', 'fill:#f9f', 'stroke-width:2px',
  'a:b', 'a::b', 'x%%y', 'x%%{y', ' %%', '.', '*', '[', ']', '<', '>', '/', '\\', '(', ')',
  '😀', '中文', '　', '#quot;', '#35;', '<br>', '**b**', 'font-weight:bold', 'font-size:20px', 'root', 'root_start',
  '---\ntitle: T\n---\n',
];

// What Pele's tests/render/state-security.test.ts puts into its templates, and more.
const PAYLOADS = [
  '__proto__', 'constructor', 'toString', 'hasOwnProperty', '<script>x</script>', 'javascript:alert(1)', '"q"', '😀',
  '中文', '#quot;', 'a b', 'x%%y', 'root', 'divider-id-1',
];

const TEMPLATES = [
  (p) => `stateDiagram-v2\n  ${p} --> b`,
  (p) => `stateDiagram-v2\n  a --> ${p}`,
  (p) => `stateDiagram-v2\n  ${p}`,
  (p) => `stateDiagram-v2\n  a : ${p}`,
  (p) => `stateDiagram-v2\n  a : title\n  a : ${p}\n  a : ${p}`,
  (p) => `stateDiagram-v2\n  a --> b : ${p}`,
  (p) => `stateDiagram-v2\n  a --> a : ${p}`,
  (p) => `stateDiagram-v2\n  state "${p}" as a\n  a --> b`,
  (p) => `stateDiagram-v2\n  state "x" as ${p}\n`,
  (p) => `stateDiagram-v2\n  state "x" as a:${p}\n`,
  (p) => `stateDiagram-v2\n  state ${p} {\n  a --> b\n  }`,
  (p) => `stateDiagram-v2\n  state "${p}" as c {\n  a --> b\n  }\n  c --> c : ${p}`,
  (p) => `stateDiagram-v2\n  state c {\n  a --> b : ${p}\n  --\n  ${p} --> d\n  }\n  c --> a : ${p}`,
  (p) => `stateDiagram-v2\n  state ${p} <<fork>>\n  state ${p} <<join>>\n  [*] --> ${p}`,
  (p) => `stateDiagram-v2\n  state ${p} <<choice>>\n  a --> ${p}`,
  (p) => `stateDiagram-v2\n  a --> b\n  note right of a : ${p}`,
  (p) => `stateDiagram-v2\n  a --> b\n  note left of a\n  ${p}\n  ${p}\n  end note`,
  (p) => `stateDiagram-v2\n  note right of ${p} : text`,
  (p) => `stateDiagram-v2\n  a --> b\n  note "${p}" as ${p}`,
  (p) => `stateDiagram-v2\n  a:::${p} --> b`,
  (p) => `stateDiagram-v2\n  a --> b\n  class a,${p} c`,
  (p) => `stateDiagram-v2\n  a --> b\n  classDef c ${p}:red,color:${p}\n  class a c`,
  (p) => `stateDiagram-v2\n  a --> b\n  style a,${p} color:${p},stroke:${p},font-size:${p}`,
  (p) => `stateDiagram-v2\n  a --> b\n  click a "${p}" "${p}"\n`,
  (p) => `stateDiagram-v2\n  ${p} --> b\n  click ${p} "https://example.com" "${p}"\n`,
  (p) => `stateDiagram-v2\n  accTitle: ${p}\n  accDescr: ${p}\n  a --> b`,
  (p) => `stateDiagram-v2\n  direction LR ${p}\n  a --> b`,
];

export const WRITTEN = [
  'stateDiagram-v2\n[*] --> A\nA --> B : go\nB --> [*]',
  'stateDiagram\n[*] --> A\nA --> [*]',
  'stateDiagram-v2\nstate A {\n  [*] --> a1\n  a1 --> [*]\n}\n[*] --> A\nA --> [*]',
  'stateDiagram-v2\nstate A {\n  state B {\n    state C {\n      x --> y\n    }\n  }\n}\nA --> C\nx --> A',
  'stateDiagram-v2\nstate A {\n  a --> b\n  --\n  c --> d\n  --\n  e\n}',
  'stateDiagram-v2\nstate A {\n  a --> b\n  --\n}',
  'stateDiagram-v2\nstate A {\n  --\n  --\n  a\n}',
  'stateDiagram-v2\nstate A {\n  direction LR\n  a --> b\n  --\n  direction BT\n  c --> d\n}',
  'stateDiagram-v2\nstate f <<fork>>\nstate j <<join>>\nstate c <<choice>>\n[*] --> f\nf --> a\nf --> b\na --> j\nb --> j\nj --> c\nc --> x : yes\nc --> y : no',
  'stateDiagram-v2\nstate f [[fork]]\nstate j [[join]]\nstate c [[choice]]\nf --> j\nj --> c',
  'stateDiagram-v2\nstate "long <<fork>> name" as x\nstate a b <<fork>>\n',
  'stateDiagram-v2\nstate x <<fork>> <<join>>\nstate y [[choice]] <<fork>>',
  'stateDiagram-v2\na --> b\nnote right of a : a note\nnote left of b\n  line one\n  line two\nend note',
  'stateDiagram-v2\nnote left of a : one\nnote right of a : two\na --> b',
  'stateDiagram-v2\nstate A {\n  x\n  note right of x : inside\n}\nnote left of A : outside',
  'stateDiagram-v2\nnote "floating" as N\na --> N',
  'stateDiagram-v2\nnote right of a:no space',
  'stateDiagram-v2\nnote right of a :x',
  'stateDiagram-v2\nnote right of a :😀x',
  'stateDiagram-v2\nnote right of a\nend note',
  'stateDiagram-v2\nnote right of a\n  text\n',
  'stateDiagram-v2\nclassDef bad fill:#f00,color:white,font-weight:bold\nclassDef good fill:#0f0;\nclass a bad\nclass b,c good\na --> b\nb --> c\nstyle c stroke:#00f,stroke-width:3px',
  'stateDiagram-v2\na:::bad --> b:::good\nclassDef bad fill:red\nclassDef good fill:green',
  'stateDiagram-v2\n[*]:::x --> a\nclassDef x fill:red',
  'stateDiagram-v2\nclassDef default fill:red\na --> b',
  'stateDiagram-v2\nclassDef a\nclassDef b fill:red\nclassDef b stroke:blue',
  'stateDiagram-v2\nstate A {\n  a\n}\nclass A cls\nclassDef cls fill:red\nstyle A stroke:blue',
  'stateDiagram-v2\nstyle a,b fill:red;stroke:blue;\nstyle a color:green',
  'stateDiagram-v2\nclass a , b c',
  'stateDiagram-v2\ndirection LR\na --> b',
  'stateDiagram-v2\ndirection TB\ndirection RL\na --> b',
  'stateDiagram-v2\na --> b\ndirection BT and more',
  'stateDiagram-v2\na --> b %% direction LR',
  'stateDiagram-v2\naccTitle: The title\naccDescr: A description\na --> b',
  'stateDiagram-v2\naccDescr {\n  line one\n    line two\n}\na --> b',
  'stateDiagram-v2\naccTitle:\na',
  '---\ntitle: My states\n---\nstateDiagram-v2\na --> b',
  '---\ntitle: A title much wider than the diagram under it\n---\nstateDiagram-v2\na',
  'stateDiagram-v2\nclick a "https://example.com" "tip"\na --> b',
  'stateDiagram-v2\nclick a href "https://example.com"\nclick b href "javascript:alert(1)"\na --> b',
  'stateDiagram-v2\na --> b\nclick a "#35;" "#quot;t#quot;"\n',
  'stateDiagram-v2\na --> b\nclick a "u"',
  'stateDiagram-v2\na --> b\nclick a',
  'stateDiagram-v2\nhide empty description\nscale 350 width\na',
  'stateDiagram-v2\nscale 350',
  'stateDiagram-v2\nscale x width',
  'stateDiagram-v2\nstate "A description" as a\nstate "B" as b {\n  x\n}\na --> b',
  'stateDiagram-v2\nstate "d" as a:extra\nstate "d" as b:one:two',
  'stateDiagram-v2\nstate " spaced " as a:extra\nstate " b " as b\na --> b',
  'stateDiagram-v2\nclassDef a fill:red;stroke:blue;,color:green;;\nclass x a',
  'stateDiagram-v2\na : first\na : second\na : third\na --> b',
  'stateDiagram-v2\na : first\nstate a {\n  x\n}',
  'stateDiagram-v2\nstate a {\n  x\n}\na : first\na : second',
  'stateDiagram-v2\na:b\na::c',
  'stateDiagram-v2\na --> b: label with: colons\na --> c :: x',
  'stateDiagram-v2\na --> a\na --> a : twice\na --> a : thrice',
  'stateDiagram-v2\nstate A {\n  x --> y\n}\nA --> A : loop\nA --> A',
  'stateDiagram-v2\ndirection LR\na --> a : loop\nstate B {\n  b --> b\n}',
  'stateDiagram-v2\nstate A {\n  [*] --> x\n}\nA --> x\nx --> A',
  'stateDiagram-v2\nstate A {\n  state B {\n    x\n  }\n}\nA --> x\nx --> A\nB --> x',
  'stateDiagram-v2\ndirection LR\nstate A {\n  direction TB\n  x --> y\n}\nA --> y\nz --> A',
  'stateDiagram-v2\nstate A {\n  state B {\n    a\n  }\n}\nstate B {\n  state A {\n    b\n  }\n}',
  'stateDiagram-v2\nstate a {\n  x\n}\nstate b {\n  a\n}\nstate a {\n  b\n}',
  'stateDiagram-v2\nstate nl\nstate x\nstate y {\n}',
  'stateDiagram-v2\nstate "x" as {\n  a\n}',
  'stateDiagram-v2\nstate one two {\n  a\n}',
  'stateDiagram-v2\nstate one two\n{\n  a\n}',
  'stateDiagram-v2\nstate root {\n  a\n}',
  'stateDiagram-v2\nroot --> a\n[*] --> root_start',
  'stateDiagram-v2\na%%b --> c\n%%x --> y',
  'stateDiagram-v2\nstate "x" as %%y\nstate "z" as w%%v',
  'stateDiagram-v2\nnote right of a%%b : t',
  'stateDiagram-v2\nnote "t" as %%n\n',
  'stateDiagram-v2\na --> b\n}',
  'stateDiagram-v2\nstate a {\nb',
  'stateDiagram-v2\nstate a {\n  ::: \n}',
  'stateDiagram-v2\n:',
  'stateDiagram-v2\na -',
  'stateDiagram-v2\n-- a',
  'stateDiagram-v2\na --> ',
  'stateDiagram-v2\nclassDef default fill:red',
  'stateDiagram-v2\nclass',
  'stateDiagram-v2\nstyle',
  'stateDiagram-v2\nnote x',
  'stateDiagram-v2\nnote left of',
  'stateDiagram-v2\nnote "x"',
  'stateDiagram-v2\nstate',
  'stateDiagram-v2\nstate "unclosed',
  'stateDiagram-v2\nstate "x"',
  'stateDiagram-v2\nstate a {\n  x : y\n}\na : z',
  'stateDiagram-v2\na --> b\n  accDescr {\n unclosed',
  'stateDiagram-v2\nstate a {\n  click a "u"\n}',
  'stateDiagram-v2\nstate a{\n  b\n}\nstate c{d}',
  'stateDiagram-v2\nstate a\n{\n}',
  'stateDiagram-v2\n[*] --> [*]\n[*]',
  'stateDiagram-v2\nA --> B\nB --> C\nC --> A\nA --> C\nC --> B\nB --> A',
  'stateDiagram-v2\nstate "中文 状态" as z\nz --> é\né --> 😀\n😀 : 描述',
  'stateDiagram-v2\na　-->　b',
  'stateDiagram-v2\na --> b',
  'stateDiagram-v2\na --> b : #quot;quoted#quot; #35; x',
  'stateDiagram-v2\na --> b : <b>bold</b> **md** _it_',
  'stateDiagram-v2\na --> b : a very long label for a transition that should wrap past the wrapping width',
  'stateDiagram-v2\nstate "a very long description of a state that will wrap past two hundred pixels for sure" as a',
  'stateDiagram-v2\na : line\na : \na : <br>two<br/>lines',
  '%%{init: {"state": {"nodeSpacing": 100, "rankSpacing": 10, "wrappingWidth": 60}}}%%\nstateDiagram-v2\na --> b : some label here\nb --> c\na --> c',
  '---\nconfig:\n  state:\n    nodeSpacing: 5000\n    wrappingWidth: -1\n---\nstateDiagram-v2\na --> b\na --> c',
  'stateDiagram-v2\nstate fork_state <<fork>>\n[*] --> fork_state\nfork_state --> A\nfork_state --> B\nfork_state --> C\nfork_state --> D\nfork_state --> E\nfork_state --> F\nfork_state --> G\nfork_state --> H\nfork_state --> I',
  'stateDiagram-v2\ndirection LR\nstate f <<fork>>\n[*] --> f\nf --> a\nf --> b\nf --> c',
  'stateDiagram-v2\nstate A {\n  direction LR\n  state f <<join>>\n  a --> f\n  b --> f\n  f --> c\n}',
  'stateDiagram-v2\nstate A {\n  a\n}\nstate B {\n  b\n}\nA --> B\nnote right of A : n\nnote left of B : m',
  'stateDiagram-v2\nstate A {\n  x\n  note right of x : n\n  x --> x : self\n}',
  'stateDiagram-v2\nstate A {\n  a\n}\nA --> a : in\na --> A : out\nA --> A',
  'stateDiagram-v2\ndirection RL\nstate A {\n  direction BT\n  a --> b\n}\nA --> a\nb --> A\nc --> A',
  'stateDiagram-v2\na --> b\nstate a {\n  x\n}\nb --> x',
  'stateDiagram-v2\nstate A {\n  b --> c\n}\nstate D {\n  e\n  --\n  f\n}\nA --> D\nc --> e\nf --> b',
  'stateDiagram-v2\na --> x\na --> y\nb --> y\nstate C {\n  x\n  y\n}',
  'stateDiagram-v2\nstate C {\n  direction LR\n  x --> y\n}\nC --> x\ny --> C\nz --> C',
  'stateDiagram-v2\nstate C {\n  direction RL\n  x --> y\n}\nC --> x\ny --> C',
  'stateDiagram-v2\ndirection LR\nstate C {\n  direction RL\n  x\n}\nC --> x\nx --> C',
  'stateDiagram-v2\ndirection LR\nstate C {\n  direction LR\n  x --> y\n}\nC --> y\nx --> C\nw --> C',
  'stateDiagram-v2\nstate C {\n  direction BT\n  x --> y\n}\nC --> y\nx --> C',
  'stateDiagram-v2\nstate Wide_composite_title {\n  x\n  y\n  z\n}\na --> x\nb --> y\nc --> z\nd --> x',
  'stateDiagram-v2\nstate T {\n  x\n}\na --> x\nb --> x\nc --> x\nd --> x\ne --> x',
  'stateDiagram-v2\nstate "A long title for this composite" as T {\n  x\n  y\n}\na --> x\nb --> y\na --> y',
  'stateDiagram-v2\nstate A {\n  state B {\n    x\n  }\n  y\n}\nq --> x\nq --> y\nr --> x',
  'stateDiagram-v2\nstate A {\n  b\n  --\n  c\n  --\n  d\n  b --> c\n}',
  'stateDiagram-v2\nstate A {\n  state B {\n    x\n    --\n    y\n  }\n  --\n  z\n}',
  'stateDiagram-v2\nstate x <<choice>>\nx --> x\nx --> y : <b>b</b>',
  'stateDiagram-v2\nstate f <<fork>>\nf --> f',
  'stateDiagram-v2\n[*] --> [*] : odd',
  'stateDiagram-v2\na --> b\nb --> a\na --> b\nb --> a : back',
  'stateDiagram-v2\na:::x\nclassDef x font-size:30px,font-style:italic,font-weight:bold\na --> b',
  'stateDiagram-v2\nstate A {\n  a\n}\nclassDef big font-size:40px\nclass A big',
  'stateDiagram-v2\nstate A {\n  a\n}\nclassDef c fill:red\nclass A c\nstate B {\n  b\n}\nstyle B fill:blue',
  'stateDiagram-v2\na --> b\nclick a "https://example.com/a?b=c&d=e" "t"\nclick b "mailto:x@y.z"\nclick c "file:///etc/passwd" "x"',
  'stateDiagram-v2\nstate A {\n  a\n}\nclick A "https://example.com" "group link"',
  'stateDiagram-v2\na --> b\n    note right of a\n        indented\n    end note\n',
  'stateDiagram-v2\nnote left of a : left\nnote left of a : again\na --> b',
  'stateDiagram-v2\ndirection LR\na --> b\nnote right of a : across\nnote left of b : back',
  'stateDiagram-v2\na\nb\nc\nd',
  'stateDiagram-v2\ns1 --> s2 --> s3',
  'stateDiagram-v2\n\n\n  \n',
  'stateDiagram-v2',
  'stateDiagram',
  'stateDiagram-v2 \na',
  'stateDiagram-v3\na',
  'stateDiagramx\na',
  '',
  'state',
  'not a state diagram',
];

function dumpStmt(s) {
  if (typeof s === 'string') return s;
  switch (s.stmt) {
    case 'state':
      return {
        stmt: 'state',
        id: s.id,
        type: s.type,
        description: s.description,
        doc: s.doc === undefined ? undefined : s.doc.map(dumpStmt),
        note: s.note,
        start: s.start,
        classes: s.classes,
      };
    case 'relation':
      return { stmt: 'relation', state1: dumpStmt(s.state1), state2: dumpStmt(s.state2), description: s.description };
    case 'click':
      return { stmt: 'click', id: dumpStmt(s.id), url: s.url, tooltip: s.tooltip };
    default:
      return { ...s };
  }
}

function dumpDb(db) {
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  return {
    direction: db.direction,
    title: opt(db.title),
    accTitle: opt(db.accTitle),
    accDescr: opt(db.accDescr),
    rootDoc: db.rootDoc.map(dumpStmt),
    states: [...db.states.values()].map((s) => ({
      id: s.id,
      type: s.type,
      descriptions: s.descriptions,
      doc: s.doc === undefined ? undefined : s.doc.length,
      note: s.note,
      classes: s.classes,
      styles: s.styles,
      textStyles: s.textStyles,
    })),
    relations: db.relations,
    classes: [...db.classes.values()],
    links: [...db.links.entries()].map(([id, l]) => [id, l.url, l.tooltip]),
    graph: {
      nodes: db.graph.nodes.map((n) => ({
        id: n.id,
        shape: n.shape,
        label: n.label,
        description: n.description,
        cssClasses: n.cssClasses,
        cssCompiledStyles: n.cssCompiledStyles,
        cssStyles: n.cssStyles,
        type: n.type,
        isGroup: n.isGroup,
        parentId: n.parentId,
        dir: n.dir,
        position: n.position,
        noteFor: n.noteFor,
        colorIndex: n.colorIndex,
      })),
      edges: db.graph.edges.map((e) => ({ ...e })),
    },
  };
}

export function stateCorpus(cases, detect, preprocess) {
  return cases('state').extracted.filter((s) => {
    try {
      return detect(preprocess(s).text) === 'state';
    } catch {
      return false;
    }
  });
}

export default async function ({ src, cases, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { state } = await src('diagrams/state/index.ts');
  const { StateDb } = await src('diagrams/state/db.ts');
  const { parseState } = await src('diagrams/state/parser.ts');
  const { tokenize } = await src('diagrams/state/lexer.ts');
  const { TOKEN_NAMES } = await src('diagrams/state/tokens.ts');

  const caught = (e) => {
    if (e instanceof RangeError) return { error: { code: 'limit', message: 'Diagram is nested too deeply.' } };
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const tokens = (source) => {
    const t = tokenize(source);
    return { tokens: t.types.map((type, i) => [TOKEN_NAMES[type], t.texts[i], t.starts[i]]), lexError: t.error };
  };
  // Text that is no state diagram goes to the parser as it is, as Pele's tests give it.
  const direct = (source) => {
    try {
      const db = new StateDb();
      parseState(source, db);
      return { ...tokens(source), db: dumpDb(db) };
    } catch (e) {
      return { ...tokens(source), ...caught(e) };
    }
  };

  // Pele's parseSource for a state diagram.
  const run = (source) => {
    let pre;
    let type;
    try {
      pre = preprocess(source);
      type = detect(pre.text);
    } catch (e) {
      return caught(e);
    }
    if (type !== 'state') return { type, direct: direct(source) };
    const text = encodeEntities(pre.text) + '\n';
    try {
      const db = state.parse(text, pre.config, pre.title, pre.lineOffset, {});
      return { type, ...tokens(text), db: dumpDb(db) };
    } catch (e) {
      return { type, ...tokens(text), ...caught(e) };
    }
  };

  const corpus = stateCorpus(cases, detect, preprocess);
  const templated = [];
  for (const t of TEMPLATES) for (const p of PAYLOADS) templated.push(t(p));
  const next = mutator([...corpus, ...WRITTEN.filter((s) => s.startsWith('stateDiagram'))], FRAGMENTS, random(7));
  const fuzz = [];
  while (fuzz.length < 3000) {
    const s = next();
    if (!lone(s) && s.length <= 4000) fuzz.push(s);
  }
  const all = [...new Set([...WRITTEN, ...cases('state').extracted, ...templated, ...fuzz])].filter((s) => !lone(s));
  write('diagram/state/testdata/pele/state.json', { states: all.map((s) => ({ src: s, ...run(s) })) });
}
