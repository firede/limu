// Git graphs parsed as Pele parses them, from text to the tree parseGit reads and the GitModel
// buildGit makes of it: Mermaid's documentation and specs, the sources of Pele's tests, cases
// written here, those sources mutated at random, random strings of tokens as Pele's test makes
// them, and random histories that mostly get past the model builder's checks. Each case is
// parsed as Pele's parse does it, after the preprocessing every diagram goes through, and as
// the parser alone reads the raw text.
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { enc, lone, mutator, peleError } from './parse.mjs';

// Pele's fragments, tests/git/oracle.test.ts.
export const FRAGMENTS = [
  'gitGraph', 'gitGraph:', 'gitGraph ', ' LR:', ' TB:', ' BT:', 'LR', 'TB', 'BT', ':', ' : ',
  'commit', 'commit ', 'branch ', 'merge ', 'checkout ', 'switch ', 'cherry-pick ', 'cherry-pick',
  'id:', 'id: ', 'msg:', 'tag:', 'tag: ', 'type:', 'type: ', 'parent:', 'order:', 'order: ',
  'NORMAL', 'REVERSE', 'HIGHLIGHT', 'id', 'tag', 'order', 'main', 'develop', 'a/b', 'a.b', 'a-', 'a.', '-a', '_a', '1.0.1',
  'title', 'title ', ' title x', 'accTitle: ', 'accDescr: ', 'accDescr {', '}', '{',
  '"', "'", '"a"', "'b'", '""', '"x y"', '0', '1', '12', '01', '1.', '1.5', '-1',
  '\n', '\r\n', ' ', '\t', '%%', '%% c', '%%{init: {}}%%', '---', '---\n',
  '\\', '\\"', 'x', 'é', ';', ',', '.', '-', '/', '#35;', ' id:"a"', ' tag:"t"', '\ncommit', '\nbranch x', '\nmerge x', '\ncheckout main',
  // And for the model builder and the drawing: ids that clash, long names, entities, Unicode.
  ' id:"0-9e3779b"', ' id:"1-3c6ef36"', '"a long commit message that goes on"', '<b>x</b>', '<br>', '😀', '日本', ':fa-car', '&amp;',
  ' ', ' ',
];

// Pele's extra cases, tests/git/oracle.test.ts.
const SEEDS = [
  'gitGraph',
  'gitGraph:',
  'gitGraph LR:',
  'gitGraph TB:\ncommit',
  'gitGraph BT :\ncommit id:"a" msg:"m" tag:"t" tag:"u" type:REVERSE',
  'gitGraph commit "msg" id: "x"',
  'gitGraph: commit\nbranch "quoted name" order: 3\ncheckout "quoted name"\ncommit\nswitch main\nmerge "quoted name" id:"m" tag:"v" type:HIGHLIGHT',
  'gitGraph\n title A title\n accTitle: at\n accDescr { multi\n line }\n commit\n cherry-pick id:"a" parent:"b" tag:""',
  '\n\n  gitGraph\n\n  commit\n\n\n  branch 1.0.1\n  branch feature/x_y-z order: 10\n',
  '---\ntitle: x\n---\ngitGraph\ncommit',
  '%%{init: {"theme":"dark"}}%%\ngitGraph %% c\ncommit %% d\n',
  'gitGraph\ncommit\nbranch commits\ncheckout TBD\nmerge LRU\nbranch title1',
];

