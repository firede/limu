// Packet diagrams parsed as Pele parses them: the tokens Pele's common/tokens.ts makes of the
// text with the packet's table, the tree parsePacket makes of them, and the model buildPacket
// makes of the tree with the configuration the text's front matter and directives give. The
// cases are Mermaid's documentation and specs, the inputs of Pele's own tests, cases written
// here, random diagrams, and all of those mutated at random.
import { enc, peleError } from '../lib/value.mjs';
import { lone, picker, testStrings, docs } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { locator } from '../lib/located.mjs';

// The fragments of Pele's tests/packet/oracle.test.ts, and more.
export const FRAGMENTS = [
  'packet', 'packet-beta', 'title', 'title ', 'accTitle: ', 'accDescr: ', 'accDescr {', 'accDescr {}', '}', ':', ' : ', '-', '+', '--', '---',
  '---\n', '\n---\n', '"', "'", '"a"', "'b'", '1', '0', '15', '16-31', '+8', '01', '1.', '.5', '1e3', '\n', '\r\n', ' ', '\t',
  '%%', '%% c', '%%{init: {}}%%', '\\', '\\"', 'x', 'é', ';', ',', '#35;', '0-7: "a"', '\n+4: "b"', '\n8: "c"',
  '\r', '\u2028', '\u3000', '😀', '中文', '%%{', '}%%', '"a\\nb"', "'it\\'s'", '&amp;', '#quot;', '<br>', 'accDescr {\n a\n}\n',
  '\n+300: "wide"', '\n+1: "bit"', '99999999999999999999', '\n+0: "zero"', '--- \n', '---\t\n', '\n---', '----',
];

// The inputs of Pele's tests/packet/oracle.test.ts beyond its corpus.
const SEEDS = [
  'packet',
  'packet\naccDescr {}\n0: "a"',
  'packet-beta',
  'packet-betax',
  'packet-beta x',
  'packetx',
  'packet%% c',
  'packet-beta%% c\n0: "a"',
  'packet\n0-15: "Source"\n16: \'x\'\n+8: "y\\"z"',
  'packet 0-15: "a"',
  'packet 0-15: "a" 16: "b"',
  'packet\n0-15: "a"\n\n\n',
  'packet\n0 - 15 : "a"',
  'packet\n0-15 "a"',
  'packet\n-15: "a"',
  'packet\n+ 8: "a"',
  'packet\n+8-9: "a"',
  'packet\n08: "a"',
  'packet\n8.: "a"',
  'packet\n0-1: "a"\n---\nfoo\n---\n2: "b"',
  'packet\n0---\n---\n1: "b"',
  'packet\n0---\n---: "b"',
  'packet\n0---: "b"',
  'packet\n0-\n1: "b"',
  'packet\ntitle T\n0: "a"',
  'packet title T\n0: "a"',
  'packet\ntitle T\naccTitle: A\naccDescr: D\n',
  'packet\ntitle T 0: "a"',
  'packet\n99999999999999999999-5: "a"',
  'packet\n0: "a" %% c\n1: "b"',
  '\n\n packet \n\n 0: "a"',
  'packet\n0: "a\n b"',
  'packet\n0: ""',
  'packet\n0:',
  'packet\n0',
  'packet\n: "a"',
  'packet\n+0: "a"',
  'packet\n10-5: "a"',
  'packet-beta\npacket',
  'packet\n1-2-3: "a"',
  'packet\naccDescr {\n a\n b\n}\n0: "a"',
  '---\ntitle: x\n---\npacket\n0: "a"',
  '%%{init: {"theme":"dark"}}%%\npacket title A %% c\n0-3: "a" %% d\n',
];

const config = (lines, body) => `---\nconfig:\n  packet:\n${lines.map((l) => '    ' + l).join('\n')}\n---\n${body}`;
const TCP = 'packet\n0-15: "Source Port"\n16-31: "Destination Port"\n32-63: "Sequence Number"\n64: "URG"';

