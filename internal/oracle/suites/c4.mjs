// C4 diagrams parsed as Pele parses them: the tokens Pele's c4/lexer.ts makes of the text, the
// model parseC4 builds of the text given to it as it is, and the model of the text as it comes
// out of Pele's parseSource, with front matter, directives and entity codes read first. The
// cases are Mermaid's documentation and specs, the inputs of Pele's own tests, cases written
// here, random diagrams, and all of those mutated at random.
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { enc, lone, mutator, peleError } from './parse.mjs';

const KEYWORD = /C4(?:Context|Container|Component|Dynamic|Deployment)/;

// The fragments of Pele's tests/c4/oracle.test.ts, and more.
export const FRAGMENTS = [
  ' ', '  ', '\n', '\r\n', '\t', ',', ',,', ', ', '(', ')', '( ,', '(,', '()', '{', '}', ' {', '{\n', '\n{', '}\n', '"', '""',
  ' ""', ' "', '$', ' $', '=', '="', '= "', '$a="b"', '$link="x"', '$tags="t"', '$offsetX="5"', '$=""', '$x=', '#', ';',
  '%%', '%% c\n', '%%{', 'a', 'b', 'x1', '"l"', 'é', 'title ', 'title\n', 'title x', 'accDescription ', 'accDescription d',
  'accTitle: ', 'accTitle', 'accDescr: ', 'accDescr', 'accDescr {', 'accDescr { d }', 'direction', 'direction TB',
  'direction  BT', 'direction\nRL', 'direction LR', 'C4Context', 'C4Container', 'C4Component', 'C4Dynamic',
  'C4Deployment', 'Person', 'Person_Ext', 'Person(', 'System', 'SystemDb', 'SystemQueue', 'System_Ext', 'SystemDb_Ext',
  'SystemQueue_Ext', 'Container', 'ContainerDb', 'ContainerQueue', 'Container_Ext', 'ContainerDb_Ext',
  'ContainerQueue_Ext', 'Component', 'ComponentDb', 'ComponentQueue', 'Component_Ext', 'ComponentDb_Ext',
  'ComponentQueue_Ext', 'Boundary', 'Boundary(b, "B") {\n', 'Enterprise_Boundary', 'System_Boundary',
  'Container_Boundary', 'Deployment_Node', 'Node', 'Node_L', 'Node_R', 'Node(n, "N")\n{', 'Rel', 'Rel(a, b, "l")',
  'BiRel', 'Rel_Up', 'Rel_U', 'Rel_Down', 'Rel_D', 'Rel_Left', 'Rel_L', 'Rel_Right', 'Rel_R', 'Rel_Back', 'RelIndex',
  'RelIndex(1, a, b, "l")', 'UpdateElementStyle', 'UpdateRelStyle', 'UpdateLayoutConfig', 'UpdateLayoutConfig("3", "1")',
  '_', 'Ext', '__proto__', '$__proto__="x"',
  ' ', ' ', '　', '﻿', '😀', '中文', '#35;', '&amp;', '<br>', '<br/>', '<b>b</b>', '\\n', 'fa:fa-car',
  '$label="L"', '$descr="d"', '$techn="t"', '$sprite="s"', '$shape="cylinder"', '$bgColor="red"', '$fontColor="blue"',
  '$borderColor="green"', '$offsetY="-10"', '$type="T"', '$textColor="red"', '$lineColor="red"', '$c4ShapeInRow="2"',
  '"https://example.com"', '$link="https://example.com/a"', '$link="javascript:alert(1)"', '"0x10"', '"-5"', '"1e3"',
  '\n---\ntitle: T\n---\n', '---\n', '%%{init: {"c4": {"diagramMarginX": 10}}}%%\n',
];

