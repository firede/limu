// What Pele's treemap formatter writes of values with specifiers of d3-format: those Mermaid
// documents, specifiers put together at random from the parts its pattern reads, with fills of
// every kind and parts out of order, and values of every size and sign, halfway between two
// results, and not finite.
import { enc } from './parse.mjs';

const SPECS = [
  ',', '$0,0', '$,', ',.2f', '.1f', '.0%', ',.1%', '.3e', 'e', 'd', ',d', '~', '.2~f', '+,', '$,.2f', '.3', 'x', '', '>10,.2f',
  ' ', '(,.2f', '-$,', '.21f', '.100f', '.0e', '.20e', '.1', '.0', 'f', '%', 'g', 'F', 'D', '~%', '.2~%', '.3~e', '0>8,d',
  '*^12,.3f', '\n>,', ' >,', '😀>,', '中>,', 'é=+$010,.2~f', '<', '>>', '^', '=$', '$$', ',,', '..2f', '.f', '2.', '1,2',
  '+-,', '00,', '0', '010', '.2.3f', '~~', 'ff', ',.2fx', ' .1f', '(', '( $,.2f', 'abc', '.99999999999999999999f',
];

export default async function ({ src, write, random }) {
  const { formatter } = await src('diagrams/treemap/format.ts');
  const rnd = random(51);
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const maybe = (p, s) => (rnd() < p ? s : '');
  const specs = [...SPECS];
  for (let i = 0; i < 700; i++) {
    let s =
      maybe(0.15, pick(['', 'x', '*', ' ', '0', '>', '😀', '\n', 'é']) + pick(['<', '>', '=', '^'])) +
      maybe(0.2, pick(['-', '+', '(', ' '])) +
      maybe(0.3, '$') +
      maybe(0.1, '0') +
      maybe(0.15, String(Math.floor(rnd() * 20))) +
      maybe(0.5, ',') +
      maybe(0.5, '.' + Math.floor(rnd() * (rnd() < 0.9 ? 8 : 40))) +
      maybe(0.2, '~') +
      maybe(0.7, pick(['f', '%', 'd', 'e', 'f', 'e', 's', 'g', 'r', 'x', 'X', 'b', 'Z']));
    // Now and then, a part out of place.
    if (rnd() < 0.1) {
      const at = Math.floor(rnd() * (s.length + 1));
      s = s.slice(0, at) + pick(['$', ',', '.', '~', '+', 'f', '1', ' ']) + s.slice(at);
    }
    specs.push(s);
  }
  const values = [
    0, -0, 1, -1, 0.5, 1.5, 2.5, -2.5, 0.125, 0.005, 0.015, 1.005, 12, 123, 1234, 12345, 123456, 1234567, 999999.5, 1e6, 0.1, 0.25,
    700000, 3.14159, 1e15, 1e20, 1e21, 1.5e21, 1e100, 1e300, 1.7976931348623157e308, 1e-6, 1e-7, 1.5e-7, 5e-324, NaN, Infinity,
    -Infinity, -1234567.891, 0.000123456, 99.995, 0.1 + 0.2, 1 / 3, 2 / 3, 123456789.987654321, 2 ** 53, 2 ** 53 + 2,
  ];
  const cases = [];
  for (const spec of specs) {
    const format = formatter(spec);
    const chosen = new Set();
    while (chosen.size < 14) chosen.add(rnd() < 0.7 ? pick(values) : (rnd() - 0.3) * 10 ** Math.floor(rnd() * 30 - 10));
    for (const v of chosen) cases.push([spec, enc(v), format(v)]);
  }
  write('diagram/treemap/testdata/pele/format.json', { cases });
}