// Cases written here: what the tokens and the model do at their edges, and Pele's render tests.
export const WRITTEN = [
  TCP,
  'packet\n0-15: "a"\n16: "b"',
  'packet\n0-10: "test"\n11-90: "multiple"',
  'packet\n0: "Wide"\n1: "Extraordinarily long"\n2-31: "x"',
  config(['bitsPerRow: 8', 'bitWidth: 10'], 'packet\n0-7: "a"\n8-15: "b"\n16-23: "c"'),
  config(['rowHeight: 80'], 'packet\n0-7: "a"\n8-15: "b"\n16-23: "c"'),
  config(['bitsPerRow: 8', 'paddingY: 30'], 'packet\n0-7: "a"\n8-15: "b"\n16-23: "c"'),
  config(['paddingX: 20'], 'packet\n0-7: "a"\n8-15: "b"\n16-23: "c"'),
  config(['bitsPerRow: 16', 'bitOrder: descending'], 'packet\n0-7: "DATA"\n8-11: "TYPE"\n12: "EN"\n13-15: "RESERVED"\n16-19: "NEXT"'),
  config(['showBits: false'], 'packet\n0-15: "a"\n16-47: "b"'),
  config(['bitsPerRow: 8'], 'packet\ntitle T\naccTitle: A\naccDescr: D\n+4: "a"\n+8: "b"'),
  config(['bitsPerRow: 1'], 'packet\n+20000: "a"\n+1: "b"'),
  config(['bitsPerRow: -4', 'bitWidth: "wide"', 'rowHeight: 0', 'paddingX: 9999'], TCP),
  config(['bitsPerRow: 2.5'], 'packet\n+9: "a"'),
  config(['bitsPerRow: 5000'], 'packet\n+9000: "a"\n+1: "b"'),
  config(['bitsPerRow: .inf'], 'packet\n+9000: "a"'),
  config(['bitsPerRow: .nan'], 'packet\n+9: "a"'),
  config(['bitsPerRow: 0.5'], 'packet\n+9: "a"'),
  config(['bitsPerRow: 1.5'], 'packet\n+9: "a"'),
  '---\nconfig:\n  packet: 5\n---\npacket\n+9: "a"',
  '---\nconfig:\n  packet: [1, 2]\n---\npacket\n+9: "a"',
  '%%{init: {"packet": {"bitsPerRow": 4}}}%%\npacket\n+9: "a"\n+3: "b"',
  '%%{init: {"packet": {"bitsPerRow": "4"}}}%%\npacket\n+9: "a"',
  'packet\n+1000000000: "wide"\n+1: "after"',
  'packet\n+' + '9'.repeat(400) + ': "x"\n+1: "y"',
  'packet\n+' + '9'.repeat(400) + ': "x"\n5: "y"',
  'packet\n+' + '9'.repeat(30) + ': "x"\n+1: "y"',
  'packet\n0-' + '9'.repeat(25) + ': "x"',
  'packet\n' + '9'.repeat(25) + ': "x"',
  'packet\n0-31: "a"\n32: "b"\n33-' + '9'.repeat(18) + ': "c"',
  'packet\n0-16: "a"\n18-20: "b"',
  'packet\n0-16: "a"\n25-20: "b"',
  'packet\n+16: "a"\n18-20: "b"',
  'packet\n0-16: "a"\n18: "b"',
  'packet\n+16: "a"\n18: "b"',
  'packet\n0-16: "a"\n17: "b"\n19: "c"',
  'packet\n0: "a"\n0: "b"',
  'packet\n1: "a"',
  'packet\n+1: "a"\n+0: "b"',
  'packet\n0-0: "a"',
  'packet\n0-31: "a"\n32-63: "b"',
  'packet\n0-32: "a"',
  'packet\n0-63: "a"\n64-64: "b"',
  'packet\n+32: "a"\n+32: "a"\n+1: "a"',
  'packet\n0: "a" 1: "b"',
  'packet\n0: "a"1: "b"',
  'packet\n0:"a"\n1 :"b"\n2: \'c\'',
  'packet\n0: "a\\nb\\tc\\\\d\\"e\\\'f\\qg\\0h"',
  "packet\n0: 'a\\'b\"c'",
  'packet\n0: "a\\',
  'packet\n0: "a\\\nb"',
  'packet\n0: "a\\\u2028b"',
  'packet\n0: "\u2028"',
  'packet\n0: "a\rb"',
  'packet\n0: "😀\\😀"',
  'packet\n0: "unclosed',
  "packet\n0: 'unclosed",
  'packet\n0: "a"\r\n1: "b"\r\n',
  'packet\r\n0: "a"\r1: "b"',
  'packet\n\u00a00: "a"',
  'packet\n0: "a"\u3000',
  'packet\n\t0\t-\t1\t:\t"a"\t',
  'packet\n0.5: "a"',
  'packet\n0.: "a"',
  'packet\n10.: "a"',
  'packet\n1.5: "a"',
  'packet\n00: "a"',
  'packet\n0x1: "a"',
  'packet\n1e3: "a"',
  'packet\n-1: "a"',
  'packet\n+-1: "a"',
  'packet\n++1: "a"',
  'packet\n+1 "a"',
  'packet\n+1:',
  'packet\n+1: a',
  'packet\n0-: "a"',
  'packet\n0-1-: "a"',
  'packet\n0--1: "a"',
  'packet\n0---1: "a"',
  'packet\n0- 1: "a"',
  'packet\n0 -1: "a"',
  'packet\n---\n0: "a"',
  'packet\n---\n---\n0: "a"',
  'packet\n---\n0: "a"\n---\n1: "b"',
  'packet\n--- \n0: "a"\n---x\n1: "b"\n---\n2: "c"',
  'packet\n0: "a"\n---\n---',
  'packet\n0: "a"\n---\r\n---\r\n1: "b"',
  'packet\n0: "a"\n---\n---\r1: "b"',
  'packet\n0: "a"\n-\n1: "b"',
  'packet\ntitle',
  'packet\ntitle\n0: "a"',
  'packet\ntitle   spaced    out   \n0: "a"',
  'packet\ntitle x %% c\n0: "a"',
  'packet\ntitle %%c\n0: "a"',
  'packet\ntitle%%c\n0: "a"',
  'packet\ntitlex\n0: "a"',
  'packet\ntitle\tT\n0: "a"',
  'packet\n  title T\n0: "a"',
  'packet\ntitle A\ntitle B\n0: "a"',
  'packet\ntitle #35; &amp; <b>b</b>\n0: "a"',
  'packet\ntitle A very long title of a packet that is wider than its fields are, by far\n0-7: "a"',
  'packet\naccTitle: x\naccTitle: y',
  'packet\naccTitle:    spaced    x   ',
  'packet\naccTitle : x',
  'packet\naccTitle:x %% c',
  'packet\naccTitle',
  'packet\naccDescr: one %% c\n0: "a"',
  'packet\naccDescr :one',
  'packet\naccDescr{x}',
  'packet\naccDescr\n{\n  x\n}',
  'packet\naccDescr {\n  line one\n    line two\n\n\n  line three   \n}\n0: "a"',
  'packet\naccDescr {\n\t\n  a  b \r\n\r\n c\n}',
  'packet\naccDescr {',
  'packet\naccDescr {\n a\n}\n}',
  'packet\naccDescr { a } 0: "a"',
  'packet\naccDescr: d\naccDescr {\n e\n}',
  'packet\n%%{init: {}}%%\n0: "a"',
  'packet\n0: "a" %%{x}%%\n1: "b"',
  'packet\n0: "a"\n%%{\n1: "b"',
  'packet\n%%{a}%%b}%%\n0: "a"',
  'packet\n  %% indented comment\n0: "a"',
  'packet\n0: "a" %%',
  'packet  %%  c\n0: "a"',
  'packet-beta\n0: "a"',
  'packet-beta-beta',
  'packet-beta\n0: "a"\npacket-beta',
  'packet\n0: "a"\npacket',
  'packet\npacket-beta',
  'packet\n0: "a"\ntitle T\n1: "b"\naccTitle: A\n2: "b"',
  'packet\n0: "&lt;b&gt;"',
  'packet\n0: "#35;#quot;"',
  'packet\n0: "<script>x</script>"',
  'packet\n0-31: "' + 'word '.repeat(40) + '"',
  'packet\n0-7: "中文中文中文中文中文"',
  'packet\n0-7: "😀😀😀😀😀😀😀😀😀😀"',
  'packet\n0-3: "a b"\n4-5: "c d e f g"\n6: " "\n7: "   x   "',
  'Packet\n0: "a"',
  'PACKET\n0: "a"',
  'packet:\n0: "a"',
  'packets\n0: "a"',
  'packet ',
  'packet\n',
  '',
  'not a packet',
];