// Cases written here, for the corners of the lexer, the parser and the model builder.
export const WRITTEN = [
  'gitGraph\ncommit\ncommit\nbranch develop\ncheckout develop\ncommit\ncommit\ncheckout main\nmerge develop\ncommit\ncommit',
  'gitGraph\ncommit id:"a"\nbranch b\ncommit id:"b1"\ncheckout main\ncommit id:"m1"\ncherry-pick id:"b1"\ncommit',
  'gitGraph\ncommit id:"a"\nbranch b\ncommit id:"b1"\ncheckout main\nmerge b id:"mg"\nbranch c\ncommit\ncherry-pick id:"mg" parent:"b1"',
  'gitGraph\ncommit id:"a"\nbranch b\ncommit id:"b1"\ncheckout main\nmerge b id:"mg"\nbranch c\ncommit\ncherry-pick id:"mg"',
  'gitGraph\ncommit id:"a"\nbranch b\ncommit id:"b1"\ncheckout main\nmerge b id:"mg"\nbranch c\ncommit\ncherry-pick id:"mg" parent:"zz"',
  'gitGraph\ncommit id:"a"\nbranch b\ncommit id:"b1"\ncheckout main\nmerge b id:"mg"\nbranch c\ncommit\ncherry-pick id:"mg" parent:"b1" tag:"" tag:"t"',
  'gitGraph\ncommit id:"a"\nbranch b\ncommit\ncherry-pick id:"a" tag:""',
  'gitGraph\ncommit id:"a"\ncherry-pick id:"a"',
  'gitGraph\ncommit id:"a"\nbranch b\ncherry-pick id:"a"',
  'gitGraph\ncommit\nbranch b\ncherry-pick id:"zz"',
  'gitGraph\ncherry-pick',
  'gitGraph\ncommit\nmerge main',
  'gitGraph\nmerge x',
  'gitGraph\ncommit\nmerge x',
  'gitGraph\ncommit\nbranch b\nmerge main',
  'gitGraph\nbranch b\ncommit\ncheckout main\nmerge b',
  'gitGraph\ncommit\nbranch b\ncheckout main\nmerge b',
  'gitGraph\ncommit\nbranch b\ncommit\ncheckout main\nmerge b\nmerge b',
  'gitGraph\ncommit\nbranch b\ncommit\ncheckout main\ncommit\nmerge b id:"x"\nbranch c\ncommit\nmerge main id:"x"',
  'gitGraph\ncommit\nbranch b\ncommit\ncheckout main\ncommit\nmerge b\ncheckout b\nmerge main',
  'gitGraph\ncommit\nbranch b\ncommit\ncheckout main\ncommit\nmerge b type:REVERSE tag:"r"',
  'gitGraph\ncommit\nbranch main',
  'gitGraph\ncheckout x',
  'gitGraph\nswitch "a b"',
  'gitGraph\ncommit id:"x"\ncommit id:"x"\ncommit id:"x"',
  'gitGraph\ncommit\ncommit id:"0-9e3779b"\ncommit',
  'gitGraph\ncommit id:"1-3c6ef36"\ncommit\ncommit',
  'gitGraph\ncommit id:""\ncommit msg:""\ncommit tag:""',
  'gitGraph\nbranch a order: 2\nbranch b order: 1\nbranch c\nbranch d order: 0\nbranch e order: 99999999999999999999999',
  'gitGraph\nbranch a order: ' + '9'.repeat(400),
  'gitGraph\nbranch a order: 1.5',
  'gitGraph\nbranch a order: 12.',
  'gitGraph\nbranch a order: 0.5',
  'gitGraph\nbranch a order: 007',
  'gitGraph\nbranch a order: x',
  'gitGraph\nbranch a order:',
  'gitGraph\nbranch',
  'gitGraph\nbranch a.',
  'gitGraph\nbranch a/',
  'gitGraph\nbranch a.b./c-',
  'gitGraph\nbranch -a',
  'gitGraph\nbranch _',
  'gitGraph\nbranch é',
  'gitGraph\ncommit id:"a\\"b" msg:\'c\\\'d\' tag:"\\n\\t\\x\\0\\\\"',
  'gitGraph\ncommit id:"a\\\nb"',
  'gitGraph\ncommit id:"a\nb"',
  'gitGraph\ncommit id:"😀\\😀"',
  'gitGraph\ncommit "m1" "m2" msg:"m3"',
  'gitGraph\ncommit id "x"',
  'gitGraph\ncommit type:normal',
  'gitGraph\ncommit type:',
  'gitGraph\ncommit id:',
  'gitGraph\ncommit; commit',
  'gitGraph\ncommit commit',
  'gitGraph\ncommitx',
  'gitGraph\ncommit-x',
  'gitGraph\ncommit.x',
  'gitGraph\ncommit:',
  'gitGraph LR',
  'gitGraph LR\ncommit',
  'gitGraph LR :\ncommit',
  'gitGraph RL:\ncommit',
  'gitGraph TB: commit',
  'gitGraph::',
  'gitGraph:\n:',
  'gitGraphx',
  'gitGraph%%c\ncommit',
  'gitGraph \ncommit',
  'gitGraph commit',
  ' gitGraph\ncommit',
  'gitgraph\ncommit',
  'gitGraph\ntitle\ncommit',
  'gitGraph\ntitle   a   b  %% c\ncommit',
  'gitGraph\ntitle1',
  'gitGraph\ntitle\tx\ncommit',
  'gitGraph\ntitle x\ntitle y\ncommit',
  'gitGraph\naccTitle:   a   b  \ncommit',
  'gitGraph\naccTitle: a %% b\ncommit',
  'gitGraph\naccTitle\ncommit',
  'gitGraph\naccDescr: a  b\ncommit',
  'gitGraph\naccDescr {\n  a   b  \n\n\n   c\t\n}\ncommit',
  'gitGraph\naccDescr{x}\ncommit',
  'gitGraph\naccDescr\n{\n x\n }\ncommit',
  'gitGraph\naccDescr {\ncommit',
  'gitGraph\naccDescr {\ncommit\naccDescr: d\naccDescr { e }',
  'gitGraph\naccDescr { a }  commit',
  'gitGraph\naccDescr {  a  b }\ncommit',
  'gitGraph\n%%{init: {}}%%\ncommit',
  'gitGraph\n  %%{init: {}}%% x\ncommit',
  'gitGraph\n%%{ x\ncommit',
  'gitGraph\n%%{ x }%%}%%\ncommit',
  'gitGraph\n---\na: b\n---\ncommit',
  'gitGraph\n---\n---\ncommit',
  'gitGraph\n---\n--- x\ncommit',
  'gitGraph\n---\ncommit',
  'gitGraph\n--- \r\na\r\n---\r\ncommit',
  'gitGraph\r\ncommit\r\nbranch b\r\ncommit\r',
  'gitGraph\rcommit',
  'gitGraph\n\tcommit\t id:"a"\t\n',
  '---\nconfig:\n  gitGraph:\n    mainBranchName: trunk\n---\ngitGraph\ncommit\nbranch b\ncommit\ncheckout trunk\nmerge b',
  '---\nconfig:\n  gitGraph:\n    mainBranchName: trunk\n---\ngitGraph\ncommit\ncheckout main',
  '---\nconfig:\n  gitGraph:\n    mainBranchName: trunk\n    mainBranchOrder: 2\n---\ngitGraph\ncommit\nbranch a order: 1\nbranch b order: 3\nbranch c',
  '---\nconfig:\n  gitGraph:\n    mainBranchName: 5\n    mainBranchOrder: "x"\n---\ngitGraph\ncommit\nbranch b',
  '---\nconfig:\n  gitGraph:\n    mainBranchOrder: -1\n---\ngitGraph\nbranch b order: -2\nbranch c',
  '%%{init: {"gitGraph": {"mainBranchName": "dev"}}}%%\ngitGraph\ncommit\nbranch main\ncommit',
  '---\ntitle: Front\n---\ngitGraph\ntitle Own\ncommit',
  '---\ntitle: Front\n---\ngitGraph\ntitle\ncommit',
  'gitGraph\ncommit msg:"#35; &amp; #quot;"\nbranch "a#35;b"\ncommit id:"#35;"',
];

