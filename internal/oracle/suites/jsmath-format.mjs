// What V8's toFixed, toExponential and toPrecision write of numbers, which a treemap's
// values are formatted with: the numbers about powers of ten and halfway between two results
// of a precision, where rounding differs most easily, numbers of every exponent made from
// random bits, and those JSON has no words for.
import { enc } from '../lib/value.mjs';

export default async function ({ write, random }) {
  const rnd = random(41);
  const xs = [0, -0, NaN, Infinity, -Infinity, 1, -1, 0.5, 1.5, 2.5, -2.5, 0.05, 0.15, 0.25, 0.35, 1.005, 1.255, 9.995, 99.995];
  xs.push(Number.MAX_VALUE, Number.MIN_VALUE, -Number.MIN_VALUE, Number.EPSILON, Number.MAX_SAFE_INTEGER, 2 ** 53 + 2, 1e21, 1e21 - 65536, 999999999999999999999);
  for (let e = -330; e <= 310; e++) {
    const p = Number(`1e${e}`);
    xs.push(p, p * (1 - Number.EPSILON), p * (1 + Number.EPSILON), -p, 5 * p, 9.5 * p, 9.95 * p, 9.999999 * p);
  }
  // Halves of every number of digits, which round up in both directions of a sign.
  for (let d = 0; d <= 20; d++) {
    for (let k = 0; k < 20; k++) {
      const n = Math.floor(rnd() * 10 ** Math.min(d + 1, 15));
      xs.push((n + 0.5) / 10 ** d, -(n + 0.5) / 10 ** d, (2 * n + 1) / 2 ** Math.min(d, 60));
    }
  }
  const view = new DataView(new ArrayBuffer(8));
  for (let i = 0; i < 6000; i++) {
    view.setUint32(0, Math.floor(rnd() * 2 ** 32));
    view.setUint32(4, Math.floor(rnd() * 2 ** 32));
    xs.push(view.getFloat64(0));
  }
  for (let i = 0; i < 3000; i++) {
    const scale = 10 ** Math.floor(rnd() * 40 - 20);
    xs.push(Math.round(rnd() * 1e6) * scale, rnd() * scale, -rnd() * 1e5);
  }
  const cases = [];
  let k = 0;
  for (const x of xs) {
    // Each number with a few numbers of digits, the most a treemap uses, and beyond.
    for (const [f, p] of [[k % 21, 1 + (k % 21)], [(k * 7) % 101, 1 + ((k * 13) % 100)], [6, 12]]) {
      cases.push([enc(x), f, x.toFixed(f), x.toExponential(f), p, x.toPrecision(p)]);
    }
    k++;
  }
  write('internal/jsmath/testdata/pele/format.json', { cases });
}