// The inputs of Pele's own tests beyond its corpus, and the edges of the grammar.
const SEEDS = [
  'C4Context',
  'C4Context\n',
  'C4Context\nPerson(a, "A")',
  'C4Context Person(a, "A")',
  'C4Context\n\nPerson(a, "A")\n\n',
  'C4Container\ntitle T\nContainer(c, "C", "Go", "Does things")',
  'C4Component\nComponent(c, "C", "Go", "Does things", $sprite="x", $tags="t", $link="l")',
  'C4Dynamic\nPerson(a, "A")\nSystem(b, "B")\nRel(a, b, "One")\nRel(b, a, "Two")\nRelIndex(9, a, b, "Again")',
  'C4Deployment\nDeployment_Node(n, "N", "Ubuntu", "d") {\nContainer(c, "C")\n}',
  'C4Deployment\nNode(n, "N")\n{\nNode_L(l, "L")\n{\nContainer(c, "C")\n}\nNode_R(r, "R") {\nContainer(d, "D")\n}\n}',
  'C4Context\nPerson(a)',
  'C4Context\nPerson()',
  'C4Context\nPerson(,)',
  'C4Context\nPerson(,,)',
  'C4Context\nPerson( , "A")',
  'C4Context\nPerson(a, "")',
  'C4Context\nPerson(a, "A"',
  'C4Context\nPerson(a, "A',
  'C4Context\nPerson(a, $x',
  'C4Context\nPerson(a, $x=',
  'C4Context\nPerson(a, $x="',
  'C4Context\nPerson(a, $x = "y")',
  'C4Context\nPerson(a, $x= "y")',
  'C4Context\nPerson(a, $x ="y")',
  'C4Context\nPerson($alias="x", "A")',
  'C4Context\nPerson(a, $label="L")',
  'C4Context\nPerson(a, $descr="D")',
  'C4Context\nPerson(a, "A", $descr="D", "pos")',
  'C4Context\nPerson(a, "A", "d")\nPerson(a, "B")',
  'C4Context\nPerson(a, "A", "d")\nSystem(a, "B", "e")',
  'C4Context\nPerson(a, unquoted label)',
  'C4Context\nPerson(a, unquoted, label)',
  'C4Context\nPerson(a, "A") Person(b, "B")',
  'C4Context\nPerson(a, "A");Person(b, "B")',
  'C4Context\nPerson (a, "A")',
  'C4Context\nPerson\n(a, "A")',
  'C4Context\nperson(a, "A")',
  'C4Context\nPerson_ext(a, "A")',
  'C4Context\nBoundary(b, "B") {\n}',
  'C4Context\nBoundary(b, "B") {\nPerson(a, "A")\n}',
  'C4Context\nBoundary(b, "B")\n{\nPerson(a, "A")\n}',
  'C4Context\nBoundary(b, "B") {Person(a, "A")}',
  'C4Context\nBoundary(b, "B")',
  'C4Context\nBoundary(b, "B") {',
  'C4Context\nBoundary(b, "B") {\nPerson(a, "A")',
  'C4Context\nBoundary(b, "B") {\nPerson(a, "A")\n}\n}',
  'C4Context\nBoundary(b, "B", "kind") {\nPerson(a, "A")\n}',
  'C4Context\nEnterprise_Boundary(b) {\nPerson(a, "A")\n}',
  'C4Context\nSystem_Boundary(b, "B", "x", "y", "z") {\nPerson(a, "A")\n}',
  'C4Context\nBoundary(b, "B") {\nBoundary(b, "C") {\nPerson(a, "A")\n}\n}',
  'C4Context\nBoundary(global, "G") {\nPerson(a, "A")\n}',
  'C4Context\nBoundary($x="y", "B") {\nPerson(a, "A")\n}',
  'C4Context\ntitle T\nPerson(a, "A")',
  'C4Context\ntitle\nPerson(a, "A")',
  'C4Context\ntitle T Person(a, "A")',
  'C4Context\ntitle T # x\nPerson(a, "A")',
  'C4Context\ntitle T; Person(a, "A")',
  'C4Context\ntitle\tT\nPerson(a, "A")',
  'C4Context\ntitle T\nPerson(a, "A")',
  'C4Context\naccTitle: T\nPerson(a, "A")',
  'C4Context\naccTitle : T\nPerson(a, "A")',
  'C4Context\naccTitle\nPerson(a, "A")',
  'C4Context\naccDescr: D\nPerson(a, "A")',
  'C4Context\naccDescr {\n  one\n  two\n}\nPerson(a, "A")',
  'C4Context\naccDescr{x}Person(a, "A")',
  'C4Context\naccDescription D\nPerson(a, "A")',
  'C4Context\naccDescr: D\naccDescr {\n e \n}\nPerson(a, "A")',
  'C4Context\ndirection TB\nPerson(a, "A")',
  'C4Context\ndirection LR\nPerson(a, "A")',
  'C4Context\nx direction RL y\nPerson(a, "A")',
  'C4Context\ndirection direction\nBT\nPerson(a, "A")',
  'C4Context\ndirection XX\nPerson(a, "A")',
  'C4Context\nPerson(a, "direction TB")',
  'direction TB\nC4Context\nPerson(a, "A")',
  'C4Context direction TB\nPerson(a, "A")',
  'C4Context\n%% c\nPerson(a, "A")',
  'C4Context\nPerson(a, "A") %% c\nPerson(b, "B")',
  'C4Context\n%%{init: {}}%%\nPerson(a, "A")',
  'C4Context\nPerson(a, "A")\nPerson(b, "B")\nRel(a, b, "x")',
  'C4Context\nRel(a, b)',
  'C4Context\nRel(a, b, "x")',
  'C4Context\nPerson(a, "A")\nRel(a, a, "self")',
  'C4Context\nPerson(a, "A")\nPerson(b, "B")\nRel(a, b, "x")\nRel(a, b, "y", "t")',
  'C4Context\nPerson(a, "A")\nPerson(b, "B")\nRel($x="a", b, "x")',
  'C4Context\nPerson(a, "A")\nPerson(b, "B")\nRel(a, b, $label="L", $techn="T")',
  'C4Context\nPerson(a, "A")\nPerson(b, "B")\nRel(a, b, "x", $offsetX="5")',
  'C4Context\nPerson(a, "A")\nPerson(b, "B")\nRel(a, b, "x")\nUpdateRelStyle(a, b, "red", "blue", "10", "-20")',
  'C4Context\nPerson(a, "A")\nPerson(b, "B")\nRel(a, b, "x")\nUpdateRelStyle(a, b, $offsetY="0x10", $offsetX="abc")',
  'C4Context\nPerson(a, "A")\nPerson(b, "B")\nRel(a, b, "x")\nUpdateRelStyle(a, b, , , $offsetY="7", $offsetX="3")',
  'C4Context\nPerson(a, "A")\nUpdateElementStyle(a, "red", "blue", "green", "true", "cylinder")',
  'C4Context\nPerson(a, "A")\nUpdateElementStyle(a, $shape="queue", $techn="T", $legendText="L")',
  'C4Context\nPerson(a, "A")\nUpdateElementStyle(x, "red")',
  'C4Context\nPerson(a, "A")\nUpdateElementStyle($a="b", "red")',
  'C4Context\nUpdateLayoutConfig("2", "1")\nPerson(a, "A")',
  'C4Context\nUpdateLayoutConfig($c4ShapeInRow="2", $c4BoundaryInRow="1")\nPerson(a, "A")',
  'C4Context\nUpdateLayoutConfig($c4BoundaryInRow="1")\nPerson(a, "A")',
  'C4Context\nUpdateLayoutConfig("0", "-1")\nPerson(a, "A")',
  'C4Context\nUpdateLayoutConfig("x")\nPerson(a, "A")',
  'C4Context\nUpdateLayoutConfig("99999999999999999999999")\nPerson(a, "A")',
  'C4Context\nUpdateLayoutConfig("1.5", " 3")\nPerson(a, "A")',
  'C4Context\nPerson(a, "A", $tags="t1,t2", $link="https://x")',
  'C4Context\nPerson(a, "A", "d", $k="v", $k="w", $j="u")',
  'C4Context\nPerson(a, "A", $__proto__="x", $constructor="y")',
  'C4Context\nPerson(a, "A", $parent="p", $alias="q", $kind="k")',
  'C4Context\nPerson(a, "A")\n\n\n}',
  'C4Context\n{',
  'C4Context\n}',
  'C4Context\n  \n  Person(a, "A")\n  ',
  'C4Context\r\nPerson(a, "A")\r\nPerson(b, "B")',
  'C4Context\rPerson(a, "A")',
  'C4Context Person(a, "A")',
  '  C4Context\n  Person(a, "A")',
  'C4Contextx\nPerson(a, "A")',
  'C4Context2\nPerson(a, "A")',
  'C4Containers\nPerson(a, "A")',
  'flowchart\nC4Container',
  'graph TD\nA-->B %% C4Dynamic',
  'x C4Deployment\nPerson(a, "A")',
  'C4Context\nPerson(a, "#35; &amp; <b>b</b> ﬂ°°lt¶ß")',
  'C4Context\nPerson(a, "😀 中文")',
  '---\ntitle: Front\n---\nC4Context\nPerson(a, "A")',
  '---\ntitle: Front\n---\nC4Context\ntitle Own\nPerson(a, "A")',
  '---\nconfig:\n  c4:\n    useMaxWidth: false\n---\nC4Context\nPerson(a, "A")',
  '%%{init: {"c4": {"c4ShapeInRow": 2}}}%%\nC4Context\nPerson(a, "A")\nPerson(b, "B")\nPerson(c, "C")',
  '',
  'C4',
];