// The sources in Mermaid's specs as Pele ports them: the string literals Pele's tests collect,
// and the whole sources the specs build by adding literals.
function specSources(pele) {
  const out = new Set();
  const dir = join(pele, 'tests/compat/git');
  const evaluate = (raw) => {
    try {
      return new Function('return `' + raw.replace(/`/g, '\\`').replace(/\$\{/g, '\\${') + '`')();
    } catch {
      return undefined;
    }
  };
  for (const file of readdirSync(dir).filter((f) => /\.(spec|test)\.[jt]s$/.test(f))) {
    const src = readFileSync(join(dir, file), 'utf8');
    for (const m of src.matchAll(/`((?:[^`\\]|\\.)*)`|'((?:[^'\\\n]|\\.)*)'|"((?:[^"\\\n]|\\.)*)"/g)) {
      const raw = m[1] ?? m[2] ?? m[3];
      if (!/gitGraph/.test(raw)) continue;
      const text = evaluate(raw);
      if (text !== undefined) out.add(text);
    }
    for (const m of src.matchAll(/const str =\s*((?:'(?:[^'\\\n]|\\.)*'|`(?:[^`\\]|\\.)*`|"(?:[^"\\\n]|\\.)*"|\s|\+)+);/g)) {
      try {
        out.add(new Function('return ' + m[1])());
      } catch {}
    }
  }
  return [...out];
}

// The corpus the suites start from: documentation, specs, Pele's seeds and the cases above.
export function corpus(pele) {
  const docs = JSON.parse(readFileSync(join(pele, 'tests/corpus/git-docs.json'), 'utf8'));
  return [...new Set([...docs, ...specSources(pele), ...SEEDS, ...WRITTEN])].filter((s) => !lone(s));
}

// Random sequences of whole tokens, as Pele's test makes them.
const WORDS = [
  'gitGraph', 'gitGraph:', 'LR', 'TB', 'BT', ':', 'commit', 'branch', 'merge', 'checkout', 'switch', 'cherry-pick',
  'id:', 'msg:', 'tag:', 'type:', 'parent:', 'order:', 'NORMAL', 'REVERSE', 'HIGHLIGHT', '"s"', "'t'", '""', 'name', 'a/b.c-d',
  '0', '7', '12', '\n', '\n', '\n', 'title x', 'accTitle: a', 'accDescr: d', 'accDescr { x\n y }', '%% c',
];
const HEADERS = ['gitGraph', 'gitGraph:', 'gitGraph :', 'gitGraph LR:', 'gitGraph TB :', 'gitGraph BT:', 'gitGraph\n', 'gitGraph:\n', 'gitGraph TB:\n', ''];
const CLAUSES = [
  'id:"a"', 'id: "b"', 'msg:"m"', '"m"', 'tag:"t"', 'tag: ""', 'type:NORMAL', 'type: REVERSE', 'type:HIGHLIGHT', 'type:"x"',
  'parent:"p"', 'order: 1', 'order:20', 'order:x', 'name', 'main', '"quoted name"', '7', 'LR', ':',
];
const STATEMENTS = [
  ['commit', ['id:"a"', 'msg:"m"', '"m"', 'tag:"t"', 'type:NORMAL', 'type: REVERSE']],
  ['branch', ['name', '"quoted name"', 'order: 1', 'order:20']],
  ['merge', ['name', 'main', 'id:"a"', 'tag:"t"', 'type:HIGHLIGHT']],
  ['checkout', ['name', 'main']],
  ['switch', ['name', '"quoted name"']],
  ['cherry-pick', ['id:"a"', 'tag:"t"', 'tag: ""', 'parent:"p"']],
  ['title t', []],
  ['accTitle: a', []],
  ['accDescr { d }', []],
  ['', []],
];

export function soup(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const gap = () => {
    const r = rnd();
    return r < 0.75 ? ' ' : r < 0.9 ? '\n' : '';
  };
  return () => {
    let s = pick(HEADERS);
    if (rnd() < 0.5) {
      for (let k = Math.floor(rnd() * 12); k > 0; k--) s += gap() + pick(WORDS);
      return s;
    }
    for (let k = Math.floor(rnd() * 5); k > 0; k--) {
      const [statement, own] = pick(STATEMENTS);
      s += (rnd() < 0.9 ? '\n' : ' ') + statement;
      for (let c = Math.floor(rnd() * 5); c > 0; c--) s += gap() + pick(own.length > 0 && rnd() < 0.8 ? own : CLAUSES);
    }
    return s;
  };
}

// Histories that keep to what the model builder allows, mostly: branches made, checked out,
// committed to, merged and cherry-picked from, with ids, tags, types and messages, in every
// direction, with the settings of front matter.
export function histories(rnd, opts = {}) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const chance = (p) => rnd() < p;
  const tags = ['v1.0', 'release', 'a much longer tag name', 'x', 'β', '日本', 'hot<fix>', '#35;1'];
  const ids = ['init', 'a', 'fix-1', 'Feature X', 'a very long commit id that goes on and on', '日本語', 'x&y', 'z<br>w'];
  const messages = ['Fix bug', 'Add feature', 'a message with #quot;quotes#quot;', ''];
  const names = ['develop', 'feature/login', 'hotfix', 'release-1.0', 'b', '"quoted name"', 'x.y'];
  return () => {
    let head = '';
    const settings = [];
    if (opts.config && chance(0.5)) {
      if (chance(0.3)) settings.push(`    showBranches: ${pick(['true', 'false'])}`);
      if (chance(0.3)) settings.push(`    showCommitLabel: ${pick(['true', 'false'])}`);
      if (chance(0.4)) settings.push(`    rotateCommitLabel: ${pick(['true', 'false'])}`);
      if (chance(0.4)) settings.push(`    parallelCommits: ${pick(['true', 'false'])}`);
      if (chance(0.15)) settings.push(`    mainBranchName: ${pick(['trunk', 'master'])}`);
      if (chance(0.15)) settings.push(`    mainBranchOrder: ${pick([0, 1, 2, -1, 0.5])}`);
    }
    const title = opts.config && chance(0.2) ? `title: ${pick(['History', 'A long title for a short history of commits'])}\n` : '';
    if (settings.length > 0 || title) {
      head = '---\n' + title + (settings.length > 0 ? 'config:\n  gitGraph:\n' + settings.join('\n') + '\n' : '') + '---\n';
    }
    const main = (settings.find((s) => s.includes('mainBranchName')) ?? 'main').replace(/.*: /, '');
    const lines = [`gitGraph${pick(['', '', ' LR:', ' TB:', ' BT:', ':'])}`];
    if (chance(0.1)) lines.push(`title ${pick(['Own title', 'T'])}`);
    const branches = [main];
    const heads = new Map([[main, null]]);
    const used = new Set();
    let current = main;
    let n = 0;
    const commitIds = [];
    const clauses = () => {
      let s = '';
      if (chance(0.3)) {
        let id = pick(ids) + (chance(0.5) ? n : '');
        if (used.has(id) && chance(0.9)) id += '-' + n;
        used.add(id);
        s += ` id:"${id}"`;
      }
      for (let k = chance(0.25) ? 1 + Math.floor(rnd() * 3) : 0; k > 0; k--) s += ` tag:"${pick(tags)}"`;
      if (chance(0.2)) s += ` type:${pick(['NORMAL', 'REVERSE', 'HIGHLIGHT'])}`;
      return s;
    };
    for (let k = 2 + Math.floor(rnd() * 14); k > 0; k--) {
      const r = rnd();
      if (r < 0.45 || heads.get(current) === null) {
        const c = clauses();
        const msg = chance(0.2) ? ` msg:"${pick(messages)}"` : '';
        lines.push('commit' + c + msg);
        const m = /id:"([^"]*)"/.exec(c);
        heads.set(current, { id: m ? m[1] : null, branch: current });
        if (m) commitIds.push({ id: m[1], branch: current });
        n++;
      } else if (r < 0.6 && branches.length < 6) {
        let name = pick(names);
        if (branches.includes(name)) name = `b${n}`;
        branches.push(name);
        heads.set(name, heads.get(current));
        lines.push(`branch ${name}${chance(0.2) ? ` order: ${Math.floor(rnd() * 4)}` : ''}`);
        current = name;
      } else if (r < 0.75) {
        current = pick(branches);
        lines.push(`${pick(['checkout', 'switch'])} ${current}`);
      } else if (r < 0.92) {
        const other = pick(branches);
        if (other === current || heads.get(other) === null || heads.get(current) === null) continue;
        const c = clauses().replace(/ msg:"[^"]*"/, '');
        lines.push(`merge ${other}${c}`);
        const m = /id:"([^"]*)"/.exec(c);
        heads.set(current, { id: m ? m[1] : null, branch: current });
        n++;
      } else {
        const src = commitIds.filter((c) => c.branch !== current);
        if (src.length === 0 || heads.get(current) === null) continue;
        const from = pick(src);
        lines.push(`cherry-pick id:"${from.id}"${chance(0.3) ? ' tag:"picked"' : ''}`);
        heads.set(current, { id: null, branch: current });
        n++;
      }
    }
    return head + lines.join('\n') + '\n';
  };
}

