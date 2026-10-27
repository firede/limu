// Sequence diagrams parsed as Pele parses them, from text to the tokens of its lexer and the
// SeqDb its statements fill: Mermaid's documentation and specs, as Pele's tests collect them,
// cases written here, those mutated at random with Pele's fragments, strings of fragments,
// statements built from the grammar's own shapes as Pele's oracle test builds them, and
// whole diagrams. Each case is parsed as Pele's parse does it, after the preprocessing every
// diagram goes through, and as the parser alone reads the raw text.
import { peleError } from '../lib/value.mjs';
import { lone, picker, testStrings, docs } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { locator } from '../lib/located.mjs';

// Pele's fragments, tests/sequence/oracle.test.ts.
export const FRAGMENTS = [
  ' ', '\n', ';', ',', ':', '#', '%%', '%', '@', '@{', '}', '{', '"', "'", '+', '-', '--', '()', '(', ')', '<', '>',
  '/', '\\', '|', '->>', '-->>', '->', '-->', '<<->>', '<<-->>', '-x', '--x', '-X', '-)', '--)', '-|\\', '-|/',
  '-\\\\', '-//', '--|\\', '--|/', '--\\\\', '--//', '/|-', '\\|-', '//-', '\\\\-', '/|--', '\\|--', '//--',
  '\\\\--', 'participant ', 'actor ', 'create ', 'destroy ', 'box ', 'end', 'end\n', 'loop ', 'rect ', 'opt ',
  'alt ', 'else ', 'par ', 'par_over ', 'and ', 'critical ', 'option ', 'break ', 'note ', 'Note ', 'left of ',
  'right of ', 'over ', 'links ', 'link ', 'properties ', 'details ', 'activate ', 'deactivate ', 'autonumber',
  'autonumber ', 'off', ' off\n', '1', '10 ', '.5 ', '1.25\n', '10.001', 'title ', 'title: ', 'title\n',
  'accTitle: ', 'accDescr: ', 'accDescr { ', 'accDescr{', 'sequenceDiagram', 'sequenceDiagram\n', ' as ', ' AS ',
  'as', 'wrap: ', 'nowrap: ', ':wrap:', 'A', 'B', 'Alice', 'Bob', 'x', 'é', 'ö', '日本', '\t', '\r\n', ' \n',
  ' ', ' ', 'rgb(0, 255, 0)', 'rgba(0,0,0,0.1)', 'green ', 'transparent', '{ "type": "database" }',
  '"alias": "x"', '<br/>', 'A->>B: hi', 'A-->>-B: ok\n', 'A->>+B: go\n', 'A()->>()B: c\n', '%%{', 'PARTICIPANT ',
  'End', 'LOOP ', 'Link', 'end-user', 'endpoint', '12', '.', '.5', '0.25', '\r', '\f', '\v', '_', 'e', 's', 'of',
  'left', 'right', 'note', 'over', 'link', 'links', 'title', 'box', 'loop', 'par', 'opt', 'wrap', 'no', 'acc',
  'accTitle', 'accDescr', 'Title', 'Descr', '-|', '/|', '\\|', '//', '\\\\', '<<', '>>', '<<-', '->>()', '#35;',
  '﻿', 'ſ', 'K',
  // And for the model and what is drawn: menus, configs, colors, Unicode.
  '{"a":"https://x"}', '{"class":"hot"}', '"x"', '[1,2]', ': a @ https://b', '😀', 'hsl(120 50% 50%)', 'Aqua ',
  '@{ "type": "queue" }', '@{ alias: y }', 'a long message that goes on and on past the width of a note',
];