// Statements and the separators between them, to build documents that reach deep into the
// grammar, as Pele's oracle test builds them.
const STATEMENTS = [
  'title T', 'accTitle: T', 'accDescr: D', 'accDescr {\n d\n}', 'accDescription D', 'Person(a, "A")', 'Person_Ext(a, "A", "d")',
  'System(b, "B", $link="l")', 'SystemDb(c, , "d")', 'Container(d, "D", "t", "d", $tags="x")', 'Component(e,"E")',
  'Rel(a, b, "x")', 'BiRel(a, b, "x", "t")', 'RelIndex(1, a, b, "x")', 'Rel_Back(a,b,"x")', 'UpdateLayoutConfig("2")',
  'UpdateElementStyle(a, $bgColor="red")', 'UpdateRelStyle(a, b, "red", $offsetX="1")', 'Person(a, unquoted)', 'Person(a)',
  'Boundary(g, "G") {', 'Enterprise_Boundary(g, "G")\n{', 'System_Boundary(g, "G")\n  {\n', 'Container_Boundary(g, "G"){',
  'Deployment_Node(n, "N") {', 'Node_L(n, "N", "t")\n{', 'Node_R(n, "N") {', 'Boundary(g, "G")', '{', '}', '}', '}',
];
const SEPARATORS = ['\n', '\n', '\n', '\n\n', '\n  ', ' \n', ' ', '', '\r\n', '\n%% c\n', ' %% c\n', ';'];
export const HEADERS = ['C4Context', 'C4Container', 'C4Component', 'C4Dynamic', 'C4Deployment'];

