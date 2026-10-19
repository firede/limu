// What the diagrams share: generic types read as Pele's parseGenericTypes reads them, from
// cases written here and strings of tildes, commas and words made at random.
const WRITTEN = [
  '', 'List', 'List~int~', 'Map~K, V~', 'Map~K~, V~', 'List~List~int~~', '~a', 'a~', '~a~b~', '~~', '~~~',
  'a~b,c~d', 'a~b, c~d~', 'x,~y~,z', 'Pair~A,B~ list', ',~,~,', 'é~ö~', '中~文~,字~',
];

export default async function ({ src, write, random }) {
  const { parseGenericTypes } = await src('diagrams/common/generics.ts');
  const next = random(5);
  const parts = ['~', '~', ',', ', ', 'a', 'List', ' ', '<', 'é'];
  const all = [...WRITTEN];
  while (all.length < 2000) {
    let s = '';
    for (let n = 1 + Math.floor(next() * 10); n > 0; n--) s += parts[Math.floor(next() * parts.length)];
    all.push(s);
  }
  write('diagram/internal/common/testdata/pele/common.json', { generics: all.map((input) => ({ input, output: parseGenericTypes(input) })) });
}
