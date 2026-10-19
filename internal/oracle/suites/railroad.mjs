// Railroad diagrams parsed as Pele parses them: the tokens Pele's common/tokens.ts makes of the
// text with the table of the notation its keyword names, the syntax tree that notation's parser
// makes of them, and the model of the text as it comes out of Pele's parseSource. The cases are
// Mermaid's documentation and specs, the inputs of Pele's own tests, cases written here in each
// of the four notations, random grammars and random pieces of them, and all of those mutated
// at random.
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { enc, peleError } from '../lib/value.mjs';
import { lone, testStrings, docs } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';

// The generators of Pele's tests/support/railroad-oracle.ts, as they are there.

function pick(rnd, list) {
  return list[Math.floor(rnd() * list.length)];
}

function several(rnd, max, make, joints) {
  const parts = [];
  for (let i = 1 + Math.floor(rnd() * max); i > 0; i--) parts.push(make());
  return parts.join(pick(rnd, joints));
}

const STRINGS = ['"a"', "'b'", '"x y"', '"\\n"', '"\\""', '"q\\\\"', '""', '"+"', '"é"', "'it\\'s'", '"a  b"', '"?"', '";"', '"(*"'];
// `title` at the start of a name is a title statement, which ends the rules. It is kept rare.
const NAMES = ['r', 'rule', 'my_rule', 'a-b', 'A1', 'terminals', 'x_', 'choice2', 'accTitles', 'digit', 'Expr', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'titles'];

function irExpression(rnd, depth) {
  const r = rnd();
  if (depth <= 0 || r < 0.4) return `${pick(rnd, ['terminal', 'nonterminal', 'special'])}(${pick(rnd, STRINGS)})`;
  if (r < 0.75) {
    return `${pick(rnd, ['sequence', 'choice'])}(${several(rnd, 3, () => irExpression(rnd, depth - 1), [', ', ',', ' , ', ',\n  '])})`;
  }
  return `${pick(rnd, ['optional', 'oneOrMore', 'zeroOrMore'])}( ${irExpression(rnd, depth - 1)} )`;
}

function ebnfPrimary(rnd, depth) {
  const r = rnd();
  if (depth <= 0 || r < 0.78) return rnd() < 0.1 ? pick(rnd, ['? any ?', '?x?', '? two words ?']) : pick(rnd, rnd() < 0.5 ? STRINGS : NAMES);
  const [open, close] = pick(rnd, [['( ', ' )'], ['[ ', ' ]'], ['{ ', ' }'], ['(', ')']]);
  return open + ebnfChoice(rnd, depth - 1) + close;
}

function ebnfTerm(rnd, depth) {
  let s = ebnfPrimary(rnd, depth);
  while (rnd() < 0.25) s += pick(rnd, ['?', '*', '+', ' - ' + ebnfPrimary(rnd, depth - 1), '-' + ebnfPrimary(rnd, depth - 1)]);
  return s;
}

function ebnfChoice(rnd, depth) {
  return several(rnd, 3, () => several(rnd, 3, () => ebnfTerm(rnd, depth), [' ', ' , ', ',', ' (* c *) ']), [' | ', '|']);
}

function abnfAlternation(rnd, depth) {
  const element = () => {
    const repeat = pick(rnd, ['', '', '', '*', '1*', '2*3', '3', '*4', '0*1', '1', '0']);
    if (depth <= 0 || rnd() < 0.78) return repeat + pick(rnd, ['"s"', '"a b"', '""', '%x41', '%d1.2', '%b0-1', 'name', 'DIGIT', 'rule-name', 'a1']);
    const [open, close] = pick(rnd, [['( ', ' )'], ['[ ', ' ]'], ['(', ')']]);
    return repeat + open + abnfAlternation(rnd, depth - 1) + close;
  };
  return several(rnd, 3, () => several(rnd, 3, element, [' ', '  ', '\n  ']), [' / ', '/']);
}

function pegChoice(rnd, depth) {
  const prefix = () => {
    const head = pick(rnd, ['', '', '', '&', '!']);
    const tail = pick(rnd, ['', '', '', '?', '*', '+']);
    if (depth <= 0 || rnd() < 0.78) return head + (rnd() < 0.1 ? '.' : pick(rnd, rnd() < 0.5 ? STRINGS : NAMES)) + tail;
    return `${head}( ${pegChoice(rnd, depth - 1)} )${tail}`;
  };
  return several(rnd, 3, () => several(rnd, 3, prefix, [' ', '  ', ' # c\n  ']), [' / ', '/']);
}

const COMMON = [
  'title', 'title ', 'title "T"', "title 'T'", 'title "a\\"b"', 'title "', 'accTitle: ', 'accTitle', 'accDescr: ', 'accDescr {', 'accDescr{x}', '}',
  '\n', '\r\n', ' ', '\t', '%%', '%% c', '%%{init: {}}%%', '---', '---\n', '---\nx: 1\n---\n', '\\', '\\"', "\\'", '\\n', '"', "'", '"a"', "'b'",
  '""', '"a b"', '"\\t\\\\"', ';', ' ;', ';\n', '=', '(', ')', ',', 'x', 'é', '#35;', '-', '_', '0', '9', 'a-b', 'A_1', '/*', '*/', '/* c */',
];