export function statementDiagrams(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  return () => {
    let src = pick(HEADERS) + pick(SEPARATORS);
    const count = 1 + Math.floor(rnd() * 8);
    for (let i = 0; i < count; i++) src += pick(STATEMENTS) + pick(SEPARATORS);
    return src;
  };
}

const SHAPES = [
  'Person', 'Person_Ext', 'System', 'SystemDb', 'SystemQueue', 'System_Ext', 'SystemDb_Ext', 'SystemQueue_Ext', 'Container',
  'ContainerDb', 'ContainerQueue', 'Container_Ext', 'ContainerDb_Ext', 'ContainerQueue_Ext', 'Component', 'ComponentDb',
  'ComponentQueue', 'Component_Ext', 'ComponentDb_Ext', 'ComponentQueue_Ext',
];
const BOUNDARIES = ['Boundary', 'Enterprise_Boundary', 'System_Boundary', 'Container_Boundary', 'Deployment_Node', 'Node', 'Node_L', 'Node_R'];
const RELS = ['Rel', 'BiRel', 'Rel_Up', 'Rel_U', 'Rel_Down', 'Rel_D', 'Rel_Left', 'Rel_L', 'Rel_Right', 'Rel_R', 'Rel_Back', 'RelIndex'];
const LABELS = [
  'A', 'User', 'Banking Customer', 'Internet Banking System', 'Mainframe Banking System', 'E-mail system', 'API', 'Web App',
  'Database', 'Queue', 'x', 'Single Page Application', 'A label long enough that it has to wrap over more than one line',
  'Wordwithoutanyspacesthatisfartoolongtofitinaboxofanywidth', '中文标签', '😀 emoji', '#35; &amp; more', 'a <br> b',
  '<b>bold</b> and <i>it</i>', 'line\\nbreak', '', ' spaced ', 'Ünïcödé', 'fa:fa-car Car', 'tab\there', 'a,b',
];
const DESCRS = [
  'Uses the bank', 'Allows customers to view information about their bank accounts, and make payments.', 'd', '',
  'Stores all of the core banking information about customers, accounts, transactions, etc.',
  'A system of the bank, with personal bank accounts. next line.', 'two <br/> lines', '中文描述很长很长很长很长很长很长很长很长',
];
const TECHNS = ['HTTPS', 'JSON/HTTPS', 'Java, Spring MVC', 'SQL', '', 'gRPC', 'Kafka topic', 'Rust and a very long name of a technology'];
const COLORS = ['red', 'blue', '#f00', '#1168bd', 'rgb(0, 128, 0)', 'grey', 'transparent', 'none', 'hsl(120, 50%, 50%)', 'notacolor', 'url(x)', '#12345678'];
const LINKS = ['https://example.com', 'https://example.com/a?b=c#d', '/relative', 'mailto:a@b.c', 'javascript:alert(1)', 'ftp://x', 'x#35;y', ''];
const SPRITES = ['', 'person', 'cylinder', 'queue', 'database', 'box', 'pipe', 'db', 'rounded', 'component', 'nothing', 'Person'];

