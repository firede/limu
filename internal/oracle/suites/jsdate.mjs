// JavaScript's Date as V8 works it out in local time, which Pele's gantt charts compute with:
// new Date(string), the constructor from fields, the getters and the setters, in several time
// zones. Node reads process.env.TZ again when it is set, so one run goes through them all.
// The cases are dates around the changes of offset of each zone, dates at random, strings
// written here and strings made of the pieces V8's date parser reads.


export const ZONES = ['UTC', 'America/New_York', 'Europe/London', 'Asia/Kolkata', 'Australia/Lord_Howe'];

const PIECES = [
  '2024', '1999', '0001', '-000001', '+002024', '-000000', '275760', '+275760', '-271821', '24', '99', '50', '49', '0', '00',
  '01', '1', '12', '13', '31', '32', '2', '9', '123', '1234', '12345', '123456', '1234567', '0000000123', '9999999999',
  '-', '+', ':', '::', '.', ',', '/', ' ', '  ', '\t', '\n', ' ', ' ', '　', '(', ')', '(x)', '((a)b)', 'T', 't',
  'Z', 'z', 'GMT', 'UTC', 'UT', 'gmt', 'EST', 'EDT', 'CST', 'CDT', 'MST', 'MDT', 'PST', 'PDT', 'Jan', 'January', 'feb',
  'Mar', 'apr', 'May', 'jun', 'Jul', 'Aug', 'Sept', 'Oct', 'nov', 'Dec', 'Decembre', 'Mon', 'Tuesday', 'AM', 'pm', 'a', 'x',
  'foo', 'é', '日本', '😀', '\u0000', '[', '_', '@', '#', 'T10', '10:30', '23:59:59', '24:00', '12:00:00.123', '.5', '+05:30',
  '-0800', '+1', 'Z1',
];

const WRITTEN = [
  '', ' ', '2024', '2024-01', '2024-01-05', '2024-1-5', '2024-01-05T10:00', '2024-01-05T10:00:00', '2024-01-05T10:00:00Z',
  '2024-01-05T10:00:00.5Z', '2024-01-05T10:00:00.123456789Z', '2024-01-05T24:00', '2024-01-05T24:00:01', '2024-01-05T25:00',
  '2024-01-05T10:00+05:30', '2024-01-05T10:00+0530', '2024-01-05T10:00-08', '2024-01-05T10', '2024-01-05 10:00',
  '2024-01-05 10:00 Z', '2024/01/05', '01/05/2024', '1/5/2024 10:00 PM', '5 January 2024', 'January 5, 2024', 'Jan 5',
  '5', '12', '13', '32', '2024-13-01', '2024-02-30', '2024-02-31', 'Fri Jan 05 2024 10:00:00 GMT+0100 (CET)',
  'Thu, 01 Jan 1970 00:00:00 GMT', 'Thu, 01 Jan 1970 00:00:00 GMT-0500', '1970-01-01T00:00:00.000+00:00', '+275760-09-13T00:00:00Z',
  '+275760-09-13T00:00:00.001Z', '-271821-04-20T00:00:00Z', '-271821-04-19T23:59:59.999Z', '-000000-01-01T00:00:00Z',
  '+002024-01-05', '-002024-01-05', '10:00', '10:00 2024-01-05', 'Jan 2024', '2024 Jan 5', 'Jan 5 2024 10:00 am',
  'Jan 5 2024 12:00 am', 'Jan 5 2024 12:00 pm', 'Jan 5 2024 13:00 pm', '12 Jan 5', 'today', 'now', 'Invalid Date',
  '2024-03-10 02:30', '2024-03-10T02:30', '2024-11-03 01:30', '2024-03-31 01:30', '2024-10-27 01:30', '2024-03-10',
  'May 1 2001 GMT+298262:', 'Jan 1 2000 GMT+12345', 'Jan 1 2000 UTC+5', 'Jan 1 2000 GMT-8:30', 'Jan 1 2000 10:00 +5:',
  '2000-01-01T00:00:00+24:00', '1e3', '0x10', 'Jan 1 2000 (comment (nested)) 10:00', 'Jan 1 2000 10:00:00.5', 'Jan 1 2000 10.30',
  'Jan 1 2000 10:30.', '1 2 3 4', 'Jan Feb 3', '2024-01-05T10:00:00Zx', '2024-01-05x', '2024-01-05 x', 'x 2024-01-05',
  'x2024', '20240105', '2024-0105', '1900-01-01', '1850-06-01 12:00', '2100-06-01 12:00', '2038-03-14 02:30', '2060-11-07 01:30',
];

function parseCases(rnd) {
  const out = new Set(WRITTEN);
  while (out.size < 6000) {
    let s = '';
    for (let k = 1 + Math.floor(rnd() * 8); k > 0; k--) s += PIECES[Math.floor(rnd() * PIECES.length)];
    out.add(s);
  }
  // Dates as people write them, at random.
  const pad = (n, w) => String(n).padStart(w, '0');
  for (let i = 0; i < 1500; i++) {
    const y = 1800 + Math.floor(rnd() * 400);
    const m = 1 + Math.floor(rnd() * 12);
    const d = 1 + Math.floor(rnd() * 31);
    const h = Math.floor(rnd() * 25);
    const mi = Math.floor(rnd() * 60);
    const forms = [
      `${y}-${pad(m, 2)}-${pad(d, 2)}`,
      `${y}-${pad(m, 2)}-${pad(d, 2)}T${pad(h, 2)}:${pad(mi, 2)}`,
      `${y}-${pad(m, 2)}-${pad(d, 2)} ${h}:${pad(mi, 2)}`,
      `${m}/${d}/${y} ${h}:${mi}`,
      `${d} ${['Jan', 'Mar', 'Oct', 'Nov'][i % 4]} ${y} ${h}:${mi}`,
      `${y}-${m}-${d}`,
    ];
    out.add(forms[i % forms.length]);
  }
  return [...out];
}