export const NOTATIONS = [
  {
    name: 'railroad',
    keyword: 'railroad-beta',
    fragments: [
      ...COMMON,
      'railroad-beta', 'terminal', 'nonterminal', 'special', 'sequence', 'choice', 'optional', 'oneOrMore', 'zeroOrMore',
      'terminal("a")', 'nonterminal("b")', 'special("c")', 'sequence(', 'choice(', 'optional(', 'oneOrMore(', 'zeroOrMore(',
      'terminalx', 'xterminal', 'choice1', 'r = terminal("a") ;', ', ', '))', '"),',
    ],
    seeds: [
      'railroad-beta',
      'railroad-beta\ntitle Example Grammar\naccTitle: Accessible Railroad\naccDescr: Railroad description\nrule = terminal("a") ;',
      'railroad-beta\nr1 = optional(terminal("a")) ;\nr2 = oneOrMore(terminal("b")) ;\nr3 = zeroOrMore(terminal("c")) ;',
      'railroad-beta\n/* comment before the first rule */\nrule = terminal("a") ;\nnext = nonterminal("rule") ;',
      'railroad-beta\naccDescr {\n  two\n  lines\n}\ntitle "Quoted \\"title\\"" %% c\nr = special(\'x\\ty\') ; %% d\n',
      'railroad-beta title x\ntitles = sequence(choice(terminal("a")), optional(oneOrMore(zeroOrMore(nonterminal("n"))))) ;',
      '---\ntitle: x\n---\nrailroad-beta\nr = choice(terminal("a"), sequence(terminal("b"), terminal("c")), special("d")) ;',
    ],
    rule: (rnd) => `${pick(rnd, NAMES)} = ${irExpression(rnd, 4)} ;`,
  },
  {
    name: 'EBNF',
    keyword: 'railroad-ebnf-beta',
    fragments: [
      ...COMMON,
      'railroad-ebnf-beta', '::=', '|', '[', ']', '{', '}', '?', '*', '+', ' - ', '? x ?', '?  ?', '??', '? a; ?', '?x?', '(*', '*)', '(* c *)', '(**)',
      '(*)', '( a )', '[ a ]', '{ a }', 'a?', 'a*', 'a+', 'a - b', 'a , b', 'a | b', 'r = a ;', 'r ::= "a" ;', '- (', '-- ', '---x',
    ],
    seeds: [
      'railroad-ebnf-beta',
      'railroad-ebnf-beta\nr = a , b , c ;\ns ::= "x" | [ y ] | { z } | ( a b ) ;',
      'railroad-ebnf-beta\nr = ? any character ? - "x" ;\ns = a - ( b | c )* - d? ;',
      'railroad-ebnf-beta\n(* iso *) /* w3c */ %% mermaid\nr = a? b* c+ ;',
      'railroad-ebnf-beta\naccDescr {\n  d\n}\nr = { a } ;\naccDescr { e',
      'railroad-ebnf-beta\nr = a? b? ;\ns = a ? b ;',
      'railroad-ebnf-beta\ntitle "T"\nr = "\\"" \'\\\'\' "\\n" ;',
    ],
    rule: (rnd) => `${pick(rnd, NAMES)} ${pick(rnd, ['=', '::='])} ${ebnfChoice(rnd, 3)} ;`,
  },
  {
    name: 'ABNF',
    keyword: 'railroad-abnf-beta',
    fragments: [
      ...COMMON,
      'railroad-abnf-beta', '/', '[', ']', '*', '1*', '*2', '2*4', '3', '03', '%x41', '%d65', '%b1000001', '%x30-39', '%x0D.0A', '%x', '%X4f.', '%x1-',
      '; comment', ';x', '; ', ' ;\n', '*a', '1*( a )', '[ a ]', '( a / b )', 'a / b', 'r = a ;\n', 'r = "a" ;', 'rule-name', '2DIGIT', '**', '1*2*3',
    ],
    seeds: [
      'railroad-abnf-beta',
      'railroad-abnf-beta\nr = 2*4a 3b *c 1*d *5e [ f ] ( g / h ) ;\ns = %x41 %d65.66 %b1-0 "str" ;',
      'railroad-abnf-beta\nr = a ; a comment\n  / b ;\n',
      'railroad-abnf-beta\ntitle "T"\naccTitle: at\nr = "a" ;\n; trailing comment',
      'railroad-abnf-beta\nr = *( a / "b" ) 1*2[ c ] ;\n',
    ],
    rule: (rnd) => `${pick(rnd, ['r', 'rule', 'a-b', 'A1', 'titles'])} = ${abnfAlternation(rnd, 3)} ;${pick(rnd, ['', '', '', ' c'])}`,
  },
  {
    name: 'PEG',
    keyword: 'railroad-peg-beta',
    fragments: [
      ...COMMON,
      'railroad-peg-beta', '<-', '<', '/', '&', '!', '?', '*', '+', '.', '# comment', '#x', '#', '&a', '!a', '!(', '&"x"', 'a?', 'a*', 'a+', '( a )',
      'a / b', 'R <- a ;', 'R <- "a" ;', '!.', '&&', '??', '..',
    ],
    seeds: [
      'railroad-peg-beta',
      'railroad-peg-beta\nR <- &a !b c? d* e+ . ;\nS <- ( a / b ) / !( c d )* ;',
      'railroad-peg-beta\n# a comment\nR <- "a" # another\n  / \'b\' ;',
      'railroad-peg-beta\ntitle "T"\nEnd <- !. ;\nAny <- &"x" . ;',
    ],
    rule: (rnd) => `${pick(rnd, NAMES)} <- ${pegChoice(rnd, 3)} ;`,
  },
];

const HEADERS = ['title T\n', 'title "Quoted \\" t"\n', "title 'q'\n", 'accTitle: a  b\n', 'accDescr: d\n', 'accDescr {\n multi\n line\n}\n', 'title\n'];

// Well-formed text of random shape in one notation.
export function grammar(notation, rnd) {
  let s = notation.keyword + pick(rnd, ['\n', ' ', '\n\n']);
  while (rnd() < 0.25) s += pick(rnd, HEADERS);
  for (let i = Math.floor(rnd() * 4); i > 0; i--) s += notation.rule(rnd) + pick(rnd, ['\n', '\n\n', '\n%% c\n']);
  return s;
}