// Random C4 diagrams, most of them well formed and drawable: shapes in nested boundaries,
// relations between them, style and layout updates, titles and front matter.
export function drawableDiagrams(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const chance = (p) => rnd() < p;
  const quote = (s) => (chance(0.9) ? `"${s}"` : s.replace(/[,)"]/g, '') || 'u');
  return () => {
    const header = pick(HEADERS);
    let s = '';
    if (chance(0.08)) s += `---\ntitle: ${pick(LABELS) || 'T'}\n---\n`;
    else if (chance(0.05)) s += '---\nconfig:\n  c4:\n    useMaxWidth: false\n---\n';
    else if (chance(0.03)) s += '%%{init: {"c4": {"c4ShapeInRow": 2}}}%%\n';
    s += header + '\n';
    if (chance(0.4)) s += `title ${pick(LABELS) || 'T'}\n`;
    if (chance(0.05)) s += 'accTitle: Accessible\n';
    if (chance(0.08)) s += chance(0.5) ? 'accDescr: Described\n' : 'accDescr {\n  one\n   two\n}\n';
    const aliases = [];
    const boundaries = [];
    let next = 0;
    const shapeLine = (indent) => {
      const alias = chance(0.05) && aliases.length ? pick(aliases) : `s${next++}`;
      aliases.push(alias);
      const kind = pick(SHAPES);
      const args = [alias, quote(pick(LABELS))];
      const container = kind.startsWith('Container') || kind.startsWith('Component');
      if (container && chance(0.7)) args.push(quote(pick(TECHNS)));
      if (chance(0.6)) args.push(quote(pick(DESCRS)));
      if (chance(0.15)) args.push(`$sprite="${pick(SPRITES)}"`);
      if (chance(0.1)) args.push(`$tags="${pick(['a,b', 'database', 'x, queue', 'person'])}"`);
      if (chance(0.15)) args.push(`$link="${pick(LINKS)}"`);
      if (chance(0.05)) args.push(`$descr="${pick(DESCRS)}"`);
      if (chance(0.03)) args.push(`$extra="e"`);
      return `${indent}${kind}(${args.join(', ')})\n`;
    };
    const block = (depth) => {
      let out = '';
      const indent = '  '.repeat(depth);
      const count = 1 + Math.floor(rnd() * (depth ? 4 : 6));
      for (let i = 0; i < count; i++) {
        if (depth < 4 && chance(depth ? 0.2 : 0.3)) {
          const alias = `b${next++}`;
          boundaries.push(alias);
          aliases.push(alias);
          const kind = header === 'C4Deployment' && chance(0.7) ? pick(BOUNDARIES.slice(4)) : pick(BOUNDARIES);
          const args = [alias, quote(pick(LABELS))];
          if (chance(0.4)) args.push(quote(pick(['', 'boundary', 'Ubuntu 18.04', 'system', 'A long type of a boundary'])));
          if (kind.includes('Node') && chance(0.4)) args.push(quote(pick(DESCRS)));
          if (chance(0.1)) args.push(`$link="${pick(LINKS)}"`);
          out += `${indent}${kind}(${args.join(', ')})${chance(0.8) ? ' {\n' : '\n' + indent + '{\n'}`;
          out += block(depth + 1);
          out += `${indent}}\n`;
        } else {
          out += shapeLine(indent);
        }
      }
      return out;
    };
    s += block(0);
    const rels = [];
    const relCount = Math.floor(rnd() * (chance(0.2) ? 14 : 6));
    for (let i = 0; i < relCount && aliases.length; i++) {
      const from = pick(aliases);
      const to = chance(0.05) ? from : chance(0.03) ? 'nowhere' : pick(aliases);
      rels.push([from, to]);
      const kind = pick(RELS);
      const args = [from, to, quote(pick(['Uses', 'Sends e-mails', 'Reads from and writes to', 'x', '', 'Makes API calls to', '中文', 'a label that is long enough to wrap']))];
      if (chance(0.5)) args.push(quote(pick(TECHNS)));
      if (chance(0.15)) args.push(quote(pick(DESCRS)));
      if (chance(0.08)) args.push(`$link="${pick(LINKS)}"`);
      if (kind === 'RelIndex') args.unshift(String(i));
      s += `${kind}(${args.join(', ')})\n`;
    }
    for (const [from, to] of rels) {
      if (!chance(0.25)) continue;
      const args = [from, to];
      if (chance(0.7)) args.push(`$textColor="${pick(COLORS)}"`);
      if (chance(0.7)) args.push(`$lineColor="${pick(COLORS)}"`);
      if (chance(0.5)) args.push(`$offsetX="${pick(['5', '-50', '0', '20', '3000', 'x'])}"`);
      if (chance(0.5)) args.push(`$offsetY="${pick(['-10', '20', '-40', '0', '-3000'])}"`);
      s += `UpdateRelStyle(${args.join(', ')})\n`;
    }
    for (const alias of aliases) {
      if (!chance(0.1)) continue;
      const args = [alias];
      if (chance(0.7)) args.push(`$bgColor="${pick(COLORS)}"`);
      if (chance(0.6)) args.push(`$fontColor="${pick(COLORS)}"`);
      if (chance(0.6)) args.push(`$borderColor="${pick(COLORS)}"`);
      if (chance(0.2)) args.push(`$shape="${pick(SPRITES)}"`);
      if (chance(0.1)) args.push(`$techn="${pick(TECHNS)}"`);
      s += `UpdateElementStyle(${args.join(', ')})\n`;
    }
    if (chance(0.3)) {
      const n = () => pick(['1', '2', '3', '4', '5', '0', '-1', 'x', '10']);
      s += chance(0.5) ? `UpdateLayoutConfig($c4ShapeInRow="${n()}", $c4BoundaryInRow="${n()}")\n` : `UpdateLayoutConfig("${n()}")\n`;
    }
    return s;
  };
}

// Strings in the sources of a folder of Pele's tests that hold the keyword, as Pele's
// loadCorpus collects them, from files whose names match.
function testStrings(pele, dir, files) {
  const out = new Set();
  const path = join(pele, dir);
  for (const file of readdirSync(path).filter((f) => files.test(f))) {
    const src = readFileSync(join(path, file), 'utf8');
    for (const m of src.matchAll(/`((?:[^`\\]|\\.)*)`|'((?:[^'\\\n]|\\.)*)'|"((?:[^"\\\n]|\\.)*)"/g)) {
      const raw = m[1] ?? m[2] ?? m[3];
      if (!KEYWORD.test(raw)) continue;
      try {
        out.add(new Function('return `' + raw.replace(/`/g, '\\`').replace(/\$\{/g, '\\${') + '`')());
      } catch {
        continue;
      }
    }
  }
  return [...out];
}