const SLOTS = {
  actor: ['A', 'B', 'Alice', 'Bob', 'a b', 'A-B', 'end-user', 'link', 'Link', 'x', 'é', '1', '2.5', 'A#1', 'loop1', 'as'],
  arrow: [
    '->>', '-->>', '->', '-->', '<<->>', '<<-->>', '-x', '--x', '-)', '--)', '-|\\', '-|/', '-\\\\', '-//', '--|\\',
    '--|/', '--\\\\', '--//', '/|-', '\\|-', '//-', '\\\\-', '/|--', '\\|--', '//--', '\\\\--',
  ],
  mark: ['+', '-', '()'],
  central: ['()'],
  txt: [': hi', ':', ' : a b', ': wrap: x', ':nowrap:y', ': a # b', ': a; b', ': <br/>x', ': -x'],
  kwId: ['participant ', 'actor ', 'destroy ', 'activate ', 'deactivate ', 'Participant '],
  create: ['create '],
  as: [' as ', ' AS ', ' as', 'as ', '\tas\t', ' as\n'],
  text: ['Name', 'a b c', '', 'x # y', 'rgb(0, 0, 255) T', 'green', 'wrap: t', 'Aqua x'],
  config: ['@{ "type": "queue" }', '@{type: actor}', ' @{ "alias": "x" }', '@{}', '@{ a', '@{ "type": "x" } '],
  note: ['note ', 'Note '],
  place: ['left of ', 'right of ', 'over ', 'Left Of '],
  over: ['over '],
  comma: [',', ', ', ' , '],
  block: ['loop ', 'rect ', 'opt ', 'alt ', 'par ', 'par_over ', 'critical ', 'break ', 'box ', 'loop', 'alt\n'],
  divider: ['else ', 'and ', 'option ', 'else'],
  end: ['end', 'End', 'end '],
  autonumber: ['autonumber', 'autonumber ', 'Autonumber '],
  num: ['1', '10', '.5', '2.25', '1.234', '1 ', '3\t'],
  off: ['off', ' off', 'OFF'],
  menu: ['links ', 'link ', 'properties ', 'details ', 'link\t', 'links  '],
  misc: ['title T', 'title: T', 'accTitle: t', 'accDescr: d', 'accDescr { a\n b }', '%% c', '# c', 'x%%y'],
};
const TEMPLATES = [
  'actor arrow actor txt', 'actor arrow actor txt', 'actor arrow mark actor txt', 'actor central arrow actor txt',
  'actor central arrow central actor txt', 'kwId actor', 'kwId actor as text', 'kwId actor config',
  'kwId actor config as text', 'create kwId actor', 'create kwId actor as text', 'note place actor txt',
  'note over actor comma actor txt', 'block text', 'block text', 'end', 'end', 'divider text', 'autonumber',
  'autonumber num', 'autonumber num num', 'autonumber off', 'menu actor txt', 'misc',
  // Near misses: one part too many, too few, or out of place.
  'actor central arrow mark actor txt', 'actor arrow mark mark actor txt', 'actor mark arrow actor txt',
  'actor arrow txt', 'actor txt', 'actor arrow actor', 'note place actor comma actor txt', 'note actor txt',
  'kwId actor as text config', 'kwId config', 'kwId actor comma actor', 'create actor arrow actor txt',
  'create block text', 'autonumber off num', 'autonumber num off', 'autonumber num num num', 'menu txt',
  'menu actor', 'menu actor arrow actor txt', 'block text end', 'divider text end', 'end end',
].map((t) => t.split(' '));
const SLOT_NAMES = Object.keys(SLOTS);