// Token-level pieces in a random order, most of them well-formed on their own.
export function pieces(notation, rnd) {
  let s = rnd() < 0.9 ? notation.keyword + pick(rnd, ['\n', ' ', '\n\n', '']) : '';
  for (let i = 1 + Math.floor(rnd() * 14); i > 0; i--) s += pick(rnd, notation.fragments) + pick(rnd, ['', ' ', ' ', '\n']);
  return s;
}

// Fragments for mutating text in any notation, with what the parse suites of other
// diagrams mutate with.
export const FRAGMENTS = [
  ...new Set([
    ...NOTATIONS.flatMap((n) => n.fragments),
    'Railroad-EBNF-Beta', 'RAILROAD-BETA', 'railroad-PEG-beta', 'railroad-xyz-beta', ' ', ' ', '　', '😀', '中文', '&amp;', '#quot;',
    '<br>', '\r', '%%{init: {"railroad": {"arcRadius": 4}}}%%\n', '---\ntitle: Front\n---\n', '---\nconfig:\n  railroad:\n    padding: 4\n---\n',
  ]),
];

const RE_KEYWORD = /railroad-(?:ebnf-|abnf-|peg-)?beta/;
const RE_ANY_KEYWORD = /railroad-(?:ebnf-|abnf-|peg-)?beta/i;

