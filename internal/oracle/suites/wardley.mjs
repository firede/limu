// Wardley maps parsed as Pele parses them: the tokens Pele's common/tokens.ts makes of the
// text with the map's table, the tree parseWardley makes of them, and the model buildWardley
// makes of the tree with the title the text's front matter gives. The cases are Mermaid's
// documentation and specs, the inputs of Pele's own tests, cases written here, random maps,
// and all of those mutated at random.
import { enc, peleError } from '../lib/value.mjs';
import { lone, picker, testStrings, docs } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { locator } from '../lib/located.mjs';

// The fragments of Pele's tests/wardley/oracle.test.ts, and more.
export const FRAGMENTS = [
  'wardley-beta', 'wardley-beta\n', 'title', 'title ', 'accTitle: ', 'accDescr: ', 'accDescr {', '}', '{', '[', ']', '(', ')', ',', '@',
  '/', '-', '--', '---', '->', '-->', '-.->', '>', '+>', '+<', '+<>', "+'x'>", "+'y'<", "+'z'<>", "+'", "'", '"', '"a b"', "'c'", ';',
  '; note', ';x', 'size', 'size [800, 600]', 'evolution', 'evolution A -> B', 'anchor', 'component', 'component A [0.5, 0.5]', 'label',
  'label [-10, 5]', 'inertia', '(inertia)', 'evolve', 'evolve A 0.9', 'pipeline', 'pipeline A {\n', 'note', 'annotations', 'annotation',
  'annotation 1,[0.5, 0.5] "t"', 'accelerator', 'deaccelerator', 'build', 'buy', 'outsource', 'market', '(buy)', '0.5', '0', '1', '12',
  '1.', '.5', '05', '1.5', '100', '0.50', '\n', '\r\n', '\r', ' ', '\t', '%%', '%% c', '%%{init: {}}%%', '---\n', 'x', 'A', 'B', 'a-b',
  'A B', '_', '9', 'é', '#35;', '&', '\\', 'Tea', 'Kettle', ' -> ', ' --> ', ' -.-> ', ' +> ', ' +<> ', '\nA -> B', '\nA --> B; l',
  ' ', ' ', '😀', '中文', 'title ', '&amp;', '<b>b</b>', '150', '99.5', '0.0', '1.0', '-5', '[0.1, 0.2]', '[1, 2]',
  '\n}\n', ' label [1, 1]', ' (build)', ' (market) inertia', '\nevolve Kettle 0.62', '\nnote "n" [0.3, 0.4]', '\npipeline Tea {\n component X [0.2]\n}\n',
];

// Seeds of Pele's tests/wardley/oracle.test.ts, which its reference accepts or refuses.
const SEEDS = [
  'wardley-beta',
  'wardley-beta title T\n',
  '\n\nwardley-beta\naccTitle: A\naccDescr {\n multi\n line\n}\ntitle T\n',
  'wardley-beta\nsize [800, 600]\nevolution "A b"@0.25 / c -> B@0.5 -> C_1@0.75 / "d e" -> D@1.0\n',
  'wardley-beta\ncomponent A [0.5, 0.5] label [-10, 5] (build) (inertia)\ncomponent B [0.1, 0.9] inertia\ncomponent "C d" [0.3, 0.3] (market)\n',
  'wardley-beta\ncomponent A [0.5, 0.5] label [10, -5] (inertia)\ncomponent _b [0.1, 0.9] (buy) inertia\ncomponent _9 [0.3, 0.3] (outsource)\n',
  'wardley-beta\ncomponent A [0.5, 0.5]\npipeline A {\n\n  component X [0.2] label [1, -2]\n\n  component "Y z" [0.8]\n}\n',
  'wardley-beta\nA -> B\nA --> B\nA -.-> B; dashed\n',
  'wardley-beta\nA +> B\nA +< B\nA +<> B\n',
  "wardley-beta\nA +'text'> B\nA +'t'< B\nA +''<> B; label\n",
  'wardley-beta\n"A" "B"\n_1 _2\nA > B\nA->B\nfoo-bar->baz\n',
  'wardley-beta\nA +> B +<; both\nA +< -> B +>\n"x y" -.-> \'z\'\n',
  'wardley-beta\nannotations [0.1, 90]\nannotation 1,[0.5, 50] "one"\nannotation 0,[1, 0] \'two\'\nnote "n" [0.2, 0.3]\n',
  'wardley-beta\naccelerator "Go" [0.5, 0.5]\ndeaccelerator Slow down [0.1, 0.2]\nevolve A 0.9\nevolve "B c" 0.10\n',
  'wardley-beta\r\ncomponent A [0.5, 0.5]\r\nA->B\r\n',
  '---\ntitle: x\n---\nwardley-beta\ncomponent A [0.5, 0.5] %% trailing\n',
  'wardley-beta\n---\nx\n---\nA->B\n',
  'wardley-beta\ncomponent market [0.1, 0.2]\n',
  'wardley-beta\ncomponent labelled thing [0.1, 0.2]\n',
  'wardley-beta\n1 2\n',
  'wardley-beta\nA B\n',
  'wardley-beta\ncomponent A [1, 2]\n',
  'wardley-beta\nevolution A / b@0.5 -> B\n',
  'wardley-beta\ncomponent A [0.5, 0.5] (inertia) (buy)\n',
];