// Cases written here, for the corners of the grammar, the model builder and the drawing.
export const WRITTEN = [
  'sequenceDiagram',
  'sequenceDiagram\n',
  'sequenceDiagram x',
  '  sequenceDiagram\nA->>B: hi',
  'sequenceDiagram\nparticipant A as Alice\nactor B as Bob\nA->>B: Hello\nB-->>A: Hi\nA-)B: async\nA--)B: async back\nA-xB: lost\nA--xB: lost too\nA->B: open\nA-->B: open dotted\nA<<->>B: both\nA<<-->>B: both dotted',
  'sequenceDiagram\nA-|\\B: a\nA-|/B: b\nA-\\\\B: c\nA-//B: d\nA--|\\B: e\nA--|/B: f\nA--\\\\B: g\nA--//B: h\nA/|-B: i\nA\\|-B: j\nA//-B: k\nA\\\\-B: l\nA/|--B: m\nA\\|--B: n\nA//--B: o\nA\\\\--B: p',
  'sequenceDiagram\nA->>A: self\nA-->>A: self dotted\nA-xA: self cross\nA-)A: self point\nA<<->>A: self both\nA-|\\A: half\nA/|-A: half back',
  'sequenceDiagram\nA->>+B: start\nB->>+B: inner\nB-->>-B: inner done\nB-->>-A: done\nactivate A\nA->>B: x\ndeactivate A',
  'sequenceDiagram\nA->>B: x\ndeactivate B',
  'sequenceDiagram\ndeactivate A',
  'sequenceDiagram\nA()->>B: from circle\nA->>()B: to circle\nA()->>()B: both\nA()->>()A: self both\nA()->>A: self from\nA->>()A: self to',
  'sequenceDiagram\nautonumber\nA->>B: one\nB->>A: two\nautonumber 10 5\nA->>B: ten\nautonumber off\nA->>B: hidden\nautonumber 1.5 .25\nA->>B: a\nA->>B: b\nA->>A: self\nA()->>()B: c\nA/|-B: d',
  'sequenceDiagram\nautonumber 0 0\nA->>B: x\nB->>A: y',
  'sequenceDiagram\nNote left of A: left\nNote right of A: right\nNote over A: over\nNote over A,B: across\nNote over B,A: back\nA->>B: x\nnote over A,C: wide note that spans past the participants it is over',
  'sequenceDiagram\nloop Every minute\nA->>B: ping\nend\nalt ok\nA->>B: yes\nelse fail\nA->>B: no\nelse\nA->>B: maybe\nend\nopt maybe\nA->>B: x\nend\npar one\nA->>B: a\nand two\nA->>C: b\nend\npar_over p\nA->>B: c\nend\ncritical c\nA->>B: d\noption o1\nA->>B: e\noption o2\nA->>B: f\nend\nbreak when\nA->>B: g\nend',
  'sequenceDiagram\nrect rgb(191, 223, 255)\nA->>B: x\nrect rgba(0, 0, 255, .1)\nB->>A: y\nend\nend\nrect\nA->>B: z\nend\nrect red\nA->>B: w\nend\nrect url(x)\nA->>B: v\nend',
  'sequenceDiagram\nloop outer\nloop inner\nA->>B: x\nend\nNote over A: n\nend',
  'sequenceDiagram\nloop\nend\nalt\nend\nloop a long condition that has to wrap past the width a frame gives its text before it grows\nA->>B: x\nend',
  'sequenceDiagram\nbox Aqua Group\nparticipant A\nparticipant B\nend\nbox rgb(33,66,99) Other\nactor C\nend\nbox transparent\nparticipant D\nend\nbox\nparticipant E\nend\nbox Just a title\nparticipant F\nend\nA->>F: across',
  'sequenceDiagram\nbox hsl(120 50% 50% / .5) H\nparticipant A\nend\nbox notacolor title\nparticipant B\nend\nbox wrap:Wrapped box title that is long\nparticipant C\nend\nbox green\nparticipant D\nend',
  'sequenceDiagram\nbox A\nparticipant X\nend\nbox B\nparticipant X\nend',
  'sequenceDiagram\nbox A\nA->>B: x\nend',
  'sequenceDiagram\nparticipant A\ncreate participant B\nA->>B: hello\ncreate actor C as Carl\nB->>C: hi\ndestroy C\nB-xC: bye\ndestroy B\nB->>A: gone',
  'sequenceDiagram\ncreate participant B\nA->>C: x',
  'sequenceDiagram\nparticipant B\ncreate participant B\nA->>B: x',
  'sequenceDiagram\ndestroy B\nA->>C: x',
  'sequenceDiagram\nA->>B: x\ncreate participant C\nA->>C: y\nC->>A: z\ndestroy A\nC->>A: w',
  'sequenceDiagram\nparticipant A@{ "type": "boundary" }\nparticipant B@{ "type": "control" }\nparticipant C@{ "type": "entity" }\nparticipant D@{ "type": "database" }\nparticipant E@{ "type": "collections" }\nparticipant F@{ "type": "queue" }\nparticipant G@{ "type": "actor" }\nparticipant H@{ "type": "nothing" }\nA->>B: a\nC->>D: b\nE->>F: c\nG->>H: d',
  'sequenceDiagram\nparticipant A@{ "type": "queue", "alias": "The queue" }\nparticipant B@{ alias: Bee } as Real\nparticipant C@{ type: database\n alias: Data }\nparticipant D@{ type: [1, 2] }\nparticipant E@{ type: 5 }\nparticipant F@{ alias: 0 }\nA->>B: x',
  'sequenceDiagram\nparticipant A@{ "type": }\nA->>B: x',
  'sequenceDiagram\nparticipant A@{ a: [ }',
  'sequenceDiagram\nparticipant A\nlinks A: {"Dashboard": "https://x", "Wiki": "https://y"}\nlink A: Mail @ mailto:a\nlink A: no at\nproperties A: {"class": "hot"}\nproperties A: [1, 2]\nproperties A: "str"\nlinks A: not json\ndetails A: el-id\nlinks Z: {"a":1}\nlink A: 😀@b\nlink A: a😀b',
  'sequenceDiagram\nparticipant A\nproperties A: {"2":"b","1":"a","x":{"n":null,"m":[1e400,-0,1.5e-7]},"x":true,"__proto__":3}\nlinks A: {"a":"\\u0041\\n\\t\\"","b":"\\ud83d\\ude00"}\nlinks A: {"a" : 1 , }\nlinks A: {"a":01}',
  'sequenceDiagram\ntitle: Legacy title\nA->>B: x',
  'sequenceDiagram\ntitle The title\naccTitle: Acc title\naccDescr: Acc descr\naccDescr {\n  line one\n    line two\n}\nA->>B: x',
  '---\ntitle: Front title\n---\nsequenceDiagram\nA->>B: x',
  '---\ntitle: A title much wider than the diagram it is over, by far, for sure, to wrap\n---\nsequenceDiagram\nA->>B: x',
  '%%{wrap}%%\nsequenceDiagram\nA->>B: a long message that wraps because the directive says so for every text\nNote over A: a long note that wraps because the directive says so',
  '---\nconfig:\n  sequence:\n    wrap: true\n    width: 100\n---\nsequenceDiagram\nA->>B: a long message that wraps because the config says so\nparticipant C as wrap:long name for a participant to wrap',
  '---\nconfig:\n  wrap: false\n  sequence:\n    wrap: true\n---\nsequenceDiagram\nA->>B: a long message that does not wrap because the top level says no',
  '---\nconfig:\n  sequence:\n    mirrorActors: false\n    rightAngles: true\n    showSequenceNumbers: true\n    messageAlign: left\n---\nsequenceDiagram\nA->>B: left aligned\nA->>A: square loop\nB-->>A: back',
  '---\nconfig:\n  sequence:\n    messageAlign: right\n    actorMargin: 100\n    boxMargin: 30\n    noteMargin: 2\n    messageMargin: 80\n    height: 80\n---\nsequenceDiagram\nbox Blue B\nA->>B: right aligned\nend\nNote over A: n',
  '---\nconfig:\n  sequence:\n    hideUnusedParticipants: true\n---\nsequenceDiagram\nparticipant A\nparticipant B\nparticipant C\nA->>C: x',
  '---\nconfig:\n  sequence:\n    actorMargin: 5000\n    width: -1\n    messageMargin: 0\n---\nsequenceDiagram\nA->>B: x',
  'sequenceDiagram\nwrap: A->>B: x',
  'sequenceDiagram\nA->>B: wrap: a long message that wraps because it says so in its text\nA->>B: nowrap: this one does not',
  'sequenceDiagram\nparticipant A as Ä Ö 日本語 😀\nA->>B: a #35; b #quot;c#quot; &amp; <b>bold</b> <br/>next line\nNote over B: x < y > z',
  'sequenceDiagram\nA->>B: x\nA->>C: across the middle participant, which a label would be struck by\nNote over A,C: note across',
  'sequenceDiagram\nend',
  'sequenceDiagram\nelse x',
  'sequenceDiagram\nloop x\nA->>B: y',
  'sequenceDiagram\nalt x\nand y\nend',
  'sequenceDiagram\nA->>B',
  'sequenceDiagram\nA->>: x',
  'sequenceDiagram\nparticipant\nA->>B: x',
  'sequenceDiagram\nparticipant A as\n',
  'sequenceDiagram\nNote A: x',
  'sequenceDiagram\nautonumber 1 2 3',
  'sequenceDiagram\nA->>B: x;B->>A: y',
  'sequenceDiagram\n%% comment\nA->>B: x %% not a comment\n# hash comment\nA->>B: y # trailing',
  'sequenceDiagram\nparticipant A #comment\nparticipant B as Bee #comment',
  'sequenceDiagram\nparticipant A<B',
  'sequenceDiagram\naccDescr {\nA->>B: x',
  'sequenceDiagram\nA ->> B : spaced\nA->>B:tight',
  'sequenceDiagram\r\nA->>B: x\r\nB->>A: y\r\n',
];

