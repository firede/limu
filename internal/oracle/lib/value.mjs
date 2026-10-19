// Values as the fixtures hold them.

// A value as JSON can carry it exactly: an object as its entries in JavaScript's order (or
// sorted, for a config, whose order nothing reads), and the numbers JSON has no words for.
// internal/fixture reads it back.
export function enc(v, sorted = false) {
  if (v === undefined) return { $: 'undefined' };
  if (v === null || typeof v === 'boolean' || typeof v === 'string') return v;
  if (typeof v === 'number') {
    if (Number.isNaN(v)) return { $n: 'NaN' };
    if (v === Infinity) return { $n: 'Infinity' };
    if (v === -Infinity) return { $n: '-Infinity' };
    if (Object.is(v, -0)) return { $n: '-0' };
    return v;
  }
  if (Array.isArray(v)) return v.map((x) => enc(x, sorted));
  let keys = Object.keys(v);
  if (sorted) keys = keys.sort();
  return { $o: keys.map((k) => [k, enc(v[k], sorted)]) };
}

// What a PeleError says, field by field.
export function peleError(e) {
  return { code: e.code, message: e.message, type: e.type, line: e.line, column: e.column, snippet: e.snippet };
}
