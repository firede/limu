// Writes examples.json, the examples the playground shows, from a checkout of Pele:
//
//   node gen.mjs /path/to/pele
//
// The examples are the diagrams of Mermaid's documentation that Pele keeps in
// tests/corpus/<name>-docs.json, one file for each type, and the examples Pele's website
// shows in their place for five types. They are chosen and titled as Pele's website does it
// in website/src/lib/examples.ts: the sources are respelled in American English, a type is
// beta when none of its examples is written without -beta, two examples of each type are
// featured by the rule of exampleGroups, and the types are sorted as its examples page sorts
// them. Every other example of a type is kept after its featured ones, in the order of the
// corpus.
//
// Choosing the featured examples draws each source with Pele, as the website does, to leave
// out those that set colors of their own, so Pele's TypeScript is loaded as it is, with
// Node's own stripping of types; Node 24 or later runs it and nothing needs to be installed.
// A source that Pele cannot draw is left out of the examples altogether: the playground lays
// every example out with limu, and Pele failing on one says that it is not a diagram limu
// should draw either.
import { execFileSync } from 'node:child_process';
import { existsSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { registerHooks, stripTypeScriptTypes } from 'node:module';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const pele = process.argv[2];
if (!pele || !existsSync(join(pele, 'src/index.ts')) || !existsSync(join(pele, 'tests/corpus'))) {
  console.error('usage: node gen.mjs /path/to/pele');
  process.exit(2);
}

// stripTypeScriptTypes warns that it is experimental each time it runs; the warning says
// nothing about the examples.
process.removeAllListeners('warning');

// An import of x.js that is not there finds x.ts, a TypeScript file is run with its types
// stripped, and a JSON file is a module with its fields as named exports, as Pele's info
// diagram imports the version from package.json.
registerHooks({
  resolve(spec, ctx, next) {
    if (spec.endsWith('.js') && ctx.parentURL?.endsWith('.ts')) {
      const ts = new URL(spec.slice(0, -3) + '.ts', ctx.parentURL);
      if (!existsSync(fileURLToPath(new URL(spec, ctx.parentURL))) && existsSync(fileURLToPath(ts))) {
        return { url: ts.href, shortCircuit: true };
      }
    }
    if (spec.endsWith('.json') && ctx.parentURL?.endsWith('.ts')) {
      return { url: new URL(spec, ctx.parentURL).href, shortCircuit: true };
    }
    return next(spec, ctx);
  },
  load(url, ctx, next) {
    if (url.endsWith('.ts')) {
      const source = stripTypeScriptTypes(readFileSync(fileURLToPath(url), 'utf8'), { mode: 'transform' });
      return { format: 'module', source, shortCircuit: true };
    }
    if (url.endsWith('.json')) {
      const value = JSON.parse(readFileSync(fileURLToPath(url), 'utf8'));
      const named = Object.keys(value).filter((key) => /^[A-Za-z_$][\w$]*$/.test(key));
      const source = `export default ${JSON.stringify(value)};\n` + named.map((key) => `export const ${key} = ${JSON.stringify(value[key])};\n`).join('');
      return { format: 'module', source, shortCircuit: true };
    }
    return next(url, ctx);
  },
});

const { render } = await import(pathToFileURL(join(pele, 'src/index.ts')).href);
const { version } = JSON.parse(readFileSync(join(pele, 'package.json'), 'utf8'));
let commit = '';
try {
  commit = execFileSync('git', ['-C', pele, 'rev-parse', '--short', 'HEAD'], { encoding: 'utf8' }).trim();
} catch {
  // A copy of Pele outside git has no commit to name.
}

// What each corpus is of, as limu's diagram.Type names it. Info is left out, as Pele's
// examples page leaves it out: it only shows a version number.
const TYPES = new Map([
  ['agentflow', 'agentflow'],
  ['architecture', 'architecture'],
  ['block', 'block'],
  ['c4', 'c4'],
  ['class', 'class'],
  ['cynefin', 'cynefin'],
  ['er', 'er'],
  ['eventmodeling', 'eventmodeling'],
  ['flowchart', 'flowchart'],
  ['gantt', 'gantt'],
  ['git', 'gitGraph'],
  ['ishikawa', 'ishikawa'],
  ['journey', 'journey'],
  ['kanban', 'kanban'],
  ['mindmap', 'mindmap'],
  ['packet', 'packet'],
  ['pie', 'pie'],
  ['quadrant', 'quadrantChart'],
  ['radar', 'radar'],
  ['railroad', 'railroad'],
  ['requirement', 'requirement'],
  ['sankey', 'sankey'],
  ['sequence', 'sequence'],
  ['state', 'state'],
  ['swimlane', 'swimlane'],
  ['timeline', 'timeline'],
  ['treemap', 'treemap'],
  ['treeview', 'treeView'],
  ['usecase', 'usecase'],
  ['venn', 'venn'],
  ['wardley', 'wardley'],
  ['xychart', 'xychart'],
]);
const UNLISTED = new Set(['info']);

// The titles of the types, as Pele's TYPE_TITLES gives them.
const TITLES = new Map([
  ['flowchart', 'Flowchart'],
  ['swimlane', 'Swimlane'],
  ['sequence', 'Sequence'],
  ['class', 'Class'],
  ['state', 'State'],
  ['er', 'Entity relationship'],
  ['gantt', 'Gantt'],
  ['git', 'Git graph'],
  ['pie', 'Pie'],
  ['mindmap', 'Mindmap'],
  ['kanban', 'Kanban'],
  ['timeline', 'Timeline'],
  ['journey', 'User journey'],
  ['quadrant', 'Quadrant chart'],
  ['xychart', 'XY chart'],
  ['requirement', 'Requirement'],
  ['sankey', 'Sankey'],
  ['architecture', 'Architecture'],
  ['c4', 'C4'],
  ['block', 'Block'],
  ['agentflow', 'Agentflow'],
  ['railroad', 'Railroad'],
  ['treeview', 'Tree view'],
  ['cynefin', 'Cynefin'],
  ['wardley', 'Wardley map'],
  ['eventmodeling', 'Event modeling'],
  ['usecase', 'Use case'],
  ['venn', 'Venn'],
  ['ishikawa', 'Ishikawa'],
  ['packet', 'Packet'],
  ['radar', 'Radar'],
  ['treemap', 'Treemap'],
]);

// What a diagram of each type is for, in a sentence.
const DESCRIPTIONS = new Map([
  ['flowchart', 'Steps of a process as boxes, joined by arrows that show where each step leads.'],
  ['swimlane', 'A process laid out in lanes, one for each person or team that does its steps.'],
  ['sequence', 'Participants side by side, with the messages they send each other in the order they are sent.'],
  ['class', 'Classes with their fields and methods, and how they inherit from and use each other.'],
  ['state', 'The states a system can be in and the events that move it from one to another.'],
  ['er', 'The entities of a data model, their attributes, and how many of one relate to how many of another.'],
  ['gantt', 'The tasks of a project as bars along a calendar, with their dates and dependencies.'],
  ['git', 'Commits on branches of a repository, and where the branches split and merge.'],
  ['pie', 'A whole divided into slices, each as large as its share.'],
  ['mindmap', 'A central idea with topics branching from it, and subtopics from those.'],
  ['kanban', 'A board of cards in columns, one column for each stage of the work.'],
  ['timeline', 'Events placed in order along a line, grouped by period.'],
  ['journey', 'The steps a user takes to do a task, scored by how each one felt.'],
  ['quadrant', 'Points placed on two axes, in a square divided into four quadrants.'],
  ['xychart', 'Bars and lines of values on a horizontal and a vertical axis.'],
  ['requirement', 'Requirements, the elements that satisfy or verify them, and how they depend on each other.'],
  ['sankey', 'Quantities flowing between nodes, each band as wide as its flow.'],
  ['architecture', 'Services and resources grouped by where they run, and how they connect.'],
  ['c4', 'A software system at one of the levels of the C4 model: its context, containers or components.'],
  ['block', 'Blocks placed on a grid of columns by hand, joined by arrows.'],
  ['agentflow', 'The work of AI agents: their tasks, tools and decisions, and how work is handed on.'],
  ['railroad', 'The rules of a grammar drawn as tracks, which a valid input follows from start to end.'],
  ['treeview', 'A hierarchy indented as a tree, as of the folders and files of a project.'],
  ['cynefin', 'Situations sorted into the domains of the Cynefin framework, from clear to chaotic.'],
  ['wardley', 'The components of a value chain, placed by how visible they are and how far they have evolved.'],
  ['eventmodeling', 'A system over time as a row of screens, commands, events and views.'],
  ['usecase', 'The actors outside a system and the use cases they take part in.'],
  ['venn', 'Sets as overlapping circles, with what they share where they overlap.'],
  ['ishikawa', 'The causes of a problem, grouped into categories along the bones of a fish.'],
  ['packet', 'The fields of a packet or header, each as wide as the bits it takes.'],
  ['radar', 'Several series of values on axes that radiate from one center.'],
  ['treemap', 'A hierarchy of values as nested rectangles, each sized by its value.'],
]);

// Pele's own examples, as its OWN gives them: they take the place, in order, of the featured
// examples of their type from Mermaid's documentation, each keeping the shape of the one it
// replaces with other labels. They are read out of examples.ts, where they are template
// literals with no substitutions.
function ownExamples() {
  const file = readFileSync(join(pele, 'website/src/lib/examples.ts'), 'utf8');
  const start = file.indexOf('const OWN = new Map([');
  const end = file.indexOf('\n]);', start);
  if (start < 0 || end < 0) throw new Error('no OWN in examples.ts');
  const body = file.slice(start, end);
  const own = new Map();
  for (const entry of body.matchAll(/\['(\w+)', \[((?:`(?:[^`\\]|\\.)*`,?\s*)+)\]\]/g)) {
    const sources = [...entry[2].matchAll(/`((?:[^`\\]|\\.)*)`/g)].map((m) => new Function('return `' + m[1] + '`')());
    own.set(entry[1], sources);
  }
  return own;
}
const OWN = ownExamples();

// The types of which only one example is featured, as Pele's SINGLE gives them.
const SINGLE = new Set(['cynefin']);

const FIRST_LIMIT = 500;
const FULLER_LIMIT = 1000;
const TITLE_LIMIT = 48;

// The lines of a source that say something, without its front matter, as Pele's statements.
const statements = (source) =>
  source.replace(/^---\n[\s\S]*?\n---\n/, '').split('\n').map((line) => line.trim()).filter((line) => line && !line.startsWith('%%'));

// Mermaid's documentation is partly in British English, and Pele's website respells its
// examples in American English, as its AMERICAN does.
const AMERICAN = [
  [/(analy)s(?=e|ing)/gi, '$1z'],
  [/(authori|populari|recogni|standardi|summari)s(?=[aei])/gi, '$1z'],
  [/(catalog)ue/gi, '$1'],
  [/(gr)e(?=y\b)/gi, '$1a'],
  [/(licen)c(?=e)/gi, '$1s'],
  [/(model)l(?=ed|ing)/gi, '$1'],
];
const american = (source) => AMERICAN.reduce((text, [british, spelling]) => text.replace(british, spelling), source);

// Some of Mermaid's documentation indents a whole example, as it sat in a block of its page.
// The playground shows the text in an editor, where that reads as a mistake, so the
// whitespace every line starts with is taken off, with the blank lines before the first.
// What is left keeps the indentation of its lines against each other, which mindmaps and
// tree views read.
const dedent = (source) => {
  const lines = source.replace(/^(?:[ \t]*\n)+/, '').split('\n');
  let common = null;
  for (const line of lines) {
    if (line.trim() === '') continue;
    const lead = /^[ \t]*/.exec(line)[0];
    if (common === null || !lead.startsWith(common)) {
      let n = 0;
      while (common !== null && n < common.length && n < lead.length && common[n] === lead[n]) n++;
      common = common === null ? lead : common.slice(0, n);
    }
  }
  return common ? lines.map((line) => (line.startsWith(common) ? line.slice(common.length) : line.trimStart())).join('\n') : lines.join('\n');
};

const sourcesOf = (corpus) =>
  Array.isArray(corpus) ? corpus.filter((source) => typeof source === 'string' && source.trim() !== '').map((source) => american(dedent(source))) : [];

// A type is beta when none of its examples is written without -beta in its keyword, as
// Pele's isBeta has it.
const isBeta = (sources) => sources.length > 0 && sources.every((source) => statements(source)[0]?.split(/\s/)[0].endsWith('-beta'));

const lines = (source) => new Set(source.split('\n').map((line) => line.trim()).filter(Boolean));

// The share of the smaller example's lines that the other one repeats, as Pele's overlap.
function overlap(a, b) {
  const first = lines(a);
  const second = lines(b);
  let shared = 0;
  for (const line of first) if (second.has(line)) shared++;
  return shared / Math.max(1, Math.min(first.size, second.size));
}

// An example sets colors of its own when Pele's SVG of it has a fill or stroke that is not
// one of its CSS variables, or its source sets theme variables or a fill or stroke, as
// Pele's ownColors has it.
const RE_OWN_PAINT = /(?:fill|stroke)(?:="|:)(?!none|currentColor|var\()/;
const RE_OWN_STYLE = /themeVariables|\b(?:fill|stroke)\s*:/;
const ownColors = (source, svg) => RE_OWN_PAINT.test(svg) || RE_OWN_STYLE.test(source);

const draw = (source) => {
  try {
    return render(source).svg;
  } catch {
    return undefined;
  }
};

// Lines that only set how a diagram is laid out, or add a commit with nothing more, which
// would title many examples alike.
const RE_SETTING = /^(?:commit|autonumber|direction\s+\w+|columns\s+\S+)$/;

// A short title for an example: the title of its front matter, else the title it gives
// itself in its body, else its first line after the keyword that is not a setting, as Pele's
// sampleLabel takes the first and the line after the keyword. A brace that opens a block at
// the end of the line is left out.
function exampleTitle(source) {
  const unquote = (text) => text.trim().replace(/^(["'])(.*)\1$/, '$2');
  const front = source.match(/^---\n[\s\S]*?^title:\s*(.+)$[\s\S]*?^---$/m)?.[1];
  const said = statements(source);
  let text = front === undefined ? undefined : unquote(front);
  if (text === undefined) {
    const keyword = said[0]?.match(/^[\w-]+\s+title\s+(.+)$/);
    if (keyword) text = unquote(keyword[1]);
  }
  for (const pattern of [/^title\s*:?\s+(.+)$/, /^accTitle\s*:\s*(.+)$/]) {
    if (text !== undefined) break;
    const line = said.find((line) => pattern.test(line));
    if (line) text = unquote(line.match(pattern)[1]);
  }
  text ??= (said.slice(1).find((line) => !RE_SETTING.test(line)) ?? said[1] ?? said[0] ?? 'Empty').replace(/\s*\{$/, '');
  return text.length > TITLE_LIMIT ? `${text.slice(0, TITLE_LIMIT - 1)}…` : text;
}

const corpusDir = join(pele, 'tests/corpus');
const types = [];
const left = [];
for (const file of readdirSync(corpusDir).filter((f) => f.endsWith('-docs.json')).sort()) {
  const name = file.slice(0, -'-docs.json'.length);
  if (UNLISTED.has(name)) continue;
  const type = TYPES.get(name);
  if (type === undefined) throw new Error(`no type of limu's for ${file}`);
  const all = sourcesOf(JSON.parse(readFileSync(join(corpusDir, file), 'utf8')));

  // As exampleGroups chooses: the fullest example of at most FIRST_LIMIT characters, and a
  // larger one of at most FULLER_LIMIT that shares less than half its lines with the first,
  // from those Pele draws, and from those that leave their colors to the theme when any do.
  const drawn = [...new Set(all)].map((source) => ({ source, svg: draw(source) }));
  for (const example of drawn) if (example.svg === undefined) left.push(`${name}: ${exampleTitle(example.source)}`);
  const usable = drawn.filter((example) => example.svg !== undefined);
  const themed = usable.filter((example) => !ownColors(example.source, example.svg));
  const choice = (themed.length > 0 ? themed : usable).map((example) => example.source);
  const bySize = [...choice].sort((a, b) => b.length - a.length);
  const first = bySize.find((source) => source.length <= FIRST_LIMIT) ?? bySize.at(-1);
  const fuller = bySize.find((source) => source !== first && source.length <= FULLER_LIMIT && overlap(source, first) < 0.5);
  const own = OWN.get(name) ?? [];
  const featured = [own[0] ?? first, SINGLE.has(name) ? undefined : (own[1] ?? fuller)].filter((source) => source !== undefined);
  for (const source of own) if (draw(source) === undefined) throw new Error(`Pele cannot draw its own ${name} example`);

  // Examples of a type that would have the same title are told apart by a number, from the
  // second of them on.
  const rest = usable.map((example) => example.source).filter((source) => !featured.includes(source));
  const seen = new Map();
  const titled = (source) => {
    const title = exampleTitle(source);
    const n = (seen.get(title) ?? 0) + 1;
    seen.set(title, n);
    return n === 1 ? title : `${title} (${n})`;
  };
  const examples = [
    ...featured.map((source) => ({ title: titled(source), featured: true, source })),
    ...rest.map((source) => ({ title: titled(source), source })),
  ];
  const title = TITLES.get(name) ?? name;
  types.push({ type, title, description: DESCRIPTIONS.get(name), beta: isBeta(all), examples });
}
types.sort((a, b) => Number(a.beta) - Number(b.beta) || a.title.localeCompare(b.title));

const out = {
  source:
    "The examples are the diagrams of Mermaid's documentation (https://mermaid.js.org, MIT License, " +
    'Copyright (c) 2014 - 2022 Knut Sveidqvist), as Pele keeps them in tests/corpus, and, where Pele\'s ' +
    "website shows its own in their place, Pele's examples (https://github.com/obsidianmd/pele, MIT License, " +
    `Copyright (c) 2026 kepano). They were chosen, titled and respelled in American English as Pele's ` +
    `website does it, from Pele ${version}${commit ? `, commit ${commit}` : ''}, by gen.mjs.`,
  types,
};
writeFileSync(new URL('./examples.json', import.meta.url), JSON.stringify(out, null, 2) + '\n');

for (const t of types) console.log(`${t.type}: ${t.examples.length} examples, ${t.examples.filter((e) => e.featured).length} featured${t.beta ? ', beta' : ''}`);
for (const l of left) console.log(`left out, as Pele cannot draw it: ${l}`);