// The sources in Mermaid's specs as Pele ports them, and its documentation, as Pele's
// loadCorpus('sequence', /sequenceDiagram/i) collects them.
function specSources(pele) {
  return [...new Set([...testStrings(pele, 'tests/compat/sequence', /\.spec\.[jt]s$/, /sequenceDiagram/i), ...docs(pele, 'sequence')])];
}

export function corpus(pele) {
  return [...new Set([...specSources(pele), ...WRITTEN])].filter((s) => !lone(s));
}

// Statements built from the grammar's own shapes, with parts swapped, dropped and repeated,
// as Pele's oracle test builds them.
export function generated(rnd) {
  const pick = picker(rnd);
  return () => {
    let src = 'sequenceDiagram\n';
    for (let lines = 1 + Math.floor(rnd() * 6); lines > 0; lines--) {
      for (const slot of pick(TEMPLATES)) {
        const r = rnd();
        if (r < 0.06) continue;
        src += pick(SLOTS[r < 0.18 ? pick(SLOT_NAMES) : slot]);
        if (r > 0.94) src += pick(SLOTS[pick(SLOT_NAMES)]);
        if (rnd() < 0.3) src += ' ';
      }
      src += rnd() < 0.85 ? '\n' : pick([';', ' \n', '', '\n\n', ' ']);
    }
    return src;
  };
}