export const TEA =
  'wardley-beta\ntitle Tea Shop\nanchor Business [0.95, 0.63]\ncomponent Cup of Tea [0.79, 0.61]\ncomponent Tea [0.63, 0.81]\n' +
  'component Kettle [0.43, 0.35]\ncomponent Power [0.10, 0.70]\nBusiness -> Cup of Tea\nCup of Tea -> Tea\nCup of Tea -> Kettle\nKettle -> Power\n' +
  'evolve Kettle 0.62\nevolve Power 0.89\nnote "Standardising power" [0.30, 0.49]\n';

const config = (lines, body) => `---\nconfig:\n  wardley-beta:\n${lines.map((l) => '    ' + l).join('\n')}\n---\n${body}`;
const W = 'wardley-beta\n';

// Cases written here: Mermaid's specs, whose texts are built from lines, Pele's render tests,
// and what the tokens, the tree and the model do at their edges.
export const WRITTEN = [
  TEA,
  ['wardley-beta', 'title Example', 'component Alpha [0.2, 0.1]', 'component Beta [0.4, 0.3]', 'Alpha -> Beta'].join('\n'),
  W + 'title Test\nevolution Genesis -> Custom -> Product -> Commodity\ncomponent A [0.5, 0.5]',
  W + 'title Test\nevolution Genesis / Concept -> Custom / Emerging -> Product / Converging -> Commodity / Accepted\ncomponent A [0.5, 0.5]',
  W + 'evolution Genesis@0.3 / Concept -> Custom@0.5 -> Product@0.9 -> Utility@1.0\ncomponent A [0.5, 0.5]\n',
  W + 'component A [0.9, 0.1] (build)\ncomponent B [0.7, 0.4] (buy) inertia\ncomponent C [0.5, 0.7] (outsource)\ncomponent D [0.3, 0.9] (market) (inertia)\n',
  W + "component A [0.9, 0.1]\ncomponent B [0.5, 0.5]\ncomponent C [0.1, 0.9]\nA +> B\nB +< C\nA +<> C\nA -.-> B; note here\nB +'cash'> C\nA -> Missing\nA -> A\n",
  W + 'component Kettle [0.45, 0.57]\ncomponent Water [0.8, 0.8]\npipeline Kettle {\n  component Campfire Kettle [0.35]\n  component Electric Kettle [0.53]\n}\nWater -> Electric Kettle\nCampfire Kettle -> Kettle\n',
  W + 'annotations [0.9, 0.1]\nannotation 2,[0.5, 0.5] "Second"\nannotation 1,[0.6, 0.65] "First"\n',
  W + 'annotation 2,[0.5, 0.5] "Second"\nannotation 1,[0.6, 0.65] "First"\n',
  W + 'component A [0.5, 0.5]\ncomponent B [0.2, 0.2]\nA -> B\n',
  W + 'component A [0.5, 0.5]\ncomponent B [0.8, 0.8]\ncomponent C [0.2, 0.8]\nA -> B\nA -> C\n',
  W + 'component A [0.5, 0.5] label [30, 30]\ncomponent B [0.8, 0.8]\nA -> B\n',
  W + 'accelerator "AI Adoption" [0.55, 0.25]\ndeaccelerator Legacy [0.15, 0.75]\n',
  W + 'size [1100, 800]\ncomponent A [0.5, 0.5]\n',
  W + 'component A long label at the right edge [0.5, 1.0]\n',
  W + 'component A [0.5, 0.5] label [-40, 30]\n',
  config(['showGrid: true'], W + 'component A [0.5, 0.5]\n'),
  '---\ntitle: Front\n---\nwardley-beta\naccTitle: Acc\naccDescr: Described\n',
  '---\ntitle: Front\n---\nwardley-beta\ntitle Own\n',
  W + 'component A [150.5, 0.5]\n',
  W + 'pipeline Nope {\n component A [0.5]\n}\n',
  W + 'note text [0.1, 0.2]',
  W + 'component A [0.5, 150.5]\n',
  W + 'component A [0.5, 100.5]\n',
  W + 'component A [100.0, 100.0]\n',
  W + 'component A [1.0, 1.0]\ncomponent B [0.0, 0.0]\ncomponent C [1.5, 1.5]\ncomponent D [99.99, 0.01]\n',
  W + 'anchor A [1.5, 0.5]\n',
  W + 'anchor A [0.5, 101.0]\n',
  W + 'note "n" [101.0, 0.5]\n',
  W + 'component P [0.5, 0.5]\npipeline P {\n component X [101.0]\n}\n',
  W + 'component A [0.5, 0.5]\nevolve A 101.0\n',
  W + 'evolve Missing 101.0\n',
  W + 'annotations [101, 0]\n',
  W + 'annotations [0, 101]\n',
  W + 'annotations [0.5, 0.5]\nannotations [101, 0]\n',
  W + 'annotation 1,[101, 0] "x"\n',
  W + 'annotation 99999999999999999999999,[101, 0] "x"\n',
  W + 'annotation 1,[' + '9'.repeat(400) + ', 0] "x"\n',
  W + 'accelerator A [101.0, 0.5]\n',
  W + 'deaccelerator A [0.5, 101.0]\n',
  W + 'size [' + '9'.repeat(400) + ', 1]\n',
  W + 'size [0, 0]\ncomponent A [0.5, 0.5]\n',
  W + 'size [100000, 20000]\ncomponent A [0.5, 0.5]\n',
  W + 'component A [0.5, 0.5]\ncomponent A [0.2, 0.3] label [5, 5]\ncomponent A [0.1, 0.1] (buy)\n',
  W + 'component A [0.5, 0.5] label [5, -5] (build) inertia\ncomponent A [0.2, 0.3]\n',
  W + 'anchor A [0.5, 0.5]\ncomponent A [0.2, 0.3]\n',
  W + 'component A [0.5, 0.5] (buy) inertia\nanchor A [0.2, 0.3]\n',
  W + 'anchor A [0.5, 0.5]\npipeline A {\n component X [0.2]\n component Y [0.6]\n}\n',
  W + 'component P [0.5, 0.5]\ncomponent P_X [0.9, 0.9] label [3, 4] (buy) inertia\npipeline P {\n component X [0.2]\n}\n',
  W + 'component P [0.5, 0.5]\npipeline P {\n component X [0.2]\n}\npipeline P {\n component Y [0.7]\n}\n',
  W + 'component P [0.5, 0.5]\ncomponent Q [0.3, 0.3]\npipeline P {\n component X [0.2]\n}\npipeline Q {\n component X [0.7]\n}\nX -> P\nX -> Q\nQ_X -> P\n',
  W + 'component A [0.5, 0.5]\ncomponent B [0.5, 0.6]\nevolve A 0.9\nevolve A 0.7\nevolve B 0.8\n',
  W + 'component A [0.5, 0.5]\nA -> B\nB -> A\nA -> A\n',
  W + "component A [0.5, 0.5]\ncomponent B [0.6, 0.6]\nA +'a<b'> B\nA +'a>b'< B\nA +'.-.'> B\nA +'' B\nA +'x'<> B; y\nA +''> B; z\n",
  W + "A +'a\nb'> B\n",
  W + 'A +<> -> B +>; x\nA +> --> B\nA -.-> B +<\n',
  W + 'A ->B\nA-> B\nA->B; \nA -> B;\nA -> B ;  spaced  \n',
  W + 'evolution A -> B\nevolution C -> D -> E\n',
  W + 'evolution A@0.1 -> B@0.2 -> C@0.3\n',
  W + 'evolution A@1.5 -> B@0.2\n',
  W + 'evolution "  a  " / "  b  " -> c\n',
  W + 'evolution A\n',
  W + 'evolution A -> \n',
  W + 'evolution A@0.1 / x@0.2 -> B\n',
  W + 'size [800, 600]\nsize [900, 700]\n',
  W + 'size [800.5, 600]\n',
  W + 'size [800 600]\n',
  W + 'size 800, 600\n',
  W + 'component A [0.5 0.5]\n',
  W + 'component A [0.5, 0.5] label [1]\n',
  W + 'component A [0.5, 0.5] label [1, 2] label [3, 4]\n',
  W + 'component A [0.5, 0.5] label [- 1, -0]\n',
  W + 'component A [0.5, 0.5] label [-0, 0]\n',
  W + 'component A [0.5, 0.5] label [1.5, 2]\n',
  W + 'component A [0.5, 0.5] (build) (buy)\n',
  W + 'component A [0.5, 0.5] (inertia\n',
  W + 'component A [0.5, 0.5] (x)\n',
  W + 'component A [0.5, 0.5] inertia inertia\n',
  W + 'component A [0.5, 0.5] build\n',
  W + 'component builder [0.5, 0.5]\n',
  W + 'component Builder [0.5, 0.5]\n',
  W + 'component markets [0.5, 0.5]\n',
  W + 'component "market" [0.5, 0.5] (market)\n',
  W + 'component A (x) & B-c [0.5, 0.5]\n',
  W + 'component A->B [0.5, 0.5]\n',
  W + 'component A-B [0.5, 0.5]\n',
  W + 'component A- [0.5, 0.5]\n',
  W + 'component A\t\tB [0.5, 0.5]\n',
  W + 'component A 1 [0.5, 0.5]\n',
  W + 'component A (b) [0.5, 0.5]\n',
  W + 'component _a-b- [0.5, 0.5]\n',
  W + 'component 9lives [0.5, 0.5]\n',
  W + 'component title thing [0.5, 0.5]\n',
  W + 'component titles [0.5, 0.5]\n',
  W + 'component accTitles [0.5, 0.5]\n',
  W + 'component Title [0.5, 0.5]\n',
  W + 'component x title [0.5, 0.5]\n',
  W + 'title',
  W + 'title\n',
  W + 'title T %% c\n',
  W + 'title T\n',
  W + 'titleT\n',
  W + 'accTitle: A\naccTitle:   B   \n',
  W + 'accTitle A\n',
  W + 'accDescr {\n a\n b\n}\n',
  W + 'accDescr {\n a\n',
  W + 'accDescr {\naccDescr {\naccDescr: x\n',
  W + 'accDescr: one\naccDescr {\n two\n}\n',
  W + 'accDescr :one',
  W + 'note "a" [0.1, 0.2]\nnote \'b\' [0.3, 0.4]\nnote "c\\"d" [0.5, 0.5]\n',
  W + 'note a [0.1, 0.2]\n',
  W + 'note "a"\n',
  W + 'annotation 1, [0.5, 0.5] "x"\n',
  W + 'annotation 1,[0.5, 0.5]\n',
  W + 'annotation 1,[50, 50] "x"\nannotation 1,[60, 60] "y"\nannotation 3,[0, 0] ""\n',
  W + 'annotation 01,[50, 50] "x"\n',
  W + 'annotations [0.5]\n',
  W + 'annotations [50, 0.5]\n',
  W + 'accelerator A [1, 2]\n',
  W + 'accelerator [0.1, 0.2]\n',
  W + 'deaccelerator "a b" [0.1, 0.2]\n',
  W + 'evolve A\n',
  W + 'evolve A 1\n',
  W + 'evolve "A" 0.5\n',
  W + 'pipeline A {\n}\n',
  W + 'component A [0.5, 0.5]\npipeline A {\n}\n',
  W + 'component A [0.5, 0.5]\npipeline A { component X [0.2]\n}\n',
  W + 'component A [0.5, 0.5]\npipeline A {\n component X [0.2] }\n',
  W + 'component A [0.5, 0.5]\npipeline A {\n component X [0.2]\n} x\n',
  W + 'component A [0.5, 0.5]\npipeline A {\n component X [0.2, 0.3]\n}\n',
  W + 'component A [0.5, 0.5]\npipeline A {\n component X [2]\n}\n',
  W + 'component A [0.5, 0.5]\npipeline A {\n component X [0.2] label [-3, 4]\n component X [0.6]\n}\n',
  W + 'component A [0.5, 0.5]\npipeline A {\n component X [0.2]\n',
  W + 'component A [0.5, 0.5]\npipeline A\n',
  W + 'component A [0.5, 0.5] %% c\n%% d\n  %% e\n%%{init: {}}%%\nA -> B %% f\n',
  W + 'A -> B %%{x}%%\n',
  '%%{init: {"theme": "dark"}}%%\nwardley-beta\ncomponent A [0.5, 0.5]\n',
  '---\nconfig:\n  wardley-beta:\n    width: 500\n    height: 400\n---\nwardley-beta\ncomponent A [0.5, 0.5]\n',
  'wardley-beta\n---\n---\n',
  'wardley-beta\n---\n',
  'wardley-beta\n--- x\n',
  'wardley-beta\n-',
  'wardley-beta\n- ->',
  'wardley-beta\n1.5 -> 2',
  'wardley-beta\n1. -> 2',
  'wardley-beta\nA -> 1',
  'wardley-beta\n1 -> A',
  'wardley-beta\n12 -> A',
  'wardley-beta\nA ->',
  'wardley-beta\nA',
  'wardley-beta\n->',
  'wardley-beta\n[',
  'wardley-beta\né',
  'wardley-beta\nA -> é',
  'wardley-beta x',
  'wardley-betax',
  'wardley-beta-beta',
  'wardley',
  'wardley-beta\nwardley-beta',
  '  wardley-beta  \n  component A [0.5, 0.5]  \n',
  '\twardley-beta\n',
  'wardley-beta %% c\n',
  'wardley-beta\r\ncomponent A [0.5, 0.5]\rA -> B',
  'wardley-beta\n component A [0.5, 0.5]\n',
  'wardley-beta\ncomponent A [0.5, 0.5]　\n',
  'wardley-beta\ncomponent "😀 中文" [0.5, 0.5]\n',
  'wardley-beta\ncomponent "#35; &amp; <b>b</b>" [0.5, 0.5]\n',
  'wardley-beta\ncomponent A [0.5, 0.5]\ncomponent B [0.5, 0.5]\nA -> B\n',
  '',
  'not a map',
];

