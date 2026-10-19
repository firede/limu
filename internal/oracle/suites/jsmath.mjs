// Math.sin, Math.cos, Math.atan2, Math.acos and Math.asin as V8 works them out, which Pele's
// charts and block diagrams compute with: on the angles and coordinates charts reach, the arithmetic charts do
// on them, random bit patterns and the special values. An angle written [k, n] is k·π/n,
// worked out as k * Math.PI / n, which keeps the fixture small; any other input is itself.

import { enc } from './parse.mjs';

const SPECIAL = [
  0, -0, Infinity, -Infinity, NaN, 1, -1, 0.5, -0.5, 2, -2, 3, -3, 10, 0.1, 5e-324, -5e-324, 2.2250738585072014e-308,
  2.225073858507201e-308, Number.MAX_VALUE, -Number.MAX_VALUE, Math.PI, -Math.PI, Math.PI / 2, -Math.PI / 2, Math.PI / 4,
  2 * Math.PI, 1e300, -1e300, 1e22, 2 ** 52, 2 ** 53, 2 ** 53 + 2, 2 ** 1023, 2 ** -1022, 2 ** -1074, 1 - 2 ** -53,
  1 + 2 ** -52, -1 - 2 ** -52, -1 + 2 ** -53, 2 ** -27, 2 ** -28, 2 ** -57, 2 ** -60, 0.4375, 2.4375, 1.1875, 0.3, 0.78125,
  2 ** 19 * Math.PI / 2, 2 ** 20 * Math.PI, 2 ** 66, 2 ** -66, 1.5,
];