// Whole diagrams that Mermaid accepts: every kind of participant and message, notes, blocks
// nested in blocks, boxes, activations, participants created and destroyed, numbers, with
// texts of all lengths, and the settings that change what is drawn.
export function drawableDiagrams(rnd) {
  const pick = picker(rnd);
  const chance = (p) => rnd() < p;
  const texts = ['hi', 'Hello Bob, how are you?', 'ok', '', 'x', 'Ä Ö ü', '日本語のテキスト', '😀 smile', 'a #35; b #quot;c#quot;',
    'x < y & z', 'a much longer message that will need to wrap somewhere along the way to the other side', 'GET /api/v1/items?id=42',
    '<b>bold</b> and <i>it</i>', 'one<br/>two<br/>three', 'wrap: wrapped by its text, a long message to wrap here', 'nowrap: kept on one line'];
  const kinds = ['participant', 'participant', 'participant', 'actor', 'participant@boundary', 'participant@control',
    'participant@entity', 'participant@database', 'participant@collections', 'participant@queue'];
  const arrows = ['->>', '-->>', '->', '-->', '-x', '--x', '-)', '--)', '<<->>', '<<-->>', '-|\\', '-|/', '-\\\\', '-//',
    '--|\\', '--|/', '--\\\\', '--//', '/|-', '\\|-', '//-', '\\\\-', '/|--', '\\|--', '//--', '\\\\--'];
  const names = ['Alice', 'Bob', 'Carol', 'Dan', 'Eve', 'Svc', 'DB', 'Q'];
  return () => {
    let head = '';
    if (chance(0.35)) {
      head += '---\n';
      if (chance(0.4)) head += `title: ${pick(['Orders', 'A much longer title for the sequence diagram over it', 'T'])}\n`;
      if (chance(0.7)) {
        head += 'config:\n';
        if (chance(0.2)) head += `  wrap: ${pick(['true', 'false'])}\n`;
        head += '  sequence:\n';
        if (chance(0.3)) head += `    mirrorActors: ${pick(['true', 'false'])}\n`;
        if (chance(0.2)) head += '    rightAngles: true\n';
        if (chance(0.25)) head += '    showSequenceNumbers: true\n';
        if (chance(0.2)) head += `    messageAlign: ${pick(['left', 'right', 'center'])}\n`;
        if (chance(0.2)) head += `    actorMargin: ${pick([0, 10, 80, 150])}\n`;
        if (chance(0.15)) head += `    width: ${pick([20, 100, 200])}\n`;
        if (chance(0.15)) head += `    height: ${pick([10, 40, 90])}\n`;
        if (chance(0.15)) head += `    messageMargin: ${pick([0, 20, 60])}\n`;
        if (chance(0.15)) head += `    noteMargin: ${pick([0, 5, 25])}\n`;
        if (chance(0.15)) head += `    boxMargin: ${pick([0, 4, 25])}\n`;
        if (chance(0.15)) head += `    hideUnusedParticipants: ${pick(['true', 'false'])}\n`;
        if (chance(0.15)) head += '    wrap: true\n';
        head += '    useMaxWidth: true\n';
      }
      head += '---\n';
    }
    const lines = ['sequenceDiagram'];
    if (chance(0.15)) lines.push(`title ${pick(['Flow', 'The whole exchange between them'])}`);
    const n = 1 + Math.floor(rnd() * 5);
    const ids = names.slice(0, n);
    const declare = (id) => {
      const kind = pick(kinds);
      const [kw, type] = kind.split('@');
      const alias = chance(0.4) ? ` as ${pick(['Alice A.', 'The Service', 'wrap:A participant with a long name', 'Ö', id])}` : '';
      const conf = type ? `@{ "type": "${type}" }` : '';
      return `${kw} ${id}${conf}${alias}`;
    };
    const boxed = chance(0.25);
    let k = 0;
    if (boxed) {
      while (k < n) {
        const take = 1 + Math.floor(rnd() * (n - k));
        if (chance(0.6)) {
          lines.push(`box ${pick(['Aqua Front', 'rgb(200, 220, 255) Back end', 'transparent Plain', 'Only a title', '', 'green', 'rgba(0,0,0,0.1) A long box title that is wider than its members'])}`);
          for (let j = 0; j < take; j++) lines.push(declare(ids[k + j]));
          lines.push('end');
        } else for (let j = 0; j < take; j++) lines.push(declare(ids[k + j]));
        k += take;
      }
    } else {
      for (const id of ids) if (chance(0.7)) lines.push(declare(id));
    }
    if (chance(0.3)) lines.push(pick(['autonumber', 'autonumber 10 10', 'autonumber 1.5 .5', 'autonumber off']));
    const open = [];
    const active = new Map();
    let created = false;
    const statements = 2 + Math.floor(rnd() * 12);
    for (let s = 0; s < statements; s++) {
      const r = rnd();
      const a = pick(ids);
      const b = chance(0.12) ? a : pick(ids);
      if (r < 0.45) {
        let mark = '';
        const m = rnd();
        if (m < 0.12) mark = '+';
        else if (m < 0.2 && (active.get(a) ?? 0) > 0) mark = '-';
        const central = chance(0.06) ? pick(['()', '']) : '';
        const centralTo = chance(0.06) && !mark ? '()' : '';
        lines.push(`${a}${central}${pick(arrows)}${centralTo || mark}${b}: ${pick(texts)}`);
        if (mark === '+' && !centralTo) active.set(b, (active.get(b) ?? 0) + 1);
        if (mark === '-' && !centralTo) active.set(a, active.get(a) - 1);
      } else if (r < 0.6) {
        const place = pick(['left of', 'right of', 'over', 'over']);
        const who = place === 'over' && chance(0.5) && n > 1 ? `${a},${pick(ids)}` : a;
        lines.push(`Note ${place} ${who}: ${pick(texts) || 'note'}`);
      } else if (r < 0.75 && open.length < 4) {
        const kind = pick(['loop', 'alt', 'opt', 'par', 'par_over', 'critical', 'break', 'rect', 'rect']);
        const text = kind === 'rect'
          ? pick(['rgb(191, 223, 255)', 'rgba(0, 0, 255, .1)', '', 'red', 'nonsense value'])
          : pick(['Every minute', 'is ok', '', 'a long condition that goes on past where a frame gives its text room to wrap', 'x']);
        lines.push(`${kind} ${text}`);
        open.push(kind);
      } else if (r < 0.82 && open.length > 0) {
        const top = open[open.length - 1];
        const divider = top === 'alt' ? 'else' : top === 'par' || top === 'par_over' ? 'and' : top === 'critical' ? 'option' : '';
        if (divider && chance(0.6)) lines.push(`${divider} ${pick(['otherwise', '', 'a second long condition for the section below the line'])}`);
        else {
          lines.push('end');
          open.pop();
        }
      } else if (r < 0.88) {
        if (chance(0.5)) {
          lines.push(`activate ${a}`);
          active.set(a, (active.get(a) ?? 0) + 1);
        } else if ((active.get(a) ?? 0) > 0) {
          lines.push(`deactivate ${a}`);
          active.set(a, active.get(a) - 1);
        }
      } else if (r < 0.93 && !created) {
        created = true;
        const id = 'New';
        lines.push(`create ${pick(['participant', 'actor'])} ${id}${chance(0.3) ? '@{ "type": "database" }' : ''}`);
        lines.push(`${a}->>${id}: ${pick(['create', 'make one'])}`);
        if (chance(0.5)) {
          lines.push(`destroy ${id}`);
          lines.push(`${id}-x${a}: ${pick(['gone', ''])}`);
        }
      } else if (r < 0.96) {
        lines.push(`destroy ${a}`);
        lines.push(`${a}-x${b === a ? pick(ids) : b}: ${pick(['bye', ''])}`);
        ids.splice(ids.indexOf(a), 1);
        if (ids.length === 0) break;
      } else {
        lines.push(pick(['autonumber', 'autonumber off', 'autonumber 5', `properties ${a}: {"class": "hot cold"}`, `links ${a}: {"Doc": "https://x"}`]));
      }
    }
    while (open.length > 0) {
      lines.push('end');
      open.pop();
    }
    if (chance(0.1)) lines.push('accTitle: Title for screen readers', 'accDescr: A description');
    return head + lines.join('\n') + '\n';
  };
}

