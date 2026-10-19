// Math.log10, which the axis of a gantt chart takes the floor of: about powers of ten, one ulp
// at a time, and at random.
import { enc } from './parse.mjs';

export default async function ({ write, random }) {
  const buf = new DataView(new ArrayBuffer(8));
  const nudge = (x, n) => {
    buf.setFloat64(0, x);
    buf.setBigUint64(0, buf.getBigUint64(0) + BigInt(n));
    return buf.getFloat64(0);
  };
  const xs = [0, -1, 1, Infinity, NaN, 5e-324, 2.2250738585072014e-308, 1e-310, Number.MAX_VALUE];
  for (let k = -320; k <= 308; k++) for (const n of [-3, -2, -1, 0, 1, 2, 3]) xs.push(nudge(Number('1e' + k), n));
  const rnd = random(33);
  for (let i = 0; i < 20000; i++) xs.push(Math.pow(10, (rnd() * 2 - 1) * (i % 2 ? 20 : 300)) * (1 + rnd()));
  for (let i = 0; i < 5000; i++) xs.push(1 + (rnd() - 0.5) * Math.pow(2, -10 - Math.floor(rnd() * 40)));
  write('internal/jsmath/testdata/pele/log10.json', { cases: xs.map((x) => [enc(x), enc(Math.log10(x))]) });
}