// The expressions Pele's tests/railroad/model.test.ts reads through its ir, ebnf, abnf and peg
// helpers, written out as the helpers write them.
function modelTestCases(pele) {
  const src = readFileSync(join(pele, 'tests/railroad/model.test.ts'), 'utf8');
  const wrap = {
    ir: (e) => `railroad-beta\nr = ${e} ;`,
    ebnf: (e) => `railroad-ebnf-beta\nr = ${e} ;`,
    abnf: (e) => `railroad-abnf-beta\nr = ${e} ;`,
    peg: (e) => `railroad-peg-beta\nR <- ${e} ;`,
  };
  const out = [];
  for (const m of src.matchAll(/\b(ir|ebnf|abnf|peg)\(('(?:[^'\\\n]|\\.)*'|"(?:[^"\\\n]|\\.)*"|`(?:[^`\\]|\\.)*`)(?: \+ ([^)]*))?\)/g)) {
    try {
      const expression = new Function('return ' + m[2] + (m[3] ? ' + ' + m[3] : ''))();
      out.push(wrap[m[1]](expression));
    } catch {
      continue;
    }
  }
  return out;
}

// Cases written here: what each notation reads that the others do not, the edges of the
// tokens, the statements before the rules, and the errors of each parser.
export const WRITTEN = [
  // The constructor notation.
  'railroad-beta\nr = terminal("a") ;',
  'railroad-beta\nr = nonterminal(\'b\') ;',
  'railroad-beta\nr = special("any character") ;',
  'railroad-beta\nr = sequence(terminal("a"), nonterminal("b")) ;',
  'railroad-beta\nr = choice(terminal("a"), nonterminal("b"), special("c")) ;',
  'railroad-beta\nr = optional(terminal("a")) ;\ns = oneOrMore(terminal("a")) ;\nt = zeroOrMore(terminal("a")) ;',
  'railroad-beta\nr = sequence(choice(sequence(terminal("a")))) ;',
  'railroad-beta\nr = terminal("a\\nb\\tc\\rd") ;',
  'railroad-beta\nr = terminal("\\"\\\\\\x\\0") ;',
  'railroad-beta\nr = terminal("\\😀") ;',
  'railroad-beta\nr = terminal("a\\\nb") ;',
  'railroad-beta\nr = terminal("a b") ;',
  'railroad-beta\nr = terminal("a\\ b") ;',
  'railroad-beta\nr = terminal("") ;',
  'railroad-beta\nr = terminal(\'\') ;',
  'railroad-beta\nr = terminal("   ") ;',
  'railroad-beta\nr = terminal(" a ") ;',
  'railroad-beta\nr = special("") ;',
  'railroad-beta\nr = special("  x  ") ;',
  'railroad-beta\nr = nonterminal("#35;&amp;<br>") ;',
  'railroad-beta\nr = terminal("#35;") ;\ns = terminal("&lt;") ;',
  'railroad-beta\nterminals = nonterminal("choice") ;',
  'railroad-beta\nterminal = terminal("a") ;',
  'railroad-beta\nchoice2 = choice(terminal("a"),terminal("b")) ;',
  'railroad-beta-x\nr = terminal("a") ;',
  'railroad-betax\nr = terminal("a") ;',
  'railroad-beta%% c\nr = terminal("a") ;',
  'railroad-beta /* c */ r = terminal("a") ;',
  'railroad-beta\nr = sequence() ;',
  'railroad-beta\nr = sequence(terminal("a"),) ;',
  'railroad-beta\nr = sequence(terminal("a") terminal("b")) ;',
  'railroad-beta\nr = optional(terminal("a"), terminal("b")) ;',
  'railroad-beta\nr = optional() ;',
  'railroad-beta\nr = terminal(a) ;',
  'railroad-beta\nr = terminal("a" ;',
  'railroad-beta\nr = terminal "a" ;',
  'railroad-beta\nr = terminal("unclosed) ;',
  'railroad-beta\nr = terminal("a")',
  'railroad-beta\nr = terminal("a") terminal("b") ;',
  'railroad-beta\nr terminal("a") ;',
  'railroad-beta\n= terminal("a") ;',
  'railroad-beta\nr = choice(terminal("a"), choice(terminal("b"), terminal("c"))) ;',
  'railroad-beta\nr = oneOrMore(zeroOrMore(optional(special("x")))) ;',
  'railroad-beta\nr = /* a */ terminal(/* b */ "a" /* c */) /* d */ ;',
  'railroad-beta\nr = terminal("a") ; /* unclosed',
  'railroad-beta\nr = terminal("a") ; /* one */ /* two',
  'railroad-beta\n/**/r = terminal("a") ;',
  'railroad-beta\n/*/ r = terminal("a") ; */',
  'railroad-beta\nr = sequence(\n  terminal("a"),\n  terminal("b"),\n  terminal("c")\n) ;',
  'railroad-beta\nr-1 = terminal("a") ;\n_r = terminal("b") ;\nr_ = terminal("c") ;',
  'railroad-beta\n1r = terminal("a") ;',
  'railroad-beta\nr = terminal("a") ;\nr = terminal("b") ;',
  'railroad-beta\nr = sequence(' + 'optional('.repeat(30) + 'terminal("deep")' + ')'.repeat(30) + ', terminal("z")) ;',
  'railroad-beta\nr = choice(' + Array.from({ length: 40 }, (_, i) => `terminal("t${i}")`).join(', ') + ') ;',
  // EBNF.
  'railroad-ebnf-beta\nr = a b , c ;',
  'railroad-ebnf-beta\nr = a | "b" c ;',
  'railroad-ebnf-beta\nr = [ a ] ;\ns = { a | b } ;\nt = ( a ) ;',
  'railroad-ebnf-beta\nr = a? b* c+ ;',
  'railroad-ebnf-beta\nr = ?  any character  ? ;',
  'railroad-ebnf-beta\nr ::= a ;',
  'railroad-ebnf-beta\nr := a ;',
  'railroad-ebnf-beta\nr = a?*+ ;',
  'railroad-ebnf-beta\nr = a - b ;',
  'railroad-ebnf-beta\nr = a - b - "c" ;',
  'railroad-ebnf-beta\nr = a - ( b | c )* ;',
  'railroad-ebnf-beta\nr = a* - [ b ] ;',
  'railroad-ebnf-beta\nr = a - [ b - c ] - d ;',
  'railroad-ebnf-beta\nr = a - - b ;',
  'railroad-ebnf-beta\nr = - a ;',
  'railroad-ebnf-beta\nr = a -b ;',
  'railroad-ebnf-beta\nr = a-b ;',
  'railroad-ebnf-beta\nr = a? b? ;',
  'railroad-ebnf-beta\nr = a? ;\nq = b? ;',
  'railroad-ebnf-beta\nr = ? ? ;',
  'railroad-ebnf-beta\nr = ?? ;',
  'railroad-ebnf-beta\nr = ?a;b? ;',
  'railroad-ebnf-beta\nr = ?　? ;',
  'railroad-ebnf-beta\nr = ?　x ? ;',
  'railroad-ebnf-beta\nr = ? 😀 ? ;',
  'railroad-ebnf-beta\nr = ? multi\nline ? ;',
  'railroad-ebnf-beta\nr = a (* iso *) /* w3c */ b ;',
  'railroad-ebnf-beta\nr = a (*) b ;',
  'railroad-ebnf-beta\nr = a (**) b ;',
  'railroad-ebnf-beta\nr = a (* one *) (* two',
  'railroad-ebnf-beta\nr = a (* (* nested *) *) b ;',
  'railroad-ebnf-beta\nr = "it\\\'s" \'it\\\'s\' ;',
  'railroad-ebnf-beta\nr = "a\\qb" ;',
  'railroad-ebnf-beta\nr = ;',
  'railroad-ebnf-beta\nr = a',
  'railroad-ebnf-beta\nr = ( ) ;',
  'railroad-ebnf-beta\nr = a | ;',
  'railroad-ebnf-beta\nr = | a ;',
  'railroad-ebnf-beta\nr = a , ;',
  'railroad-ebnf-beta\nr = , a ;',
  'railroad-ebnf-beta\nr = [ a ) ;',
  'railroad-ebnf-beta\nr = ( a ] ;',
  'railroad-ebnf-beta\nr = { a ;',
  'railroad-ebnf-beta\nr = a - ;',
  'railroad-ebnf-beta\nr = a ) ;',
  'railroad-ebnf-beta\nr = a ] ;',
  'railroad-ebnf-beta\nr = a } ;',
  'railroad-ebnf-beta\nr = a ;;',
  'railroad-ebnf-beta\nr = a ; b',
  'railroad-ebnf-beta\nr = * ;',
  'railroad-ebnf-beta\nr = a * * ;',
  'railroad-ebnf-beta\nr = a\n---\nb ;',
  'railroad-ebnf-beta\nr = a\n---\nx: 1\n---\nb ;',
  'railroad-ebnf-beta\nr = a -- b ;',
  'railroad-ebnf-beta\nr = a --- b ;',
  'railroad-ebnf-beta\nr = title ;',
  'railroad-ebnf-beta\ntitles = a ;',
  'railroad-ebnf-beta\nr = a ;\ntitle Late',
  'railroad-ebnf-beta\naccTitles = a ;',
  'railroad-ebnf-beta\naccDescrs = a ;',
  'railroad-ebnf-beta\nr = "\\"" \'\\\'\' "\\n" ;',
  'railroad-ebnf-beta\nr = ( ( a ) ) ;',
  'railroad-ebnf-beta\nr = [ [ a ] ] ;\ns = { { a } } ;',
  'railroad-ebnf-beta\nr = ' + '( '.repeat(40) + 'a' + ' )'.repeat(40) + ' ;',
  'railroad-ebnf-beta\nr = ' + '[ '.repeat(25) + 'a' + ' ]'.repeat(25) + ' ;',
  'railroad-ebnf-beta\nr = a' + ' - b'.repeat(30) + ' ;',
  'railroad-ebnf-beta\nr = a' + '*+?'.repeat(20) + ' ;',
  'railroad-ebnf-beta\nr = ' + Array.from({ length: 30 }, (_, i) => `a${i}`).join(' | ') + ' ;',
  'railroad-ebnf-beta\nr = ' + 'a '.repeat(60) + ';',
  'railroad-ebnf-beta\nr = a ;\nr = b ;',
  'railroad-ebnf-beta\nr = a ;\n\n\n%% between\n\ns = b ;\n',
  'railroad-ebnf-beta\r\nr = a ;\r\ns = b ;\r\n',
  'railroad-ebnf-beta\nr\t=\ta\t;',
  'railroad-ebnf-beta\nr = a b ;',
  'railroad-ebnf-beta\nr = é ;',
  'railroad-ebnf-beta\nr = "é" "😀" "中文" ;',
  // ABNF.
  'railroad-abnf-beta\nr = *a ;\ns = 1*a ;\nt = 2*4a ;\nu = *4a ;\nv = 3a ;',
  'railroad-abnf-beta\nr = 1a ;\ns = 007a ;\nt = 0*1a ;\nu = *1"a" ;',
  'railroad-abnf-beta\nr = 2*( a / b ) ;\ns = 3[ a ] ;',
  'railroad-abnf-beta\nr = ' + '9'.repeat(400) + 'a ;',
  'railroad-abnf-beta\nr = ' + '9'.repeat(400) + '*' + '9'.repeat(400) + 'a ;',
  'railroad-abnf-beta\nr = *' + '9'.repeat(400) + 'a ;',
  'railroad-abnf-beta\nr = 0a ;\ns = 0*0a ;\nt = 5*2a ;\nu = 0*a ;\nv = 1*1a ;',
  'railroad-abnf-beta\nr = 00*01a ;\ns = 2*2a ;\nt = 1*0a ;',
  'railroad-abnf-beta\nr = %x41 %d65.66 %b1000001 %x30-39 ;',
  'railroad-abnf-beta\nr = %x0D.0A %X4F %D9 %B1 ;',
  'railroad-abnf-beta\nr = %x41. ;',
  'railroad-abnf-beta\nr = %x41- ;',
  'railroad-abnf-beta\nr = %x41-42-43.44 ;',
  'railroad-abnf-beta\nr = %x ;',
  'railroad-abnf-beta\nr = %q41 ;',
  'railroad-abnf-beta\nr = a / "b" c / [ d ] ;',
  'railroad-abnf-beta\nr = rule-name DIGIT ;',
  'railroad-abnf-beta\nr = a ; a comment, so the rule goes on\n  b ;\ns = c ;',
  'railroad-abnf-beta\nr = a ; \n',
  'railroad-abnf-beta\nr = a ;',
  'railroad-abnf-beta\nr = a ;\n',
  'railroad-abnf-beta\nr = a ;\r\n',
  'railroad-abnf-beta\nr = a ;\rs = b ;',
  'railroad-abnf-beta\nr = a ;; c\n ;',
  'railroad-abnf-beta\nr = "a\\" b ;',
  'railroad-abnf-beta\nr = "multi\nline" ;',
  'railroad-abnf-beta\nr = "unclosed ;',
  'railroad-abnf-beta\nr = \'a\' ;',
  'railroad-abnf-beta\nr = a /\n ;',
  'railroad-abnf-beta\nr = 2* ;',
  'railroad-abnf-beta\nr = 2 ;',
  'railroad-abnf-beta\nr = 2*3 ;',
  'railroad-abnf-beta\nr = ** a ;',
  'railroad-abnf-beta\nr = 1*2*3a ;',
  'railroad-abnf-beta\nr = 2DIGIT ;',
  'railroad-abnf-beta\nr = 2 DIGIT ;',
  'railroad-abnf-beta\nr = ( a ] ;',
  'railroad-abnf-beta\nr = [ a ) ;',
  'railroad-abnf-beta\nr = ( a ;',
  'railroad-abnf-beta\nr = a ) ;',
  'railroad-abnf-beta\nr = ( ) ;',
  'railroad-abnf-beta\nr = a // b ;',
  'railroad-abnf-beta\nr = / a ;',
  'railroad-abnf-beta\nr_x = a ;',
  'railroad-abnf-beta\n_r = a ;',
  'railroad-abnf-beta\nr = a_b ;',
  'railroad-abnf-beta\nr = a-1-b ;',
  'railroad-abnf-beta\nr =/ a ;',
  'railroad-abnf-beta\nr = <prose> ;',
  'railroad-abnf-beta\ntitle "a\\tb"\nr = a ;',
  'railroad-abnf-beta\ntitle "a\\"\nr = a ;',
  'railroad-abnf-beta\nr = ( ( a ) ) ;',
  'railroad-abnf-beta\nr = ' + '2*3( '.repeat(30) + 'a' + ' )'.repeat(30) + ' ;',
  'railroad-abnf-beta\nr = ' + 'a / '.repeat(30) + 'b ;',
  // PEG.
  'railroad-peg-beta\nR <- a? b* c+ ;',
  'railroad-peg-beta\nR <- a / "b" c ;',
  'railroad-peg-beta\nR <- . ;\nS <- &a ;\nT <- !"x" ;\nU <- !. ;',
  'railroad-peg-beta\nR <- !&a ;',
  'railroad-peg-beta\nR <- !( a b ) ;\nS <- !a* ;\nT <- &( a ) ;\nU <- ( !a )+ ;',
  'railroad-peg-beta\nR <- &special ;\nS <- !( . ) ;\nT <- !( ( "x" ) ) ;',
  'railroad-peg-beta\nR <- a** ;',
  'railroad-peg-beta\nR <- a?+ ;',
  'railroad-peg-beta\nR <- a # to the end of the line\n b ;',
  'railroad-peg-beta\nR <- a #\nb ;',
  'railroad-peg-beta\nR <- ;',
  'railroad-peg-beta\nR = a ;',
  'railroad-peg-beta\nR <- a ) ;',
  'railroad-peg-beta\nR <- ( a ;',
  'railroad-peg-beta\nR <- & ;',
  'railroad-peg-beta\nR <- ! ;',
  'railroad-peg-beta\nR <- a / ;',
  'railroad-peg-beta\nR <- / a ;',
  'railroad-peg-beta\nR <- a // b ;',
  'railroad-peg-beta\nR < - a ;',
  'railroad-peg-beta\nR <-a;',
  'railroad-peg-beta\nR <- [a-z] ;',
  'railroad-peg-beta\nR <- "\\b" \'\\\'\' ;',
  'railroad-peg-beta\nR <- .. ;',
  'railroad-peg-beta\nR <- ( ( a ) ) ;',
  'railroad-peg-beta\nR <- !( &( !( a ) ) ) ;',
  'railroad-peg-beta\nR <- ' + '( '.repeat(30) + 'a' + ' )*'.repeat(30) + ' ;',
  'railroad-peg-beta\nR <- ' + '!a &"b" '.repeat(20) + ';',
  'railroad-peg-beta\nEnd <- !. ;\nAny <- &"x" . ;',
  // The statements before the rules, in every notation.
  'railroad-ebnf-beta\ntitle   "A \\"quoted\\"\\ttitle"   \naccTitle:   Acc   title\naccDescr: One   line\nr = a ;',
  "railroad-peg-beta\ntitle 'single'\naccDescr {\n   Two\n   lines\n}\nR <- a ;",
  'railroad-beta title Plain  title %% comment\nr = terminal("a") ;',
  'railroad-ebnf-beta\ntitle first\ntitle second\nr = a ;',
  'railroad-ebnf-beta\ntitle\nr = a ;',
  'railroad-ebnf-beta\ntitle "\nr = a ;',
  "railroad-ebnf-beta\ntitle '\nr = a ;",
  'railroad-ebnf-beta\ntitle ""\nr = a ;',
  'railroad-ebnf-beta\ntitle "a\'\nr = a ;',
  'railroad-ebnf-beta\ntitle "unclosed\nr = a ;',
  'railroad-ebnf-beta\ntitle "a" b"\nr = a ;',
  'railroad-ebnf-beta\ntitle "\\\\"\nr = a ;',
  'railroad-beta\ntitle "a\\nb"\nr = terminal("a") ;',
  'railroad-peg-beta\ntitle "a\\qb"\nR <- a ;',
  'railroad-ebnf-beta\ntitle\tTabbed\nr = a ;',
  'railroad-ebnf-beta\ntitle%%c\nr = a ;',
  'railroad-ebnf-beta\ntitle x %% c\nr = a ;',
  'railroad-ebnf-beta\n  title Indented\nr = a ;',
  'railroad-ebnf-beta\ntitle #35; &amp; <b>b</b>\nr = a ;',
  'railroad-ebnf-beta\naccTitle: x\naccTitle: y\nr = a ;',
  'railroad-ebnf-beta\naccTitle : x\nr = a ;',
  'railroad-ebnf-beta\naccTitle:x %% c\nr = a ;',
  'railroad-ebnf-beta\naccTitle\nr = a ;',
  'railroad-ebnf-beta\naccTitle:\nr = a ;',
  'railroad-ebnf-beta\naccDescr :one\nr = a ;',
  'railroad-ebnf-beta\naccDescr{x}\nr = a ;',
  'railroad-ebnf-beta\naccDescr\n{\n  x\n}\nr = a ;',
  'railroad-ebnf-beta\naccDescr {\n  line one\n    line two\n\n\n  line three   \n}\nr = a ;',
  'railroad-ebnf-beta\naccDescr {\n\t\n  a  b \r\n\r\n c\n}\nr = a ;',
  'railroad-ebnf-beta\naccDescr {\nr = a ;',
  'railroad-ebnf-beta\naccDescr { a } accDescr { b',
  'railroad-ebnf-beta\naccDescr { a\naccDescr { b\nr = a ;',
  'railroad-ebnf-beta\naccDescr: d\naccDescr {\n e\n}\nr = a ;',
  'railroad-ebnf-beta\naccDescr {}\nr = a ;',
  'railroad-ebnf-beta\naccDescr:\nr = a ;',
  'railroad-ebnf-beta\naccDescr {\n  #35; &amp;\n}\nr = a ;',
  'railroad-ebnf-beta\naccTitle: Grammar\naccDescr {\n  Two\n  lines\n}\nr = a ;',
  'railroad-ebnf-beta\naccTitle:    leading   spaces\nr = a ;',
  'railroad-ebnf-beta\naccDescr {\n  first\n\n  second\n}\ntitle T\naccTitle: A\nr = a ;',
  'railroad-ebnf-beta\nr = a ;\naccTitle: late',
  '---\ntitle: Front\n---\nrailroad-ebnf-beta\nr = a ;',
  '---\ntitle: Front\n---\nrailroad-ebnf-beta\ntitle Own\nr = a ;',
  '---\ntitle: Front\n---\nrailroad-ebnf-beta\ntitle\nr = a ;',
  '---\ntitle: Front\n---\nrailroad-ebnf-beta\ntitle ""\nr = a ;',
  '---\ntitle: ""\n---\nrailroad-peg-beta\nR <- a ;',
  '---\nconfig:\n  railroad:\n    arcRadius: 20\n---\nrailroad-abnf-beta\nr = a ;',
  '%%{init: {"railroad": {"arcRadius": 4}}}%%\nrailroad-beta\nr = terminal("a") ;',
  '%%{init: {}}%%\nrailroad-peg-beta\nR <- a ;',
  'railroad-ebnf-beta\n%%{init: {}}%%\nr = a ;',
  'railroad-ebnf-beta\nr = a %%{x}%%\n ;',
  'railroad-ebnf-beta\nr = a ;\n%%{\ns = b ;',
  'railroad-ebnf-beta\n  %% indented comment\nr = a ;',
  'railroad-ebnf-beta\nr = a ; %%',
  // Keywords and what detects.
  'Railroad-EBNF-Beta\nr = "a" ;',
  'RAILROAD-BETA\nr = terminal("a") ;',
  'railroad-PEG-beta\nR <- a ;',
  'railroad-Abnf-beta\nr = a ;',
  'railroad-xyz-beta\nr = a ;',
  'railroad-ebnf-betax\nr = a ;',
  'railroad-ebnf-beta-x\nr = a ;',
  'railroad-ebnf\nr = a ;',
  '  \n\t railroad-ebnf-beta\nr = a ;',
  ' railroad-ebnf-beta\nr = a ;',
  'railroad-ebnf-beta',
  'railroad-ebnf-beta\n',
  'railroad-beta',
  'railroad-abnf-beta ',
  'railroad-peg-beta\n\n\n',
  'railroad-ebnf-beta r = a ;',
  'railroad-ebnf-beta\nrailroad-ebnf-beta\nr = a ;',
  'railroad-ebnf-beta\nr = a ;\nrailroad-ebnf-beta',
  'railroad-abnf-beta\nr = "a" ;',
  'railroad-peg-beta\nr <- "a" ;',
  'r = "a" ;',
  '',
  'not a railroad diagram',
];

// The corpus the suites start from: documentation, specs, the strings of Pele's tests, its
// seeds and the cases above, and each of those under the other notations' keywords, which
// exercise the errors and the shared constructs, as Pele's corpusFor borrows them.
export function corpus(pele) {
  const own = [
    ...new Set([
      ...docs(pele, 'railroad'),
      ...testStrings(pele, 'tests/compat/railroad', /\.(spec|test)\.[jt]s$/, RE_KEYWORD),
      ...testStrings(pele, 'tests/railroad', /^(model|oracle)\.test\.ts$/, RE_ANY_KEYWORD),
      ...testStrings(pele, 'tests/render', /^railroad.*\.test\.ts$/, RE_ANY_KEYWORD),
      ...modelTestCases(pele),
      ...NOTATIONS.flatMap((n) => n.seeds),
      ...WRITTEN,
    ]),
  ].filter((s) => !lone(s) && s.length <= 4000);
  const borrowed = [];
  for (const n of NOTATIONS) {
    for (const s of own) if (RE_KEYWORD.test(s) && !s.includes(n.keyword)) borrowed.push(s.replace(RE_KEYWORD, n.keyword));
  }
  return [...new Set([...own, ...borrowed])];
}

// The notation of a text by its keyword, as Pele's notationOf reads it, or railroad when it
// names none, so that every text goes through some table.
const NOTATION_OF = { 'ebnf-': 'ebnf', 'abnf-': 'abnf', 'peg-': 'peg' };
function notationOf(source) {
  const m = /^\s*railroad-(ebnf-|abnf-|peg-)?beta/i.exec(source);
  return m === null ? 'railroad' : (NOTATION_OF[m[1]?.toLowerCase()] ?? 'railroad');
}

// A syntax tree as JSON: its objects with the fields Pele gives them, those it leaves
// undefined left out.
function tree(v) {
  if (Array.isArray(v)) return v.map(tree);
  if (v !== null && typeof v === 'object') {
    const out = {};
    for (const [k, x] of Object.entries(v)) if (x !== undefined) out[k] = tree(x);
    return out;
  }
  return v;
}

// A node of the rule model, written short: its type, its text or bounds, and its parts.
export function dumpNode(node) {
  switch (node.type) {
    case 'terminal':
      return ['t', node.value];
    case 'nonterminal':
      return ['n', node.name];
    case 'special':
      return ['s', node.text];
    case 'sequence':
      return ['seq', ...node.elements.map(dumpNode)];
    case 'choice':
      return ['alt', ...node.alternatives.map(dumpNode)];
    case 'optional':
      return ['opt', dumpNode(node.element)];
    case 'repetition':
      return ['rep', enc(node.min), enc(node.max), dumpNode(node.element)];
  }
  throw new Error('no such node ' + node.type);
}

// How deep a text's brackets go at most, which a tree written as JSON goes some times deeper.
function depth(s) {
  let d = 0;
  let most = 0;
  for (const c of s) {
    if ('([{'.includes(c)) most = Math.max(most, ++d);
    else if (')]}'.includes(c)) d--;
  }
  return most;
}

export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { railroad } = await src('diagrams/railroad/index.ts');
  const { tokenize } = await src('diagrams/common/tokens.ts');
  const parsers = {
    railroad: { ...(await src('diagrams/railroad/ir.ts')), parse: 'parseIr', table: 'IR_TOKENS' },
    ebnf: { ...(await src('diagrams/railroad/ebnf.ts')), parse: 'parseEbnf', table: 'EBNF_TOKENS' },
    abnf: { ...(await src('diagrams/railroad/abnf.ts')), parse: 'parseAbnf', table: 'ABNF_TOKENS' },
    peg: { ...(await src('diagrams/railroad/peg.ts')), parse: 'parsePeg', table: 'PEG_TOKENS' },
  };

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const opt = (x) => (x === undefined ? undefined : x);
  // The tokens and the tree the text's notation makes of the text as it is.
  const direct = (source) => {
    const notation = notationOf(source);
    const p = parsers[notation];
    const types = p[p.table];
    const out = { notation };
    try {
      const t = tokenize(source, types, 'railroad');
      out.tokens = t.kinds.slice(0, -1).map((k, i) => [types[k].name, source.slice(t.starts[i], t.ends[i])]);
    } catch {
      out.tokens = null;
    }
    try {
      const ast = p[p.parse](source);
      // A tree nested past what a JSON reader takes is checked by its model alone.
      out.ast = depth(source) > 400 ? null : tree(ast);
    } catch (e) {
      Object.assign(out, caught(e));
    }
    return out;
  };
  // Pele's parseSource for a railroad diagram, and the raw text given to its notation alone.
  const run = (source) => {
    const out = { src: source, direct: direct(source) };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'railroad') return out;
    try {
      const m = railroad.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title);
      out.model = {
        notation: m.notation,
        title: opt(m.title),
        accTitle: opt(m.accTitle),
        accDescr: opt(m.accDescr),
        rules: m.rules.map((r) => [r.name, dumpNode(r.definition)]),
      };
    } catch (e) {
      Object.assign(out, caught(e));
    }
    return out;
  };

  const base = corpus(pele);
  const all = [...base];
  const seen = new Set(all);
  const add = (s) => {
    if (s.length <= 3000 && !lone(s) && !seen.has(s)) {
      seen.add(s);
      all.push(s);
    }
  };
  // A quarter each of mutated corpus texts, random token pieces, random grammars, and
  // mutated random grammars, in each notation, as Pele's oracle test makes them.
  for (const [k, notation] of NOTATIONS.entries()) {
    const rnd = random(70 + k);
    const own = base.filter((s) => s.includes(notation.keyword));
    const mutated = mutator(own.length ? own : base, notation.fragments, rnd);
    for (let i = 0; i < 1200; i++) {
      const kind = i % 4;
      let s = kind === 0 ? mutated() : kind === 1 ? pieces(notation, rnd) : grammar(notation, rnd);
      if (kind === 3) s = mutator([s], notation.fragments, rnd)();
      add(s);
    }
    // And more well-formed grammars, which the model is made of.
    for (let i = 0; i < 500; i++) add(grammar(notation, rnd));
  }
  // Mutated text of any notation, with fragments of all; one in six cut short, as a rule cut
  // off by the end of the text ends on a different error than one cut off by a line break.
  const rnd = random(77);
  const next = mutator([...seen].filter((s) => s.length <= 3000), FRAGMENTS, rnd);
  for (let i = 0; i < 1500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  // Pele's stress cases, smaller.
  for (const s of [
    'railroad-ebnf-beta\nr = ?' + ' a'.repeat(500),
    'railroad-ebnf-beta\nr = ' + '?  '.repeat(300),
    'railroad-ebnf-beta\nr = ' + '(* a '.repeat(200),
    'railroad-ebnf-beta\nr = ' + '(*)'.repeat(300),
    'railroad-beta\n' + '/* a '.repeat(200),
    'railroad-ebnf-beta\n' + '/**/(**)'.repeat(100),
    'railroad-ebnf-beta\n' + 'accDescr { '.repeat(100),
    'railroad-ebnf-beta\n' + ('accDescr' + ' '.repeat(92)).repeat(10),
    'railroad-beta\n' + ('accDescr: ' + '% '.repeat(45) + '\n').repeat(10),
    'railroad-peg-beta\n' + ('accTitle' + ' '.repeat(42)).repeat(20),
    'railroad-beta\ntitle ' + 'a % '.repeat(200),
    'railroad-abnf-beta\n' + 'title '.repeat(200),
    'railroad-beta\n' + 'titlea'.repeat(200),
    'railroad-beta\nr = terminal("' + 'a\\"'.repeat(300),
    'railroad-peg-beta\nR <- ' + "'" + '\\\\'.repeat(500),
    'railroad-abnf-beta\nr = "' + 'a '.repeat(500),
    'railroad-ebnf-beta\nr = a ' + '---\n'.repeat(300),
    'railroad-beta\n' + '%%{ a '.repeat(200),
    'railroad-ebnf-beta\nr = a ' + '-'.repeat(1000),
    'railroad-abnf-beta\nr = ' + '1'.repeat(1000) + 'a ;',
    'railroad-abnf-beta\nr = ' + '12*34 '.repeat(200),
    'railroad-abnf-beta\nr = %x' + '41.'.repeat(300) + ' ;',
    'railroad-abnf-beta\n' + '; a comment\n'.repeat(100),
    'railroad-peg-beta\n' + '# a comment\n'.repeat(100),
    'railroad-beta\n' + 'terminal'.repeat(200),
    'railroad-ebnf-beta\nr = ' + '('.repeat(300) + 'a' + ')'.repeat(300) + ' ;',
    'railroad-ebnf-beta\nr = ' + '['.repeat(600) + 'a' + ']'.repeat(600) + ' ;',
    'railroad-ebnf-beta\nr = ' + '([{'.repeat(300),
    'railroad-beta\nr = ' + 'choice('.repeat(300),
    'railroad-peg-beta\nR <- ' + '!&'.repeat(500) + 'a ;',
    'railroad-abnf-beta\nr = ' + '['.repeat(500) + 'a' + ']'.repeat(500) + ' ;',
  ]) {
    seen.delete(s);
    add(s);
  }
  write('diagram/railroad/testdata/pele/railroad.json', { cases: all.map(run) });
}