// The corpus the suites start from: documentation, specs, the strings of Pele's tests, its
// seeds and the cases above.
export function corpus(pele) {
  return [
    ...new Set([
      ...docs(pele, 'packet'),
      ...testStrings(pele, 'tests/compat/packet', /\.spec\.[jt]s$/, /packet/),
      ...testStrings(pele, 'tests/compat/packet', /\.test\.[jt]s$/, /packet/),
      ...testStrings(pele, 'tests/render', /^packet.*\.test\.ts$/, /packet/),
      ...SEEDS,
      ...WRITTEN,
    ]),
  ].filter((s) => !lone(s) && s.length <= 50000);
}

const LABELS = [
  'a', 'x', 'Source Port', 'Destination Port', 'Sequence Number', 'Data (variable length)', 'URG', 'ACK', 'Checksum', '',
  ' spaced ', 'Extraordinarily long label for a narrow field', 'Wide', 'two words', '中文标签', '😀 emoji', '#35;', '&amp; more',
  '#quot;q#quot;', 'a\\"b', 'it\\\'s', '<b>bold</b>', 'line\\nbreak', 'tab\\tbed', 'word '.repeat(12), 'iiiiiiiiiiii', 'WWWWWW',
  'Reserved', '(Options and Padding)', 'é', 'a-b', '0', 'mixed 中文 text', 'x\u00a0y', 'trailing   ', 'Ünïcödé', 'Window',
];
const TITLES = ['T', 'TCP Packet', 'UDP Packet', 'A title that is much wider than the packet it names, by a long way', '中文', '#35; &amp;'];

