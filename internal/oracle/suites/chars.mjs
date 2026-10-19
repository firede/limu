// How JavaScript reads characters and numbers, where the ports keep its meaning: Pele's table
// of letters outside ASCII, String() of numbers, Number() of strings and parseInt in base 16.
import { enc } from './parse.mjs';

export default async function ({ src, write, random }) {
  const { isUnicodeLetter } = await src('util/unicode.ts');
  // The letters, as ranges of code units.
  const letters = [];
  for (let c = 0; c <= 0xffff; c++) {
    if (!isUnicodeLetter(c)) continue;
    const last = letters[letters.length - 1];
    if (last && last[1] === c - 1) last[1] = c;
    else letters.push([c, c]);
  }

  const next = random(3);
  const numbers = [0, -0, 1, -1, 0.1, 0.5, 1e21, 1e-7, 1.5e-7, 123e-20, 1e20, 123456789012345680000, 2 ** 53, 2 ** 53 + 2,
    Number.MAX_VALUE, Number.MIN_VALUE, Infinity, -Infinity, NaN, 100, 1.25, 0.000001, 4294967295];
  for (let i = 0; i < 2000; i++) {
    const exp = Math.floor(next() * 60) - 30;
    const v = (next() - 0.5) * 10 ** exp;
    numbers.push(next() < 0.3 ? Math.round(v) : v);
  }
  const strings = ['', ' ', '0', '-0', '+1', '1.', '.5', '.', '1e', '1e5', '1E+5', '0x1F', '0X1f', '-0x1F', '0o17', '0b11', '0b2',
    '00', '007', '1_0', 'Infinity', '-Infinity', '+Infinity', 'infinity', 'NaN', '　 12 \n', '﻿3', '1e400', '0x',
    '0xFFFFFFFFFFFFFFFFFFFF', '9007199254740993', '1 2', '12px', 'é', '٣'];
  const bits = ['0', '1', '9', 'x', 'X', 'f', 'g', '.', 'e', '-', '+', ' ', ' ', '_', 'o', 'b'];
  for (let i = 0; i < 2000; i++) {
    let s = '';
    const n = Math.floor(next() * 8);
    for (let k = 0; k < n; k++) s += bits[Math.floor(next() * bits.length)];
    strings.push(s);
  }
  write('internal/chars/testdata/pele/chars.json', {
    letters,
    toString: numbers.map((n) => ({ n: enc(n), s: String(n) })),
    number: strings.map((s) => ({ s, n: enc(Number(s)), hex: enc(parseInt(s, 16)), dec: enc(parseInt(s, 10)) })),
  });
}