// The corpus the suites start from: documentation, specs, the strings of Pele's tests, its
// seeds and the cases above.
export function corpus(pele) {
  const docs = JSON.parse(readFileSync(join(pele, 'tests/corpus/c4-docs.json'), 'utf8'));
  return [
    ...new Set([
      ...docs,
      ...testStrings(pele, 'tests/compat/c4', /\.spec\.[jt]s$/),
      ...testStrings(pele, 'tests/render', /^c4.*\.test\.ts$/),
      ...testStrings(pele, 'tests/c4', /\.test\.ts$/),
      ...SEEDS,
    ]),
  ].filter((s) => !lone(s) && s.length <= 20000);
}

const FIELDS = ['type', 'descr', 'techn', 'sprite', 'tags', 'link', 'bgColor', 'fontColor', 'borderColor', 'shadowing', 'shape', 'legendText', 'legendSprite'];
const REL_FIELDS = ['techn', 'descr', 'sprite', 'tags', 'link', 'textColor', 'lineColor'];
const named = (a) => (a ? [a.key, a.value] : undefined);

// A model as the test writes limu's: every field that is defined, a parent by its place among
// the boundaries.
export function dumpModel(db) {
  const index = new Map(db.boundaries.map((b, i) => [b, i]));
  const element = (e) => {
    const o = { alias: e.alias, label: e.label, parent: e.parent ? index.get(e.parent) : null };
    if (e.aliasAttr) o.aliasAttr = named(e.aliasAttr);
    if (e.labelAttr) o.labelAttr = named(e.labelAttr);
    for (const f of FIELDS) if (e[f] !== undefined) o[f] = e[f];
    if (e.kind !== undefined) o.kind = e.kind;
    if (e.nodeType !== undefined) o.nodeType = e.nodeType;
    if (e.extra) o.extra = [...e.extra];
    return o;
  };
  const rel = (r) => {
    const o = { type: r.type, from: r.from, to: r.to, label: r.label };
    if (r.fromAttr) o.fromAttr = named(r.fromAttr);
    if (r.toAttr) o.toAttr = named(r.toAttr);
    if (r.labelAttr) o.labelAttr = named(r.labelAttr);
    for (const f of REL_FIELDS) if (r[f] !== undefined) o[f] = r[f];
    if (r.offsetX !== undefined) o.offsetX = enc(r.offsetX);
    if (r.offsetY !== undefined) o.offsetY = enc(r.offsetY);
    if (r.extra) o.extra = [...r.extra];
    return o;
  };
  const out = {
    c4Type: db.c4Type ?? null,
    shapes: db.shapes.map(element),
    boundaries: db.boundaries.map(element),
    rels: db.rels.map(rel),
    shapeInRow: enc(db.shapeInRow),
    boundaryInRow: enc(db.boundaryInRow),
  };
  if (db.title !== undefined) out.title = db.title;
  if (db.accDescr !== undefined) out.accDescr = db.accDescr;
  return out;
}