// The corpus the suites start from: documentation, specs, the strings of Pele's tests, its
// seeds and the cases above.
export function corpus(pele) {
  return [
    ...new Set([
      ...docs(pele, 'wardley'),
      ...testStrings(pele, 'tests/compat/wardley', /\.spec\.[jt]s$/, /wardley-beta/),
      ...testStrings(pele, 'tests/wardley', /\.test\.[jt]s$/, /wardley-beta/),
      ...testStrings(pele, 'tests/render', /^wardley.*\.test\.ts$/, /wardley-beta/),
      ...SEEDS,
      ...WRITTEN,
    ]),
  ].filter((s) => !lone(s) && s.length <= 50000);
}

// Pele's stress cases, made smaller: what makes a pattern scan again, or a map large.
export function hostile() {
  const N = 4000;
  const repeat = (count, line) => Array.from({ length: count }, (_, i) => line(i)).join('');
  const frac = (i) => (0.001 * (i % 1000)).toFixed(3);
  return [
    W + 'component A [0.5, 0.5]\n' + '0'.repeat(N),
    W + 'component A [0.5, 0.5]\n' + '1.'.repeat(N / 2),
    W + repeat(200, (i) => `component c${i} [${frac(i)}, ${frac(i * 7)}]\n`),
    W + repeat(150, (i) => `component c${i} [${frac(i * 13)}, ${frac(i * 7)}]\n`) + repeat(150, (i) => `c${i} -> c${i + 1}\n`),
    W + repeat(50, (i) => `component c${i} [0.5, ${frac(i * 17)}]\n`) + repeat(100, (i) => `u${i} -> v${i}\n`),
    W + 'component A [0.9, 0.1]\ncomponent B [0.1, 0.9]\n' + repeat(100, (i) => `A +<> B; label ${i}\n`),
    W + 'component A [0.5, 0.5]\n'.repeat(300),
    W + 'component P [0.5, 0.5]\npipeline P {\n' + repeat(200, (i) => ` component p${i} [${frac(i * 3)}]\n`) + '}\n',
    W + repeat(100, (i) => `component P${i} [${frac(i * 9)}, 0.5]\npipeline P${i} {\n component x [0.2]\n}\n`),
    W + 'annotations [0.9, 0.1]\n' + repeat(100, (i) => `annotation ${i},[${frac(i * 11)}, 0.5] "text ${i}"\n`),
    W + repeat(70, (i) => `note "n${i}" [0.5, ${frac(i * 13)}]\naccelerator a${i} [0.2, ${frac(i * 13)}]\ndeaccelerator d${i} [0.8, ${frac(i * 13)}]\n`),
    W + repeat(100, (i) => `component c${i} [0.5, 0.1]\n`) + repeat(100, (i) => `evolve c${i} 0.9\n`),
    W + 'evolution ' + repeat(300, (i) => `s${i}@0.5 -> `) + 'end\n',
    W + 'component ' + 'a'.repeat(N) + ' [0.5, 0.5]',
    W + 'component ' + 'ab '.repeat(N / 4) + '[0.5, 0.5]',
    W + 'component "' + 'word '.repeat(N / 5) + '" [0.5, 0.5]',
    W + 'note "' + 'word '.repeat(N / 5) + '" [0.5, 0.5]',
    W + 'component A [0.1, 0.1]\ncomponent B [0.9, 0.9]\nA -> B; ' + 'x'.repeat(N),
    W + ('a' + ' '.repeat(99)).repeat(N / 100) + '1',
    W + ('a' + ' '.repeat(49) + '1' + ' '.repeat(49)).repeat(N / 100),
    W + '1' + '-'.repeat(N) + '!',
    W + ('1' + '-'.repeat(99) + ' ').repeat(N / 101),
    W + '-'.repeat(N),
    W + '---\n'.repeat(N / 4),
    W + "A +'x\n".repeat(N / 6),
    W + "A +'" + 'x'.repeat(N),
    W + 'component "a\n'.repeat(N / 13),
    W + 'accDescr {\n'.repeat(N / 11),
    W + ('accTitle' + ' '.repeat(40)).repeat(N / 48),
    W + '%%{\n'.repeat(N / 4),
    W + 'size [' + '9'.repeat(N) + ', 1]',
    W + 'component A [0.' + '5'.repeat(N) + ', 0.5]',
    W + ('1'.repeat(98) + '. ').repeat(N / 100),
    W + '\n'.repeat(N),
    W + 'component A [0.5, 0.5] %% c\n'.repeat(N / 28),
    W + 'componentannotationsevolve'.repeat(N / 26),
  ];
}

