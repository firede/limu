import mermaidKeywordFile from '../../../limumygo/mermaid/keywords.json' with { type: 'json' };

// A longer line is left as it is, as highlighting it would cost more than it shows.
const maxHighlightLineLength = 2_000;

export type CodeLanguage = 'mermaid' | 'go' | 'ts' | 'shell' | 'css' | 'xml';

// Flat on purpose: CodeMirror's StreamLanguage copies editor state shallowly.
export interface HighlightState {
  started: boolean;
  frontmatter: boolean;
  quote: boolean;
  meta: boolean;
  comment: boolean;
  tag: boolean;
  expectsCommand: boolean;
  expectsRedirect: boolean;
  expectsPath: boolean;
  // The type of the Mermaid diagram, by limu's name of it, once the line that declares it is
  // read; '' before, or when it is no type limu reads.
  diagramType: string;
  declared: boolean;
}

export const createHighlightState = (): HighlightState => ({
  started: false,
  frontmatter: false,
  quote: false,
  meta: false,
  comment: false,
  tag: false,
  expectsCommand: true,
  expectsRedirect: false,
  expectsPath: false,
  diagramType: '',
  declared: false,
});

const escapeHtml = (value: string) => value
  .replaceAll('&', '&amp;')
  .replaceAll('<', '&lt;')
  .replaceAll('>', '&gt;')
  .replaceAll('"', '&quot;')
  .replaceAll("'", '&#39;');

const span = (className: string | undefined, value: string) => !value ? '' : className
  ? `<span class="${className}">${escapeHtml(value)}</span>`
  : escapeHtml(value);

const quoted = (value: string) => value.length > 1 && value.at(-1) === value[0]
  ? span('syn-punctuation', value[0]) + span('syn-string', value.slice(1, -1)) + span('syn-punctuation', value[0])
  : span('syn-punctuation', value[0]) + span('syn-string', value.slice(1));