export default async function ({ write, random }) {
  const next = random(41);
  const bits = new DataView(new ArrayBuffer(8));
  const anyBits = () => {
    bits.setUint32(0, Math.floor(next() * 2 ** 32));
    bits.setUint32(4, Math.floor(next() * 2 ** 32));
    return bits.getFloat64(0);
  };
  const nudge = (x, n) => {
    bits.setFloat64(0, x);
    bits.setBigUint64(0, bits.getBigUint64(0) + BigInt(n));
    return bits.getFloat64(0);
  };
  const int = (lo, hi) => lo + Math.floor(next() * (hi - lo + 1));
  const sign = () => (next() < 0.5 ? -1 : 1);
  // A number of any size, from a subnormal to near the largest.
  const anySize = (lo = -1074, hi = 1023) => sign() * (1 + next()) * 2 ** int(lo, hi);
  // A number as data in a chart has it, with a few decimals.
  const decimal = (scale, places) => Math.round((next() - 0.5) * 2 * scale * 10 ** places) / 10 ** places;
  const pick = (list) => list[Math.floor(next() * list.length)];

  // Angles, for Math.sin and Math.cos together.
  const trig = [];
  const angle = (x) => {
    const v = Array.isArray(x) ? (x[0] * Math.PI) / x[1] : x;
    trig.push([Array.isArray(x) ? x : enc(x), enc(Math.sin(v)), enc(Math.cos(v))]);
  };
  for (const x of SPECIAL) angle(x);
  for (let n = 1; n <= 12; n++) for (let k = -4 * n; k <= 4 * n; k++) angle([k, n]);
  for (let k = -360; k <= 720; k++) angle([k, 180]);
  for (let i = 0; i < 18000; i++) {
    const n = pick([int(1, 64), int(1, 400), 180, 360, int(1, 5000)]);
    angle([int(-4 * n, 4 * n), n]);
  }
  for (let i = 0; i < 2000; i++) angle([int(-(2 ** 21), 2 ** 21), pick([1, 2, 4])]); // far out, near multiples of π/2
  for (let i = 0; i < 4000; i++) {
    // What a pie chart works out: a start a quarter turn back, and a share of a whole turn.
    const total = int(1, 1000);
    const share = int(0, total);
    const start = -Math.PI / 2 + (2 * Math.PI * share) / total;
    angle(i % 2 ? start : start + (Math.PI * int(1, total)) / total);
  }
  for (let i = 0; i < 4000; i++) angle(next() * 2 * Math.PI);
  for (let i = 0; i < 2000; i++) angle(decimal(10, int(0, 3)));
  for (let i = 0; i < 1500; i++) angle(nudge(Math.PI / 4, int(-3, 3)) * sign() * int(1, 3));
  for (let i = 0; i < 3000; i++) angle(anySize());
  for (let i = 0; i < 1500; i++) angle(anySize(-40, 40));
  for (let i = 0; i < 1500; i++) angle(anySize(19, 1023));
  for (let i = 0; i < 2000; i++) angle(anyBits());

  // Math.atan2(y, x) in every quadrant.
  const atan2 = [];
  const a2 = (y, x) => atan2.push([enc(y), enc(x), enc(Math.atan2(y, x))]);
  for (const y of SPECIAL) for (const x of SPECIAL) a2(y, x);
  for (let i = 0; i < 14000; i++) a2(int(-200, 200), int(-200, 200));
  for (let i = 0; i < 6000; i++) a2(decimal(500, 2), decimal(500, 2));
  for (let i = 0; i < 4000; i++) {
    // The difference of two points, as an arrow's direction is worked out.
    const x1 = decimal(400, 3);
    const y1 = decimal(400, 3);
    a2(decimal(400, 3) - y1, decimal(400, 3) - x1);
  }
  for (let i = 0; i < 2000; i++) a2((next() - 0.5) * 2000, (next() - 0.5) * 2000);
  for (let i = 0; i < 2000; i++) a2(anySize(), anySize());
  for (let i = 0; i < 1500; i++) a2(anySize(-80, 80), anySize(-80, 80));
  for (let i = 0; i < 1000; i++) a2(anyBits(), anyBits());

  // Math.acos over [-1, 1] and beyond.
  const acos = [];
  const ac = (x) => acos.push([enc(x), enc(Math.acos(x))]);
  for (const x of SPECIAL) ac(x);
  for (let i = 0; i < 12000; i++) ac(decimal(1, int(1, 4)));
  for (let i = 0; i < 4000; i++) ac(int(-1000, 1000) / pick([1000, 999, 7, 360, int(1, 1000)]));
  for (let i = 0; i < 3000; i++) ac(sign() * (1 - next() * 2 ** -int(0, 53)));
  for (let i = 0; i < 2000; i++) ac(sign() * (0.5 + (next() - 0.5) * 2 ** -int(0, 53)));
  for (let i = 0; i < 3000; i++) ac(next() * 2 - 1);
  for (let i = 0; i < 1000; i++) ac(sign() * next() * 2 ** -int(0, 70));
  for (let i = 0; i < 1000; i++) ac(anyBits());

  // Math.asin over [-1, 1] and beyond, and on what a block diagram's self-loop around a
  // circle asks of it: a radius over the circle's width, at most 1.
  const asin = [];
  const as = (x) => asin.push([enc(x), enc(Math.asin(x))]);
  for (const x of SPECIAL) as(x);
  for (let i = 0; i < 8000; i++) as(decimal(1, int(1, 4)));
  for (let i = 0; i < 4000; i++) as(int(-1000, 1000) / pick([1000, 999, 7, 360, int(1, 1000)]));
  for (let i = 0; i < 3000; i++) as(Math.min(1, int(1, 30) / (2 * (decimal(100, 3) + 22))));
  for (let i = 0; i < 3000; i++) as(sign() * (1 - next() * 2 ** -int(0, 53)));
  for (let i = 0; i < 2000; i++) as(sign() * (0.975 + (next() - 0.5) * 2 ** -int(0, 53)));
  for (let i = 0; i < 2000; i++) as(sign() * (0.5 + (next() - 0.5) * 2 ** -int(0, 53)));
  for (let i = 0; i < 3000; i++) as(next() * 2 - 1);
  for (let i = 0; i < 1000; i++) as(sign() * next() * 2 ** -int(0, 70));
  for (let i = 0; i < 1000; i++) as(anyBits());

  write('internal/jsmath/testdata/pele/math.json', { trig, atan2, acos, asin });
}