// A tree as JSON can carry it, with an order of no finite value written as enc writes it.
const dumpAst = (ast) => JSON.parse(JSON.stringify(ast, (_, v) => (typeof v === 'number' && !Number.isFinite(v) ? enc(v) : v)));

function dumpModel(m) {
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  return {
    title: opt(m.title),
    accTitle: opt(m.accTitle),
    accDescr: opt(m.accDescr),
    direction: m.direction,
    commits: [...m.commits.values()].map((c) => ({
      id: c.id,
      message: c.message,
      seq: c.seq,
      type: c.type,
      tags: c.tags,
      parents: c.parents,
      branch: c.branch,
      customType: c.customType,
      customId: c.customId,
    })),
    branches: [...m.branches.entries()],
    lanes: m.lanes,
    currentBranch: m.currentBranch,
    head: m.head === null ? null : m.head.id,
    warnings: m.warnings,
  };
}

export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { parseGit } = await src('diagrams/git/parser.ts');
  const { buildGit } = await src('diagrams/git/db.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const parsed = (text, config, title) => {
    let ast;
    try {
      ast = parseGit(text);
    } catch (e) {
      return caught(e);
    }
    try {
      return { ast: dumpAst(ast), model: dumpModel(buildGit(ast, config, title)) };
    } catch (e) {
      return { ast: dumpAst(ast), ...caught(e) };
    }
  };
  // Pele's parseSource for a git graph, and the raw text given to the parser alone.
  const run = (source) => {
    const out = { src: source, direct: parsed(source, {}, undefined) };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'gitGraph') return out;
    const piped = parsed(encodeEntities(pre.text) + '\n', pre.config, pre.title);
    if (JSON.stringify(piped) === JSON.stringify(out.direct)) return { ...out, same: true };
    return { ...out, ...piped };
  };

  const base = corpus(pele);
  const all = [...base];
  const seen = new Set(all);
  const add = (s) => {
    if (s.length <= 3000 && !lone(s) && !seen.has(s)) {
      seen.add(s);
      all.push(s);
    }
  };
  const rnd = random(21);
  const next = mutator(base, FRAGMENTS, rnd);
  for (let i = 0; i < 3000; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  const words = soup(random(22));
  for (let i = 0; i < 1500; i++) add(words());
  const history = histories(random(23), { config: true });
  for (let i = 0; i < 1500; i++) add(history());
  const hnext = mutator([...all.slice(-1500)], FRAGMENTS, random(24));
  for (let i = 0; i < 800; i++) add(hnext());

  write('diagram/gitgraph/testdata/pele/gitgraph.json', { cases: all.map(run) });
}