const NAMES = [
  'A', 'B', 'C', 'Tea', 'Cup of Tea', 'Hot Water', 'Kettle', 'Power', 'Business', 'Public', 'User', 'Data (raw)', 'R&D', 'a-b', '_x',
  'Compute', 'Electric Kettle', 'Campfire Kettle', 'Very long component name that goes on', 'x1', 'Payments', 'Search', 'Cloud',
];
const QUOTED = ['"Quoted name"', "'single'", '"😀 emoji"', '"中文名字"', '"a\\"b"', '"#35; &amp;"', '""', '"two\\nlines"', '"  spaced  "'];
const STAGES = ['Genesis', 'Custom Built', 'Product', 'Commodity', 'Concept', 'Emerging', 'Utility', '"Quoted stage"', 'X'];

// Random Wardley maps, most of them well formed, with random configuration.
export function randomMaps(rnd) {
  const pick = picker(rnd);
  const chance = (p) => rnd() < p;
  const frac = () => (chance(0.1) ? pick(['0.0', '1.0', '0.5', '0.99', '0.01']) : (rnd() * 0.98 + 0.01).toFixed(chance(0.7) ? 2 : 3));
  const pct = () => (chance(0.5) ? frac() : (rnd() * 99 + 0.5).toFixed(1));
  return () => {
    const settings = [];
    if (chance(0.25)) {
      if (chance(0.5)) settings.push(`width: ${pick([300, 500, 640, 800, 900, 1200, 150, 20000, -5, 0.5, '"x"'])}`);
      if (chance(0.5)) settings.push(`height: ${pick([200, 300, 400, 600, 800, 150, 20000, '"x"'])}`);
      if (chance(0.5)) settings.push(`showGrid: ${pick(['true', 'false', '"true"', '1'])}`);
    }
    let s = '';
    if (chance(0.1)) s += `---\ntitle: ${pick(['Front', 'A front matter title that is long enough to wrap somewhere', '中文'])}\n---\n`;
    if (settings.length) {
      if (chance(0.7)) s += `---\nconfig:\n  wardley-beta:\n${settings.map((l) => '    ' + l).join('\n')}\n---\n`;
      else s += `%%{init: {"wardley-beta": {${settings.map((l) => l.replace(/^(\w+): (.*)$/, (_, k, v) => `"${k}": ${v}`)).join(', ')}}}}%%\n`;
    }
    s += 'wardley-beta\n';
    if (chance(0.5)) s += `title ${pick(['Tea Shop', 'Map', 'A rather long title for a Wardley map, long enough to wrap', '#35; &amp;'])}\n`;
    if (chance(0.1)) s += 'accTitle: Accessible\n';
    if (chance(0.1)) s += chance(0.5) ? 'accDescr: Described\n' : 'accDescr {\n  one\n   two\n}\n';
    if (chance(0.2)) s += `size [${pick([400, 600, 800, 1100, 1200, 300, 100, 20000])}, ${pick([300, 400, 600, 800, 100, 20000])}]\n`;
    if (chance(0.3)) {
      const k = 2 + Math.floor(rnd() * 4);
      const bounded = chance(0.5);
      let edge = 0;
      const parts = [];
      for (let i = 0; i < k; i++) {
        edge = i === k - 1 && chance(0.7) ? 1 : Math.min(1, edge + rnd() * 0.5);
        let stage = pick(STAGES);
        if (bounded || chance(0.1)) stage += `@${edge.toFixed(2)}`;
        if (chance(0.2)) stage += ` / ${pick(STAGES)}`;
        parts.push(stage);
      }
      s += `evolution ${parts.join(' -> ')}\n`;
    }
    const names = [];
    const nodes = Math.floor(rnd() * (chance(0.2) ? 18 : 9));
    for (let i = 0; i < nodes; i++) {
      const name = chance(0.1) ? pick(QUOTED) : chance(0.7) ? pick(NAMES) : `N${i}`;
      names.push(name.replace(/^["']|["']$/g, ''));
      if (chance(0.12)) {
        s += `anchor ${name} [${pct()}, ${pct()}]\n`;
        continue;
      }
      s += `component ${name} [${pct()}, ${pct()}]`;
      if (chance(0.15)) s += ` label [${pick(['', '-'])}${Math.floor(rnd() * 60)}, ${pick(['', '-'])}${Math.floor(rnd() * 60)}]`;
      if (chance(0.25)) s += ` (${pick(['build', 'buy', 'outsource', 'market'])})`;
      if (chance(0.15)) s += pick([' inertia', ' (inertia)']);
      s += '\n';
    }
    if (names.length && chance(0.2)) {
      const parent = pick(names);
      s += `pipeline ${/^[A-Za-z_][\w ]*$/.test(parent) ? parent : `"${parent}"`} {\n`;
      for (let k = 1 + Math.floor(rnd() * 4); k > 0; k--) {
        const child = chance(0.5) ? pick(NAMES) : `P${k}`;
        names.push(child);
        s += `  component ${child} [${frac()}]${chance(0.15) ? ` label [${Math.floor(rnd() * 20)}, -${Math.floor(rnd() * 20)}]` : ''}\n`;
      }
      s += '}\n';
    }
    const ref = (n) => (/^[A-Za-z_][\w ]*$/.test(n) && !/^(title|acc)/.test(n) ? n : `"${n.replace(/"/g, '\\"')}"`);
    for (let k = names.length ? Math.floor(rnd() * (names.length + 3)) : 0; k > 0; k--) {
      const from = chance(0.05) ? 'Missing' : pick(names);
      const to = pick(names);
      let arrow = pick(['->', '->', '->', '-->', '-.->', '>', "+'flow'>", "+'cash'<", "+'both'<>"]);
      let line = `${ref(from)}${chance(0.1) ? ` ${pick(['+>', '+<', '+<>'])}` : ''} ${arrow} ${ref(to)}`;
      if (chance(0.1)) line += ` ${pick(['+>', '+<', '+<>'])}`;
      if (chance(0.2)) line += `; ${pick(['label', 'a longer label on a link', 'x', '中文'])}`;
      s += line + '\n';
    }
    for (let k = names.length && chance(0.3) ? 1 + Math.floor(rnd() * 3) : 0; k > 0; k--) s += `evolve ${ref(pick(names))} ${frac()}\n`;
    if (chance(0.15)) s += `note "${pick(['Standardising power', 'A note', 'A rather long note that should wrap on a narrow map, by some way'])}" [${frac()}, ${frac()}]\n`;
    if (chance(0.15)) {
      if (chance(0.5)) s += `annotations [${frac()}, ${frac()}]\n`;
      for (let k = 1 + Math.floor(rnd() * 3); k > 0; k--) s += `annotation ${Math.floor(rnd() * 5)},[${frac()}, ${frac()}] "${pick(['First', 'Second', 'A longer annotation text', ''])}"\n`;
    }
    if (chance(0.1)) s += `accelerator ${pick(['Go', '"AI Adoption"', 'Fast'])} [${frac()}, ${frac()}]\n`;
    if (chance(0.1)) s += `deaccelerator ${pick(['Legacy', '"Slow down"'])} [${frac()}, ${frac()}]\n`;
    return s;
  };
}

const opt = (x) => (x === undefined || x === '' ? undefined : x);

function dumpLabel(l) {
  return l === undefined ? enc(undefined) : [l.negX, enc(l.offsetX), l.negY, enc(l.offsetY)];
}

// The tree parseWardley makes, with every part Pele may leave undefined written so.
export function dumpAst(ast) {
  return {
    title: enc(ast.title),
    accTitle: enc(ast.accTitle),
    accDescr: enc(ast.accDescr),
    size: ast.size ? [enc(ast.size.width), enc(ast.size.height)] : enc(undefined),
    evolution: ast.evolution ? ast.evolution.stages.map((s) => [s.name, enc(s.boundary), enc(s.secondName)]) : enc(undefined),
    anchors: ast.anchors.map((a) => [a.name, enc(a.visibility), enc(a.evolution)]),
    components: ast.components.map((c) => [c.name, enc(c.visibility), enc(c.evolution), dumpLabel(c.label), enc(c.decorator?.strategy), c.inertia]),
    links: ast.links.map((l) => [l.from, enc(l.fromPort), enc(l.arrow), l.to, enc(l.toPort), enc(l.linkLabel)]),
    evolves: ast.evolves.map((e) => [e.component, enc(e.target)]),
    pipelines: ast.pipelines.map((p) => [p.parent, p.components.map((c) => [c.name, enc(c.evolution), dumpLabel(c.label)])]),
    notes: ast.notes.map((n) => [n.text, enc(n.visibility), enc(n.evolution)]),
    annotations: ast.annotations.map((a) => [enc(a.x), enc(a.y)]),
    annotation: ast.annotation.map((a) => [enc(a.number), enc(a.x), enc(a.y), a.text]),
    accelerators: ast.accelerators.map((a) => [a.name, enc(a.x), enc(a.y)]),
    deaccelerators: ast.deaccelerators.map((a) => [a.name, enc(a.x), enc(a.y)]),
  };
}

// The model buildWardley makes.
export function dumpModel(m) {
  return {
    title: enc(opt(m.title)),
    accTitle: enc(opt(m.accTitle)),
    accDescr: enc(opt(m.accDescr)),
    nodes: m.nodes.map((n) => [
      n.id, n.label, enc(n.x), enc(n.y), enc(n.className), enc(n.labelOffsetX), enc(n.labelOffsetY), enc(n.inPipeline),
      enc(n.isPipelineParent), enc(n.inertia), enc(n.sourceStrategy),
    ]),
    links: m.links.map((l) => [l.source, l.target, enc(l.dashed), enc(l.label), enc(l.flow)]),
    trends: m.trends.map((t) => [t.nodeId, enc(t.targetX), enc(t.targetY)]),
    pipelines: m.pipelines.map((p) => [p.nodeId, p.componentIds]),
    annotations: m.annotations.map((a) => [enc(a.number), a.coordinates.map((c) => [enc(c.x), enc(c.y)]), enc(a.text)]),
    notes: m.notes.map((n) => [n.text, enc(n.x), enc(n.y)]),
    accelerators: m.accelerators.map((a) => [a.name, enc(a.x), enc(a.y)]),
    deaccelerators: m.deaccelerators.map((a) => [a.name, enc(a.x), enc(a.y)]),
    annotationsBox: m.annotationsBox ? [enc(m.annotationsBox.x), enc(m.annotationsBox.y)] : enc(undefined),
    axes: [enc(m.axes.xLabel), enc(m.axes.yLabel), enc(m.axes.stages), enc(m.axes.stageBoundaries)],
    size: m.size ? [enc(m.size.width), enc(m.size.height)] : enc(undefined),
  };
}

export default async function ({ src, pele, write, random }) {
  const located = await locator(src);
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { wardley } = await src('diagrams/wardley/index.ts');
  const { parseWardley, WARDLEY_TOKENS } = await src('diagrams/wardley/parser.ts');
  const { tokenize } = await src('diagrams/common/tokens.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const tokens = (source) => {
    try {
      const t = tokenize(source, WARDLEY_TOKENS, 'wardley');
      return { tokens: t.kinds.slice(0, -1).map((k, i) => [WARDLEY_TOKENS[k].name, source.slice(t.starts[i], t.ends[i])]) };
    } catch (e) {
      return { tokens: null };
    }
  };
  const direct = (source) => {
    try {
      return { ...tokens(source), ast: dumpAst(parseWardley(source)) };
    } catch (e) {
      return { ...tokens(source), ...caught(e) };
    }
  };
  // Pele's parseSource for a Wardley map, and the raw text given to the parser alone.
  const run = (source) => {
    const out = { src: source, direct: direct(source) };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'wardley') return out;
    try {
      out.model = dumpModel(located(source, false, () => wardley.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title)));
    } catch (e) {
      Object.assign(out, caught(e));
    }
    return out;
  };

  const all = [];
  const seen = new Set();
  const add = (s) => {
    if (s.length <= 6000 && !lone(s) && !seen.has(s)) {
      seen.add(s);
      all.push(s);
    }
  };
  corpus(pele).forEach(add);
  const maps = randomMaps(random(51));
  for (let i = 0; i < 1500; i++) add(maps());
  // Mutated text; one in six is cut short, as a statement cut off by the end of the text ends
  // on a different error than one cut off by a line break.
  const rnd = random(52);
  const next = mutator([...seen].filter((s) => s.length <= 3000), FRAGMENTS, rnd);
  for (let i = 0; i < 3500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  // Random strings of tokens, where the order of the token types and their ends matter.
  const trnd = random(53);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['wardley-beta\n', 'wardley-beta ', 'wardley-beta\ncomponent ', 'wardley-beta\nA ', 'wardley-beta\nevolution ', 'wardley-beta\nA +', ''];
  for (let i = 0; i < 1500; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }
  for (const s of hostile()) add(s);
  write('diagram/wardley/testdata/pele/wardley.json', { cases: all.map(run) });
}