// Random packet diagrams, most of them well formed, with random configuration.
export function randomDiagrams(rnd) {
  const pick = picker(rnd);
  const chance = (p) => rnd() < p;
  const quote = (label) => (label.includes("'") || chance(0.8) ? `"${label}"` : `'${label}'`);
  return () => {
    const settings = [];
    if (chance(0.5)) {
      if (chance(0.5)) settings.push(`bitsPerRow: ${pick([1, 2, 3, 4, 7, 8, 8, 16, 16, 24, 32, 33, 64, 0, -4, 2.5, 4096])}`);
      if (chance(0.3)) settings.push(`bitWidth: ${pick([1, 5, 10, 18, 20, 24, 40, 0.5, 12.25, 400, 500])}`);
      if (chance(0.3)) settings.push(`rowHeight: ${pick([1, 10, 16, 20, 32, 50, 80, 33.3, 400])}`);
      if (chance(0.2)) settings.push(`paddingX: ${pick([0, 1, 2, 3, 10, 20, 100, 2.5])}`);
      if (chance(0.2)) settings.push(`paddingY: ${pick([0, 1, 10, 30, 7.7])}`);
      if (chance(0.3)) settings.push(`showBits: ${pick(['true', 'false', '"no"'])}`);
      if (chance(0.3)) settings.push(`bitOrder: ${pick(['descending', 'ascending', 'x'])}`);
    }
    let s = '';
    if (chance(0.1)) s += `---\ntitle: ${pick(TITLES)}\n---\n`;
    if (settings.length) {
      if (chance(0.7)) s += `---\nconfig:\n  packet:\n${settings.map((l) => '    ' + l).join('\n')}\n---\n`;
      else s += `%%{init: {"packet": {${settings.map((l) => l.replace(/^(\w+): (.*)$/, (_, k, v) => `"${k}": ${/^[-\d.]+$|^true$|^false$|^"/.test(v) ? v : `"${v}"`}`)).join(', ')}}}}%%\n`;
    }
    s += chance(0.85) ? 'packet' : 'packet-beta';
    if (chance(0.1)) s += ` title ${pick(TITLES)}`;
    s += '\n';
    if (chance(0.3)) s += `title ${pick(TITLES)}\n`;
    if (chance(0.1)) s += 'accTitle: Accessible\n';
    if (chance(0.1)) s += chance(0.5) ? 'accDescr: Described\n' : 'accDescr {\n  one\n   two\n}\n';
    let bit = 0;
    const fields = 1 + Math.floor(rnd() * (chance(0.2) ? 20 : 8));
    for (let k = 0; k < fields; k++) {
      const width = chance(0.6) ? 1 + Math.floor(rnd() * 16) : chance(0.7) ? 1 + Math.floor(rnd() * 64) : pick([0, 100, 200, 300]);
      let start = bit;
      if (chance(0.02)) start += pick([1, -1, 2]);
      const label = quote(pick(LABELS));
      const r = rnd();
      if (r < 0.4) s += `+${width}: ${label}`;
      else if (r < 0.55 || width === 1) s += width === 1 && chance(0.7) ? `${start}: ${label}` : `${start}-${start + width - 1}: ${label}`;
      else s += chance(0.03) ? `${start + width - 1}-${start}: ${label}` : `${start}-${start + width - 1}: ${label}`;
      if (chance(0.1)) s += ' %% comment';
      s += chance(0.05) ? '\r\n' : '\n';
      if (chance(0.05)) s += '\n';
      bit += width;
    }
    return s;
  };
}

function dumpBlock(b) {
  return [enc(b.start), enc(b.end), enc(b.bits), b.label];
}

export default async function ({ src, pele, write, random }) {
  const located = await locator(src);
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { packet } = await src('diagrams/packet/index.ts');
  const { parsePacket, PACKET_TOKENS } = await src('diagrams/packet/parser.ts');
  const { tokenize } = await src('diagrams/common/tokens.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  const tokens = (source) => {
    try {
      const t = tokenize(source, PACKET_TOKENS, 'packet');
      return { tokens: t.kinds.slice(0, -1).map((k, i) => [PACKET_TOKENS[k].name, source.slice(t.starts[i], t.ends[i])]) };
    } catch (e) {
      return { tokens: null };
    }
  };
  const direct = (source) => {
    try {
      const ast = parsePacket(source);
      return {
        ...tokens(source),
        ast: {
          title: enc(ast.title),
          accTitle: enc(ast.accTitle),
          accDescr: enc(ast.accDescr),
          blocks: ast.blocks.map((b) => [enc(b.start), enc(b.end), enc(b.bits), b.label]),
        },
      };
    } catch (e) {
      return { ...tokens(source), ...caught(e) };
    }
  };
  // Pele's parseSource for a packet diagram, and the raw text given to the parser alone.
  const run = (source) => {
    const out = { src: source, direct: direct(source) };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'packet') return out;
    try {
      const m = located(source, false, () => packet.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title));
      out.model = {
        title: opt(m.title),
        accTitle: opt(m.accTitle),
        accDescr: opt(m.accDescr),
        bitsPerRow: enc(m.bitsPerRow),
        rows: m.rows.map((row) => row.map(dumpBlock)),
      };
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
  const diagrams = randomDiagrams(random(21));
  for (let i = 0; i < 1500; i++) add(diagrams());
  // Mutated text; one in six is cut short, as a statement cut off by the end of the text ends
  // on a different error than one cut off by a line break.
  const rnd = random(7);
  const next = mutator([...seen].filter((s) => s.length <= 4000), FRAGMENTS, rnd);
  for (let i = 0; i < 3500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  // Random strings of tokens, where the order of the token types and their ends matter.
  const trnd = random(8);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['packet\n', 'packet ', 'packet\n0', 'packet\n+', 'packet\n0-', 'packet\n0: ', 'packet\ntitle', 'packet\naccDescr', ''];
  for (let i = 0; i < 1000; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }
  // Pele's stress cases, which stop at limits.
  for (const s of [
    'packet\n' + '0'.repeat(3000),
    'packet\n0' + '-'.repeat(3000),
    'packet\n' + '---\n'.repeat(800),
    'packet\n0: "' + 'a'.repeat(3000),
    'packet\n0: "' + '\\'.repeat(3000),
    'packet\n' + '%%{\n'.repeat(800),
    'packet\naccDescr {' + ' a\n'.repeat(1000),
    'packet\n' + ' '.repeat(3000) + '0: "a"',
    'packet\n' + '\n \t'.repeat(1000) + '0: "a"',
    'packet\n' + '%% c\n'.repeat(700) + '0: "a"',
    'packet\n' + 'title t\n'.repeat(400),
  ]) {
    seen.delete(s);
    add(s);
  }
  write('diagram/packet/testdata/pele/packet.json', { cases: all.map(run) });
}