export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { c4 } = await src('diagrams/c4/index.ts');
  const { C4Db } = await src('diagrams/c4/db.ts');
  const { parseC4 } = await src('diagrams/c4/parser.ts');
  const { tokenize } = await src('diagrams/c4/lexer.ts');
  const { TOKEN_NAMES } = await src('diagrams/c4/tokens.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const direct = (source) => {
    const t = tokenize(source);
    const out = { tokens: t.types.map((k, i) => [TOKEN_NAMES[k], source.slice(t.starts[i], t.ends[i])]) };
    try {
      const db = new C4Db();
      parseC4(source, db);
      out.model = dumpModel(db);
    } catch (e) {
      Object.assign(out, caught(e));
    }
    return out;
  };
  // Pele's parseSource for a C4 diagram, and the raw text given to the parser alone.
  const run = (source) => {
    const out = { src: source, direct: direct(source) };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'c4') return out;
    try {
      out.model = dumpModel(c4.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title));
    } catch (e) {
      Object.assign(out, caught(e));
    }
    return out;
  };

  const base = corpus(pele);
  const all = [...base];
  const seen = new Set(all);
  const add = (s) => {
    if (s.length <= 4000 && !lone(s) && !seen.has(s)) {
      seen.add(s);
      all.push(s);
    }
  };
  for (const s of base) add(s + '\n');
  const statements = statementDiagrams(random(31));
  for (let i = 0; i < 2000; i++) add(statements());
  const drawable = drawableDiagrams(random(32));
  for (let i = 0; i < 1800; i++) add(drawable());
  // Mutated text; one in six is cut short, as a statement cut off by the end of the text ends
  // on a different error than one cut off by a line break.
  const rnd = random(33);
  const next = mutator([...seen].filter((s) => s.length <= 4000), FRAGMENTS, rnd);
  for (let i = 0; i < 3500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  // Random strings of tokens, where the order of the token types and their ends matter.
  const trnd = random(34);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['C4Context\n', 'C4Context ', 'C4Dynamic\nPerson(', 'C4Context\nPerson(a, ', 'C4Context\nRel(', 'C4Context\nBoundary(b) ', ''];
  for (let i = 0; i < 1000; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }
  // Pele's stress cases, smaller, which reach the lexer's searches and the parser's loops.
  const H = 'C4Context\n';
  for (const s of [
    H + 'Rel(a,b,"x")'.repeat(200) + ' direction XX',
    H + 'direction '.repeat(200) + '\n' + 'direction\n'.repeat(200),
    H + ('x direction' + '\n'.repeat(5)).repeat(100),
    H + 'Person(' + ' '.repeat(2000),
    H + 'Person(a, "' + 'x'.repeat(2000),
    H + 'Person(a, $' + 'k'.repeat(2000),
    H + 'Person(a, $k=' + ' '.repeat(2000) + 'x',
    H + 'accDescr {' + ' x\n'.repeat(500),
    H + ('accTitle' + ' '.repeat(20)).repeat(50),
    H + '%% c' + '\r\n'.repeat(500) + 'Person(a, "b")',
    H + 'Person'.repeat(300),
    H + 'Person(' + ','.repeat(2000) + ')',
    H + 'Person(' + 'a,'.repeat(1000) + '"z")',
    H + 'Boundary(b, "B") {\n'.repeat(150) + 'System(s, "S")\n' + '}\n'.repeat(150),
  ]) {
    seen.delete(s);
    add(s);
  }
  write('diagram/c4/testdata/pele/c4.json', { cases: all.map(run) });
}