// A model as the oracle writes it, field by field.
function dumpDb(db) {
  const boxes = db.boxes;
  return {
    actors: [...db.actors.entries()].map(([id, a]) => ({
      id,
      box: a.box ? boxes.indexOf(a.box) : -1,
      name: a.name,
      description: a.description,
      wrap: a.wrap,
      prevActor: a.prevActor,
      nextActor: a.nextActor,
      links: [...a.links.entries()].map(([k, v]) => [k, JSON.stringify(v)]),
      properties: [...a.properties.entries()].map(([k, v]) => [k, JSON.stringify(v)]),
      details: a.details,
      type: a.type,
    })),
    created: [...db.createdActors.entries()],
    destroyed: [...db.destroyedActors.entries()],
    boxes: boxes.map((b) => ({ name: b.name, wrap: b.wrap, fill: b.fill, actorKeys: b.actorKeys })),
    messages: db.messages,
    notes: db.notes,
    title: db.title,
    accTitle: db.accTitle,
    accDescr: db.accDescr,
  };
}

export default async function ({ src, pele, write, random }) {
  const located = await locator(src);
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { sequence } = await src('diagrams/sequence/index.ts');
  const { SeqDb } = await src('diagrams/sequence/db.ts');
  const { parseSequence } = await src('diagrams/sequence/parser.ts');
  const { tokenize } = await src('diagrams/sequence/lexer.ts');
  const { TOKEN_NAMES } = await src('diagrams/sequence/tokens.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const parsed = (parseIt) => {
    try {
      return { db: dumpDb(parseIt()) };
    } catch (e) {
      return caught(e);
    }
  };
  const tokens = (text) => {
    const { types, texts, starts } = tokenize(text);
    return types.map((t, i) => [TOKEN_NAMES[t], texts[i], starts[i]]);
  };
  // A lone surrogate, which a Go string cannot hold, written by JSON.stringify as an escape.
  const loneOut = (v) => /\\ud[89a-f][0-9a-f]{2}/i.test(JSON.stringify(v));
  const run = (source) => {
    const direct = {
      tokens: tokens(source),
      ...parsed(() => {
        const db = new SeqDb();
        parseSequence(source, db);
        return db;
      }),
    };
    const out = { src: source, direct };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'sequence') return out;
    const text = encodeEntities(pre.text) + '\n';
    const piped = parsed(() => located(source, false, () => sequence.parse(text, pre.config, pre.title)));
    // Most text comes out of the pipeline as it went in, with a line feed after it, and
    // parses the same.
    const { tokens: _, ...rest } = direct;
    if (JSON.stringify(piped) === JSON.stringify(rest)) return { ...out, tokens: tokens(text), same: true };
    return { ...out, tokens: tokens(text), ...piped };
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
  // Mutated text; one in six is cut short.
  const rnd = random(7);
  const next = mutator(base, FRAGMENTS, rnd);
  for (let i = 0; i < 2500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  // Strings of fragments, where the lookaheads and word boundaries of the lexer decide.
  const frnd = random(8);
  for (let i = 0; i < 1200; i++) {
    let s = frnd() < 0.8 ? 'sequenceDiagram\n' : '';
    for (let k = 2 + Math.floor(frnd() * 14); k > 0; k--) s += FRAGMENTS[Math.floor(frnd() * FRAGMENTS.length)];
    add(s);
  }
  const gen = generated(random(9));
  for (let i = 0; i < 1500; i++) add(gen());
  const drawable = drawableDiagrams(random(10));
  for (let i = 0; i < 600; i++) add(drawable());

  const cases = all.map(run).filter((c) => !loneOut(c));
  write('diagram/sequence/testdata/pele/sequence.json', { cases });
}