function highlightStyleValue(value: string) {
  return [...value.matchAll(/#[0-9a-fA-F]{3,8}\b|-?\d*\.?\d+[a-z%]*|"[^"]*"|'[^']*'|--[\w-]+|[(),]|\s+|[^\s(),"'#]+|./g)].map((match) => {
    const part = match[0];
    if (/^#[0-9a-fA-F]{3,8}$/.test(part)) return span('syn-constant', part);
    if (/^-?\d*\.?\d+[a-z%]*$/.test(part)) return span('syn-number', part);
    if (/^["']/.test(part)) return quoted(part);
    if (part.startsWith('--')) return span('syn-variable', part);
    if (/^[(),]$/.test(part)) return span('syn-punctuation', part);
    if (value[match.index + part.length] === '(') return span('syn-function', part);
    return escapeHtml(part);
  }).join('');
}

const highlightDeclarations = (value: string) => value.split(/([,;])/).map((part) => {
  if (part === ',' || part === ';') return span('syn-punctuation', part);
  const declaration = part.match(/^(\s*)([\w-]+)(\s*:\s*)(.*)$/);
  if (!declaration) return escapeHtml(part);
  return escapeHtml(declaration[1]) + span('syn-variable', declaration[2]) + span('syn-punctuation', declaration[3]) + highlightStyleValue(declaration[4]);
}).join('');

function highlightYamlLine(line: string) {
  const entry = line.match(/^(\s*(?:-\s+)?)([\w-]+)(:)(\s*)(.*)$/);
  if (!entry) return span('syn-string', line);
  const value = entry[5];
  let highlighted = span('syn-string', value);
  if (/^(["']).*\1$/.test(value)) highlighted = quoted(value);
  else if (/^(?:true|false|null|~)$/.test(value)) highlighted = span('syn-constant', value);
  else if (/^[+-]?(?:\d+(?:\.\d+)?|\.\d+)$/.test(value)) highlighted = span('syn-number', value);
  return escapeHtml(entry[1]) + span('syn-yaml-key', entry[2]) + span('syn-punctuation', entry[3]) + escapeHtml(entry[4]) + highlighted;
}

const mermaidDeclaration = /^(?:flowchart-elk|flowchart|graph|xychart(?:-beta)?|gitGraph|pie|mindmap|sequenceDiagram|classDiagram|stateDiagram(?:-v2)?|erDiagram|gantt|journey|timeline|quadrantChart|requirement(?:Diagram)?|C4(?:Context|Container|Component|Dynamic|Deployment)|(?:architecture|sankey|block|packet|kanban|radar|treemap|info|agentflow|usecase|treeView|venn|cynefin|wardley|swimlane|ishikawa|eventmodeling)(?:-beta)?|railroad-(?:ebnf-|abnf-|peg-)?beta)\b/;
// The type a declaration says, by limu's name of it, as diagram.DetectType gives it.
const mermaidDeclaredType = (declaration: string) => {
  const name = declaration.replace(/-beta$/, '').replace(/-v2$/, '');
  if (/^(?:flowchart|graph)/.test(name)) return 'flowchart';
  if (name.startsWith('C4')) return 'c4';
  if (name.startsWith('railroad')) return 'railroad';
  const named: Record<string, string> = { sequenceDiagram: 'sequence', classDiagram: 'class', stateDiagram: 'state', erDiagram: 'er', requirementDiagram: 'requirement' };
  return named[name] ?? name;
};
// The keywords of each type, which limumygo's highlighter reads too, so that a diagram's
// text looks the same here as in the playground: the words that start a statement of the
// type, and those that are keywords anywhere in one. Text of no type known gets every type's
// words.
const wordSet = (list = '') => new Set(list.split(/\s+/).filter(Boolean));
const mermaidKeywords = new Map(Object.entries(mermaidKeywordFile.starts).map(([type, list]) => [type, {
  starts: wordSet(`${list} ${mermaidKeywordFile.common}`),
  inline: wordSet((mermaidKeywordFile.inline as Record<string, string>)[type]),
}]));
const mermaidAnyKeywords = { starts: wordSet(`${Object.values(mermaidKeywordFile.starts).join(' ')} ${mermaidKeywordFile.common}`), inline: new Set<string>() };
const mermaidDirections = wordSet(mermaidKeywordFile.directions);
const mermaidLink = /^(?:[xo<](?=--|==|-\.))?(?:--+|==+|-\.+-?|~~~+)(?:>|[xo](?!\w))?/;
// An entity relationship diagram's relationship: the cardinality of each end about a solid
// or dashed line, as ||--o{.
const mermaidRelationship = /^[|}][|o](?:--|\.\.)[o|][|{]/;
const mermaidLinkEnd = /(?:--+|==+|\.+-+)(?:>|[xo](?!\w))?/;
const mermaidShapes: [string, string[]][] = [
  ['(((', [')))']], ['([', ['])']], ['[(', [')]']], ['[[', [']]']], ['((', ['))']], ['{{', ['}}']],
  ['[/', ['/]', '\\]']], ['[\\', ['\\]', '/]']], ['[', [']']], ['(', [')']], ['{', ['}']],
];

function highlightMermaidLine(line: string, state: HighlightState) {
  if (!state.started && line.trim()) {
    state.started = true;
    if (/^\s*---\s*$/.test(line)) {
      state.frontmatter = true;
      return span('syn-punctuation', line);
    }
  }
  if (state.frontmatter) {
    if (/^\s*---\s*$/.test(line)) {
      state.frontmatter = false;
      return span('syn-punctuation', line);
    }
    return highlightYamlLine(line);
  }

  let output = '';
  let index = 0;
  let afterNode = false;
  // No word of the statement has been read yet.
  let start = true;
  let styleTarget = 0;
  // Past a colon that starts a label, as a message's in a sequence diagram: its words are
  // text, not names, and none of them is a keyword.
  let label = false;

  const string = (closers: string[] = []) => {
    const rest = line.slice(index);
    let end = -1;
    for (const closer of closers) {
      const position = rest.indexOf(closer);
      if (position >= 0 && (end < 0 || position < end)) end = position;
    }
    if (rest.startsWith('"')) {
      const close = rest.indexOf('"', 1);
      if (close < 0) {
        state.quote = true;
        output += quoted(rest);
        index = line.length;
        return;
      }
      output += quoted(rest.slice(0, close + 1));
      index += close + 1;
      return;
    }
    const text = end < 0 ? rest : rest.slice(0, end);
    output += span('syn-string', text);
    index += text.length;
  };

  if (state.quote) {
    const close = line.indexOf('"');
    if (close < 0) return span('syn-string', line);
    state.quote = false;
    output += span('syn-string', line.slice(0, close)) + span('syn-punctuation', '"');
    index = close + 1;
  }

  while (index < line.length) {
    const rest = line.slice(index);
    const wasAfterNode = afterNode;
    afterNode = false;
    let match: RegExpMatchArray | null;

    if (state.meta) {
      if ((match = rest.match(/^\s+|^,/))) output += match[0] === ',' ? span('syn-punctuation', ',') : match[0];
      else if (rest[0] === '}') { state.meta = false; match = ['}']; output += span('syn-punctuation', '}'); }
      else if (rest[0] === '"') { string(); continue; }
      else if ((match = rest.match(/^([\w-]+)(\s*:)/))) output += span('syn-variable', match[1]) + span('syn-punctuation', match[2]);
      else if ((match = rest.match(/^[^\s,}"]+/))) output += span(/^(?:true|false)$/.test(match[0]) ? 'syn-constant' : /^-?\d/.test(match[0]) ? 'syn-number' : 'syn-string', match[0]);
      index += match![0].length;
      continue;
    }

    if ((match = rest.match(/^\s+/))) {
      output += match[0];
      afterNode = false;
    } else if ((match = rest.match(/^%%\{.*?(?:\}%%|$)/))) {
      output += span('syn-language', match[0]);
    } else if (rest.startsWith('%%')) {
      output += span('syn-comment', rest);
      break;
    } else if (rest[0] === '"') {
      string();
      continue;
    } else if (rest.startsWith('@{')) {
      state.meta = true;
      match = ['@{'];
      output += span('syn-punctuation', '@{');
    } else if ((match = rest.match(/^(:::)([\w-]*)/))) {
      output += span('syn-punctuation', match[1]) + span('syn-language', match[2]);
    } else if ((match = rest.match(mermaidRelationship))) {
      output += span('syn-punctuation', match[0]);
    } else if (rest[0] === ':' && styleTarget < 2) {
      match = [':'];
      output += span('syn-punctuation', ':');
      label = true;
    } else if (rest[0] === ';') {
      // A semicolon ends a statement, and the next may start on the same line.
      match = [';'];
      output += span('syn-punctuation', ';');
      start = true;
      label = false;
      styleTarget = 0;
    } else if (rest[0] === '|') {
      const close = rest.indexOf('|', 1);
      output += span('syn-punctuation', '|');
      index += 1;
      if (close > 0) {
        string(['|']);
        output += span('syn-punctuation', '|');
        index += 1;
      }
      continue;
    } else if ((match = rest.match(mermaidLink))) {
      output += span('syn-punctuation', match[0]);
      start = false;
      // "A -- text --> B": the words between an open link and its arrow are the label.
      const end = /^(?:--|==|-\.)$/.test(match[0]) ? rest.slice(match[0].length).match(mermaidLinkEnd) : null;
      if (end?.index) {
        const text = rest.slice(match[0].length, match[0].length + end.index);
        if (/\S/.test(text) && !/[[\](){}|]/.test(text)) {
          output += span('syn-string', text) + span('syn-punctuation', end[0]);
          index += match[0].length + text.length + end[0].length;
          continue;
        }
      }
    } else if (styleTarget > 1 && /^[\w-]+\s*:/.test(rest)) {
      output += highlightDeclarations(rest);
      break;
    } else if (!state.declared && (match = rest.match(mermaidDeclaration))) {
      state.declared = true;
      state.diagramType = mermaidDeclaredType(match[0]);
      output += span('syn-keyword', match[0]);
      start = false;
    } else if ((match = rest.match(/^\d+(?:\.\d+)?\b/))) {
      output += span('syn-number', match[0]);
      start = false;
      if (styleTarget) styleTarget = 2;
    } else if ((match = rest.match(/^[\p{L}_][\p{L}\p{N}_]*(?:[-.](?![-.])[\p{L}\p{N}_]+)*/u))) {
      // A line that declares no type limu reads still declares one, of which no keyword is known.
      state.declared = true;
      const word = match[0];
      const keywords = mermaidKeywords.get(state.diagramType) ?? mermaidAnyKeywords;
      if (!label && ((start && keywords.starts.has(word)) || keywords.inline.has(word))) {
        output += span('syn-keyword', word);
        if (start && /^(?:style|classDef|linkStyle)$/.test(word)) styleTarget = 1;
      } else if (!label && mermaidDirections.has(word)) {
        output += span('syn-constant', word);
        if (styleTarget) styleTarget = 2;
      } else {
        const call = rest[word.length] === '(' && /\bcall\s+$/.test(line.slice(0, index));
        output += span(label ? 'syn-string' : call ? 'syn-function' : 'syn-variable', word);
        afterNode = !call && !label;
        if (styleTarget) styleTarget = 2;
      }
      start = false;
    } else if (wasAfterNode && rest[0] === '>') {
      output += span('syn-punctuation', '>');
      index += 1;
      string([']']);
      continue;
    } else {
      const shape = mermaidShapes.find(([open]) => rest.startsWith(open));
      if (shape) {
        start = false;
        output += span('syn-punctuation', shape[0]);
        index += shape[0].length;
        string(shape[1]);
        const closer = shape[1].find((close) => line.startsWith(close, index));
        if (closer) {
          output += span('syn-punctuation', closer);
          index += closer.length;
        }
        continue;
      }
      match = [rest[0]];
      output += span(/[\])}&;,:~/\\<>=.-]/.test(rest[0]) ? 'syn-punctuation' : undefined, rest[0]);
    }
    index += match[0].length;
  }
  return output;
}

const tsKeyword = /^(?:as|async|await|break|case|catch|class|const|continue|default|do|else|enum|export|extends|finally|for|function|if|implements|in|instanceof|interface|let|new|of|return|static|switch|throw|try|type|typeof|var|while|yield)$/;
// Types built into the language are language features, colored apart from named types.
const tsBuiltinType = /^(?:string|number|boolean|void|unknown|never|any|object|this|super)$/;
const tsType = /^[A-Z][\w$]*$/;

function highlightTypeScriptLine(line: string, state: HighlightState) {
  let output = '';
  let rest = line;
  if (state.comment) {
    const end = line.indexOf('*/');
    if (end < 0) return span('syn-comment', line);
    state.comment = false;
    output = span('syn-comment', line.slice(0, end + 2));
    rest = line.slice(end + 2);
  }
  if (state.quote) {
    // Inside a template literal that began on an earlier line.
    const end = rest.search(/(?<!\\)`/);
    if (end < 0) return output + span('syn-string', rest);
    state.quote = false;
    output += span('syn-string', rest.slice(0, end)) + span('syn-punctuation', '`');
    rest = rest.slice(end + 1);
  }
  const tokens = rest.match(/\/\/.*$|\/\*[\s\S]*?(?:\*\/|$)|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|`(?:\\.|[^`\\])*`?|\b\d[\d_]*(?:\.\d+)?\b|[A-Za-z_$][\w$]*|=>|===|!==|==|!=|<=|>=|&&|\|\||\?\?|\.\.\.|\s+|./g) ?? [];
  return output + tokens.map((token, index) => {
    if (token.startsWith('//')) return span('syn-comment', token);
    if (token.startsWith('/*')) {
      if (!token.endsWith('*/')) state.comment = true;
      return span('syn-comment', token);
    }
    if (/^['"`]/.test(token)) {
      if (token[0] === '`' && (token.length === 1 || token.at(-1) !== '`')) state.quote = true;
      return quoted(token);
    }
    if (/^\d/.test(token)) return span('syn-number', token);
    if (/^[A-Za-z_$]/.test(token)) {
      if (token === 'import' || token === 'from') return span('syn-import', token);
      if (/^(?:true|false|null|undefined)$/.test(token)) return span('syn-constant', token);
      if (tsKeyword.test(token)) return span('syn-keyword', token);
      const next = tokens.slice(index + 1).find((candidate) => !/^\s+$/.test(candidate));
      if (next === '(' && !/^[A-Z]/.test(token)) return span('syn-function', token);
      if (tsBuiltinType.test(token)) return span('syn-language', token);
      return span(tsType.test(token) ? 'syn-type' : 'syn-variable', token);
    }
    return span(/^\s+$/.test(token) ? undefined : 'syn-punctuation', token);
  }).join('');
}

const goKeyword = /^(?:break|case|chan|const|continue|default|defer|else|fallthrough|for|func|go|goto|if|interface|map|range|return|select|struct|switch|type|var)$/;
const goBuiltinType = /^(?:any|bool|byte|comparable|complex64|complex128|error|float32|float64|int|int8|int16|int32|int64|rune|string|uint|uint8|uint16|uint32|uint64|uintptr)$/;

// Go, as the documents show it: keywords, strings, numbers and comments in their colors,
// calls as functions, and names that are exported, as svg.Options, as types, except a
// field named in a composite literal. Other names stay in the text's color, so that a page
// of Go is not all colored.
function highlightGoLine(line: string, state: HighlightState) {
  let output = '';
  let rest = line;
  if (state.comment) {
    const end = line.indexOf('*/');
    if (end < 0) return span('syn-comment', line);
    state.comment = false;
    output = span('syn-comment', line.slice(0, end + 2));
    rest = line.slice(end + 2);
  }
  if (state.quote) {
    // Inside a raw string that began on an earlier line.
    const end = rest.indexOf('`');
    if (end < 0) return output + span('syn-string', rest);
    state.quote = false;
    output += span('syn-string', rest.slice(0, end)) + span('syn-punctuation', '`');
    rest = rest.slice(end + 1);
  }
  const tokens = rest.match(/\/\/.*$|\/\*[\s\S]*?(?:\*\/|$)|"(?:\\.|[^"\\])*"?|'(?:\\.|[^'\\])*'?|`[^`]*`?|\b(?:0[xX][\da-fA-F_]+|\d[\d_]*(?:\.\d*)?(?:[eE][+-]?\d+)?)\b|[A-Za-z_]\w*|:=|\.\.\.|==|!=|<=|>=|&&|\|\||<-|\s+|./g) ?? [];
  return output + tokens.map((token, index) => {
    if (token.startsWith('//')) return span('syn-comment', token);
    if (token.startsWith('/*')) {
      if (!token.endsWith('*/')) state.comment = true;
      return span('syn-comment', token);
    }
    if (/^["'`]/.test(token)) {
      if (token[0] === '`' && (token.length === 1 || token.at(-1) !== '`')) state.quote = true;
      return quoted(token);
    }
    if (/^\d/.test(token)) return span('syn-number', token);
    if (/^[A-Za-z_]/.test(token)) {
      if (token === 'import' || token === 'package') return span('syn-import', token);
      if (/^(?:true|false|nil|iota)$/.test(token)) return span('syn-constant', token);
      if (goKeyword.test(token)) return span('syn-keyword', token);
      const next = tokens.slice(index + 1).find((candidate) => !/^\s+$/.test(candidate));
      if (next === '(') return span('syn-function', token);
      if (goBuiltinType.test(token)) return span('syn-language', token);
      if (/^[A-Z]/.test(token) && next !== ':') return span('syn-type', token);
      return span(undefined, token);
    }
    return span(/^\s+$/.test(token) ? undefined : 'syn-punctuation', token);
  }).join('');
}

function highlightShellLine(line: string, state: HighlightState) {
  // Keep shell words intact: paths, flags, and URLs are not identifiers.
  const tokens = line.match(/\s+|#.*$|&&|\|\||[|;&<>]+|\\$|(?:\\.|"(?:\\.|[^"\\])*"|'[^']*'|[^\s|;&<>"'\\])+|./g) ?? [];
  let { expectsCommand, expectsRedirect, expectsPath } = state;
  let continued = false;

  const highlighted = tokens.map((token) => {
    if (/^\s+$/.test(token)) return escapeHtml(token);
    if (token.startsWith('#')) return span('syn-comment', token);
    continued = false;
    if (/^[|;&]+$/.test(token)) {
      expectsCommand = true;
      expectsPath = false;
      expectsRedirect = false;
      continued = /^(?:\||\|\||&&)$/.test(token);
      return span('syn-punctuation', token);
    }
    if (/^[<>]+$/.test(token)) {
      expectsRedirect = true;
      return span('syn-punctuation', token);
    }
    if (token === '\\') {
      continued = true;
      return span('syn-punctuation', token);
    }

    const assignment = token.match(/^(?:[A-Za-z_][\w]*|--[\w-]+)=/)?.[0];
    const command = expectsCommand && !expectsRedirect && !expectsPath && !assignment && !token.startsWith('-');
    const path = !command && !assignment && !token.startsWith('-')
      && (expectsPath || expectsRedirect || token.includes('/') || /^\.?[\w-]+(?:\.[\w-]+)+$/.test(token));
    expectsPath = /^(?:--output|-o)$/.test(token);
    if (!assignment && !expectsRedirect && !token.startsWith('-')) expectsCommand = command && token === 'npx';
    expectsRedirect = false;

    const value = assignment ? token.slice(assignment.length) : token;
    return escapeHtml(assignment ?? '') + value.split(/("(?:\\.|[^"\\])*"|'[^']*'|\$\{[^}]+\}|\$[A-Za-z_][\w]*)/g).filter(Boolean).map((part) => {
      if (/^(['"])[\s\S]*\1$/.test(part)) return quoted(part);
      if (part.startsWith('$')) return span('syn-variable', part);
      return span(command ? 'syn-command' : path || assignment ? 'syn-string' : undefined, part);
    }).join('');
  }).join('');
  Object.assign(state, continued
    ? { expectsCommand, expectsRedirect, expectsPath }
    : { expectsCommand: true, expectsRedirect: false, expectsPath: false });
  return highlighted;
}

// A selector, with an attribute test inside it set apart: the attribute's name, and its value as a string.
function highlightSelector(selector: string) {
  return selector.split(/(\[[^\]]*\]?)/).map((piece) => {
    const test = piece.match(/^\[\s*([\w-]+)\s*(?:([~|^$*]?=)\s*("[^"]*"?|'[^']*'?|[^\]\s]*))?([^\]]*)(\]?)$/);
    if (!test) return span(piece.trim() ? 'syn-selector' : undefined, piece);
    const value = test[3] ?? '';
    return span('syn-punctuation', '[') + span('syn-variable', test[1]) + span('syn-punctuation', test[2] ?? '')
      + (/^["']/.test(value) ? quoted(value) : span('syn-string', value))
      + escapeHtml(test[4]) + span('syn-punctuation', test[5]);
  }).join('');
}

function highlightCssLine(line: string, state: HighlightState) {
  let output = '';
  let rest = line;
  if (state.comment) {
    const end = line.indexOf('*/');
    if (end < 0) return span('syn-comment', line);
    state.comment = false;
    output = span('syn-comment', line.slice(0, end + 2));
    rest = line.slice(end + 2);
  }
  return output + rest.split(/(\/\*[\s\S]*?(?:\*\/|$))/).map((part) => {
    if (part.startsWith('/*')) {
      if (!part.endsWith('*/')) state.comment = true;
      return span('syn-comment', part);
    }
    const declaration = part.match(/^(\s*)([\w-]+)(\s*:\s*)([^{}]*?)(;?\s*)$/);
    if (declaration) {
      return escapeHtml(declaration[1]) + span('syn-variable', declaration[2]) + span('syn-punctuation', declaration[3])
        + highlightStyleValue(declaration[4]) + span('syn-punctuation', declaration[5]);
    }
    // What comes before a block: an at-rule, whose name is a keyword, or a selector.
    const rule = part.match(/^(\s*)(@[\w-]+)(.*)$/);
    if (rule) {
      return escapeHtml(rule[1]) + span('syn-keyword', rule[2])
        + rule[3].split(/([{}():])/).map((piece) => /^[{}():]$/.test(piece) ? span('syn-punctuation', piece) : escapeHtml(piece)).join('');
    }
    return part.split(/([{},])/).map((piece) => /^[{},]$/.test(piece) ? span('syn-punctuation', piece) : highlightSelector(piece)).join('');
  }).join('');
}

function highlightXmlLine(line: string, state: HighlightState) {
  let output = '';
  let index = 0;
  while (index < line.length) {
    const rest = line.slice(index);
    let length: number;
    if (state.comment) {
      const end = rest.indexOf('-->');
      length = end < 0 ? rest.length : end + 3;
      state.comment = end < 0;
      output += span('syn-comment', rest.slice(0, length));
    } else if (state.tag) {
      const part = rest.match(/^\s+|^\/?>|^=|^"[^"]*"?|^'[^']*'?|^[^\s=>/"']+|^./)![0];
      length = part.length;
      if (part === '>' || part === '/>') {
        state.tag = false;
        output += span('syn-punctuation', part);
      } else if (/^["']/.test(part)) output += quoted(part);
      else output += span(part === '=' ? 'syn-punctuation' : /^\s+$/.test(part) ? undefined : 'syn-variable', part);
    } else if (rest.startsWith('<!--')) {
      state.comment = true;
      length = 0;
    } else {
      const open = rest.match(/^(<\/?)([\w:.-]+)/);
      if (open) {
        state.tag = true;
        length = open[0].length;
        output += span('syn-punctuation', open[1]) + span('syn-tag', open[2]);
      } else {
        const next = rest.indexOf('<', 1);
        length = next < 0 ? rest.length : next;
        output += escapeHtml(rest.slice(0, length));
      }
    }
    index += length;
  }
  return output;
}

export function highlightLine(line: string, language: CodeLanguage, state: HighlightState = createHighlightState()) {
  if (line.length > maxHighlightLineLength) return escapeHtml(line);
  if (language === 'go') return highlightGoLine(line, state);
  if (language === 'ts') return highlightTypeScriptLine(line, state);
  if (language === 'shell') return highlightShellLine(line, state);
  if (language === 'css') return highlightCssLine(line, state);
  if (language === 'xml') return highlightXmlLine(line, state);
  return highlightMermaidLine(line, state);
}

export function highlightLines(lines: string[], language: CodeLanguage) {
  const state = createHighlightState();
  return lines.map((line) => highlightLine(line, language, state));
}

export function highlightCode(code: string, language: CodeLanguage, showLineNumbers = false) {
  const normalized = code.replace(/^\n|\n$/g, '');
  return highlightLines(normalized.split('\n'), language).map((line, index) => `<span class="doc-code-line"><span class="doc-line-number"${showLineNumbers ? '' : ' hidden'}>${index + 1}</span><span class="doc-code-source">${line}</span></span>`).join('');
}

const RE_INLINE_TYPE = [
  // A slice, a pointer or a map of a type, or a type of a package, as []scene.Item,
  // *style.Theme, map[string]any or svg.Options.
  /^(?:\[\]|\*|map\[\w+\])[\w.]+$/,
  /^[a-z]\w*\.[A-Z]\w*$/,
  // A type built into Go, or a type name in several capitalized words.
  /^(?:string|bool|int|float64|float32|error|any|rune|byte)$/,
  /^(?:[A-Z][a-z]+){2,}$/,
];
const RE_INLINE_CSS_CALL = /^(var)\((--[\w-]+)\)$/;

// Inline code in prose gets color in two cases only: a Go type and a CSS variable.
// Names, calls, values, tags, and Mermaid syntax are left as they are.
export function highlightInline(value: string): string | undefined {
  if (value.length > 120) return undefined;
  if (RE_INLINE_TYPE.some((pattern) => pattern.test(value))) return highlightLine(value, 'go');
  if (/^--[\w-]+$/.test(value)) return span('syn-variable', value);
  const match = value.match(RE_INLINE_CSS_CALL);
  if (match) return span(undefined, match[1]) + span('syn-punctuation', '(') + span('syn-variable', match[2]) + span('syn-punctuation', ')');
  return undefined;
}
