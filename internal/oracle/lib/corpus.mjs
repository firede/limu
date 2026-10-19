// What the suites take their cases from: the sources in Pele's tests and documents, as Pele's
// own tests collect them, and choices made at random.
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

// The string literals in JavaScript source: in backquotes, which may span lines, or in single
// or double quotes, which may not.
const LITERAL = /`((?:[^`\\]|\\.)*)`|'((?:[^'\\\n]|\\.)*)'|"((?:[^"\\\n]|\\.)*)"/g;

// What the body of a string literal evaluates to, as a template literal with nothing put in
// it; undefined where it is not one.
export function unquote(raw) {
  try {
    return new Function('return `' + raw.replace(/`/g, '\\`').replace(/\$\{/g, '\\${') + '`')();
  } catch {
    return undefined;
  }
}

// The string literals of source that match keyword, evaluated, in the order they are in, as
// Pele's loadCorpus finds them.
export function literals(source, keyword) {
  const out = [];
  for (const m of source.matchAll(LITERAL)) {
    const raw = m[1] ?? m[2] ?? m[3];
    if (!keyword.test(raw)) continue;
    const text = unquote(raw);
    if (text !== undefined) out.push(text);
  }
  return out;
}

// The literals of a file that match keyword, as literals finds them.
export const fileLiterals = (file, keyword) => literals(readFileSync(file, 'utf8'), keyword);

// The paths of the files in the folder dir of Pele's whose names match files, in the order
// readdirSync gives them, or sorted.
export function testFiles(pele, dir, files, sorted = false) {
  const names = readdirSync(join(pele, dir)).filter((f) => files.test(f));
  if (sorted) names.sort();
  return names.map((f) => join(pele, dir, f));
}

// The literals that match keyword in the files of a folder of Pele's tests whose names match
// files, each once, as Pele's loadCorpus and loadTestStrings collect them.
export function testStrings(pele, dir, files, keyword) {
  const out = new Set();
  for (const file of testFiles(pele, dir, files)) for (const s of fileLiterals(file, keyword)) out.add(s);
  return [...out];
}

// The documentation's diagrams of a kind, tests/corpus/<name>-docs.json.
export const docs = (pele, name) => JSON.parse(readFileSync(join(pele, 'tests/corpus', name + '-docs.json'), 'utf8'));

// An array literal in a file of Pele's, the one that follows start and ends where close does,
// at a line that starts with ]; unless it says otherwise, evaluated with the names given.
export function arrayIn(file, start, names = {}, close = '\n];') {
  const src = readFileSync(file, 'utf8');
  const from = src.indexOf(start);
  if (from < 0) throw new Error(`no ${start} in ${file}`);
  const open = from + start.length;
  const end = src.indexOf(close, open);
  return new Function(...Object.keys(names), 'return [' + src.slice(open, end) + '\n];')(...Object.values(names));
}

// Lone surrogates have no UTF-8, so Go strings cannot hold them; inputs with them are left out.
export function lone(s) {
  return /[\ud800-\udbff](?![\udc00-\udfff])|(?<![\ud800-\udbff])[\udc00-\udfff]/.test(s);
}

// A function that picks one of a list, with rnd.
export const picker = (rnd) => (list) => list[Math.floor(rnd() * list.length)];
