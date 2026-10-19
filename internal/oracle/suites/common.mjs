// What the diagrams share: generic types read as Pele's parseGenericTypes reads them, from
// cases written here and strings of tildes, commas and words made at random; and labels laid
// out by Pele's plainLabel, fitLabel and literalLabel, from texts made at random of the pieces
// those cut words between.
const WRITTEN = [
  '', 'List', 'List~int~', 'Map~K, V~', 'Map~K~, V~', 'List~List~int~~', '~a', 'a~', '~a~b~', '~~', '~~~',
  'a~b,c~d', 'a~b, c~d~', 'x,~y~,z', 'Pair~A,B~ list', ',~,~,', 'é~ö~', '中~文~,字~',
];

// Texts for the labels, written here to reach each piece fitLabel cuts between.
const LABELS = [
  '', ' ', 'a', 'Supercalifragilisticexpialidocious', 'two words', '<br>', 'a<br>b', 'a<BR />b', 'a<br\t/>b',
  '<b>bold</b>', 'x < y', ':fa-car', 'fa:fa-car icon', 'ﬂ°°35¶ß', 'ﬂ°quot¶ß', 'ﬂ°¶ß', 'ﬂ°°¶ß', '&amp;&#35;&#;&;',
  '😀😀😀😀😀😀😀😀😀😀', '中文中文中文中文中文中文', 'line\nbreak', 'tab\there', '　wide　space',
  'averyveryverylongwordindeed<br>andanotherlongwordtoo', '&ampersandwithoutend', 'é'.repeat(30),
];

const PIECES = [
  'word', 'Word', 'longerword', 'Supercalifragilistic', 'a', ' ', ' ', '  ', '　', '\t', '\n', '-', '.',
  '<br>', '<br/>', '<BR />', '<br \t/>', '<b>', '</b>', '<', '>', '<span>', 'ﬂ°°35¶ß', 'ﬂ°quot¶ß', 'ﬂ°¶ß', '¶ß',
  '&amp;', '&#35;', '&lt;', '&bogus;', '&', '#', ';', ':fa-car', 'fa:fa-car', ':fa', '😀', '𠀀', 'é', '中文', 'Ω',
];

export default async function ({ src, write, random }) {
  const { parseGenericTypes } = await src('diagrams/common/generics.ts');
  const { plainLabel, fitLabel } = await src('diagrams/common/fit.ts');
  const { literalLabel } = await src('diagrams/common/literal-label.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');
  const next = random(5);
  const parts = ['~', '~', ',', ', ', 'a', 'List', ' ', '<', 'é'];
  const all = [...WRITTEN];
  while (all.length < 2000) {
    let s = '';
    for (let n = 1 + Math.floor(next() * 10); n > 0; n--) s += parts[Math.floor(next() * parts.length)];
    all.push(s);
  }

  const pick = random(20261019);
  const texts = [...LABELS];
  while (texts.length < 600) {
    let s = '';
    for (let n = 1 + Math.floor(pick() * 12); n > 0; n--) s += PIECES[Math.floor(pick() * PIECES.length)];
    texts.push(s);
  }
  const labels = texts.map((text) => {
    const size = [12, 14, 16, 13.5][Math.floor(pick() * 4)];
    const max = [-1, 0, 5, 30, 60, 100, 4000][Math.floor(pick() * 7)];
    const face = Math.floor(pick() * 8);
    return {
      text,
      size,
      max,
      face,
      plain: plainLabel(text, metricsMeasurer, size, max, face),
      fit: fitLabel(text, metricsMeasurer, size, max, face),
      literal: literalLabel(text, metricsMeasurer, size, max, face),
    };
  });

  write('diagram/internal/common/testdata/pele/common.json', {
    generics: all.map((input) => ({ input, output: parseGenericTypes(input) })),
    labels,
  });
}