// Instants about each change of offset in a zone, from 1840 to 2110, found by bisection, and
// others at random.
function transitions() {
  const found = [];
  const off = (t) => new Date(t).getTimezoneOffset() * 60000 + (new Date(t).getSeconds() - new Date(t).getUTCSeconds()) * 1000;
  const step = 7 * 86400000;
  for (let t = Date.UTC(1840, 0, 1); t < Date.UTC(2110, 0, 1); t += step) {
    if (off(t) === off(t + step)) continue;
    let lo = t;
    let hi = t + step;
    while (hi - lo > 1000) {
      const mid = lo + Math.floor((hi - lo) / 2000) * 1000;
      if (off(mid) === off(lo)) lo = mid;
      else hi = mid;
    }
    found.push(hi);
  }
  return found;
}

export default async function ({ write, random }) {
  const rnd = random(31);
  const strings = parseCases(random(32));
  const out = { zones: [] };
  const saved = process.env.TZ;
  for (const zone of ZONES) {
    process.env.TZ = zone;
    const value = (d) => (Number.isNaN(d.getTime()) ? null : d.getTime());
    const fields = (t) => {
      const d = new Date(t);
      return [d.getFullYear(), d.getMonth(), d.getDate(), d.getDay(), d.getHours(), d.getMinutes(), d.getSeconds(),
        d.getMilliseconds(), d.getTimezoneOffset(), d.getUTCSeconds()];
    };
    const instants = [0, -1, 1, 8.64e15, -8.64e15, 8.64e15 - 1, -8.64e15 + 1];
    const changes = transitions();
    for (const t of changes) for (const d of [-3600001, -3600000, -1800000, -1001, -1000, -1, 0, 1, 999, 1000, 1800000, 3600000]) instants.push(t + d);
    while (instants.length < changes.length * 12 + 3000) instants.push(Math.floor((rnd() * 2 - 1) * (rnd() < 0.5 ? 8.64e15 : 5e12)));

    // Local times about each change, built from fields, and fields at random.
    const built = [];
    for (const t of changes) {
      const d = new Date(t);
      for (const dh of [-3, -2, -1, 0, 1, 2]) {
        for (const mi of [0, 1, 29, 30, 59]) built.push([d.getFullYear(), d.getMonth(), d.getDate(), d.getHours() + dh, mi, 0, 0]);
      }
    }
    while (built.length < changes.length * 30 + 3000) {
      const r = rnd();
      built.push([
        r < 0.1 ? Math.floor(rnd() * 100) : r < 0.15 ? Math.floor((rnd() * 2 - 1) * 300000) : 1800 + Math.floor(rnd() * 400),
        Math.floor(rnd() * 30) - 9,
        Math.floor(rnd() * 70) - 20,
        Math.floor(rnd() * 30) - 3,
        Math.floor(rnd() * 70) - 5,
        rnd() < 0.2 ? rnd() * 100 : Math.floor(rnd() * 60),
        Math.floor(rnd() * 2000) - 500,
      ]);
    }

    // Each setter on dates about the changes and at random.
    const set = [];
    const bases = [...changes.map((t) => t - 3600000), ...changes, ...instants.slice(0, 400), NaN];
    const ops = [
      ['date', () => Math.floor(rnd() * 80) - 20],
      ['month', () => Math.floor(rnd() * 40) - 14],
      ['monthDate', () => [Math.floor(rnd() * 14) - 1, Math.floor(rnd() * 34) - 1]],
      ['fullYear', () => (rnd() < 0.2 ? Math.floor(rnd() * 100) : 1800 + Math.floor(rnd() * 400))],
      ['hours', () => [Math.floor(rnd() * 30) - 3, Math.floor(rnd() * 70) - 5, Math.floor(rnd() * 60), Math.floor(rnd() * 1000)]],
    ];
    for (const t of bases) {
      for (const [op, arg] of ops) {
        const a = arg();
        const d = new Date(t);
        if (op === 'date') d.setDate(a);
        else if (op === 'month') d.setMonth(a);
        else if (op === 'monthDate') d.setMonth(a[0], a[1]);
        else if (op === 'fullYear') d.setFullYear(a);
        else d.setHours(a[0], a[1], a[2], a[3]);
        set.push([Number.isNaN(t) ? null : t, op, a, value(d)]);
      }
    }

    out.zones.push({
      zone,
      parse: strings.map((s) => [s, value(new Date(s))]),
      fields: instants.map((t) => [t, fields(t)]),
      built: built.map((f) => [f, value(new Date(...f)), Date.UTC(...f)]),
      set,
    });
  }
  process.env.TZ = saved;
  write('internal/jsdate/testdata/pele/jsdate.json', out);
}
