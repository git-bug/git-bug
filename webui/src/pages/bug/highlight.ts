import hljs from 'highlight.js/lib/common';

// highlight.js/lib/common ships the curated set of ~30 popular languages
// — enough for most code we'll see in a PR without pulling the full
// ~500 KB language bundle. Extensions not in the map return null and
// the caller renders the line as plain text.

// extToLang maps common file extensions to highlight.js language aliases.
// Only extensions whose content highlight.js actually supports end up
// here — guessing a language hljs doesn't ship yields an exception.
const extToLang: Record<string, string> = {
  ts: 'typescript',
  tsx: 'typescript',
  js: 'javascript',
  jsx: 'javascript',
  mjs: 'javascript',
  cjs: 'javascript',
  go: 'go',
  rs: 'rust',
  py: 'python',
  rb: 'ruby',
  java: 'java',
  kt: 'kotlin',
  kts: 'kotlin',
  scala: 'scala',
  swift: 'swift',
  c: 'c',
  h: 'c',
  cc: 'cpp',
  cpp: 'cpp',
  cxx: 'cpp',
  hpp: 'cpp',
  hh: 'cpp',
  cs: 'csharp',
  fs: 'fsharp',
  php: 'php',
  sh: 'bash',
  bash: 'bash',
  zsh: 'bash',
  fish: 'bash',
  ps1: 'powershell',
  sql: 'sql',
  json: 'json',
  yaml: 'yaml',
  yml: 'yaml',
  toml: 'ini', // toml isn't in common, fall back to ini which covers most of it
  ini: 'ini',
  xml: 'xml',
  html: 'xml',
  htm: 'xml',
  svg: 'xml',
  css: 'css',
  scss: 'scss',
  less: 'less',
  md: 'markdown',
  markdown: 'markdown',
  dockerfile: 'dockerfile',
  diff: 'diff',
  patch: 'diff',
  r: 'r',
  lua: 'lua',
  pl: 'perl',
  graphql: 'graphql',
  gql: 'graphql',
  proto: 'protobuf',
  nix: 'nix',
  dart: 'dart',
};

// basenameToLang covers files whose whole name (not extension) picks the
// language — Makefile, Dockerfile, etc.
const basenameToLang: Record<string, string> = {
  Dockerfile: 'dockerfile',
  Makefile: 'makefile',
  Earthfile: 'dockerfile',
  Rakefile: 'ruby',
  Gemfile: 'ruby',
};

// languageForPath picks a highlight.js language alias from a file path,
// or returns null when we don't know one. We do extension first (cheap
// + common case), then fall back to basename lookup, then a last check
// against hljs's own registered aliases.
export function languageForPath(path: string): string | null {
  const slash = path.lastIndexOf('/');
  const base = slash >= 0 ? path.slice(slash + 1) : path;
  if (basenameToLang[base]) return basenameToLang[base];
  const dot = base.lastIndexOf('.');
  if (dot <= 0) return null;
  const ext = base.slice(dot + 1).toLowerCase();
  const guess = extToLang[ext];
  if (!guess) return null;
  // Confirm hljs actually has this language registered. Guards against
  // the extension map drifting from what's actually bundled.
  return hljs.getLanguage(guess) ? guess : null;
}

// highlightLine returns the HTML markup for a single line of code,
// coloured according to `language`. When we don't recognise the
// language, we return the escaped plain text so the caller can still
// render it safely via dangerouslySetInnerHTML. `ignoreIllegals: true`
// prevents hljs from bailing on per-line snippets that wouldn't parse
// as a whole file.
//
// Prefer `highlightBlockLines` over this when you have multiple lines
// that share lexer state (a file, a hunk): per-line highlighting
// mis-colours anything inside multi-line strings/comments since it
// resets state between lines.
export function highlightLine(line: string, language: string | null): string {
  const base = baseHighlightLine(line, language);
  return wrapTemplateVars(base);
}

function baseHighlightLine(line: string, language: string | null): string {
  if (!language) return escapeHtml(line);
  try {
    return hljs.highlight(line, { language, ignoreIllegals: true }).value;
  } catch {
    return escapeHtml(line);
  }
}

// highlightBlockLines runs hljs on the joined-with-newlines block so
// lexer state carries across line boundaries (multi-line strings,
// template literals, block comments). The returned HTML is then split
// at each newline, rebalancing any spans that cross the break so each
// per-line fragment is self-contained HTML.
//
// Why not just split by newline in the input? Because hljs emits
// <span class="hljs-…">…</span> wrappers that may open on one input
// line and close on another. Naively splitting the output by \n would
// produce unbalanced fragments that render incorrectly.
export function highlightBlockLines(
  lines: string[],
  language: string | null
): string[] {
  let html: string;
  if (!language) {
    html = lines.map(escapeHtml).join('\n');
  } else {
    try {
      html = hljs.highlight(lines.join('\n'), {
        language,
        ignoreIllegals: true,
      }).value;
    } catch {
      html = lines.map(escapeHtml).join('\n');
    }
  }
  // Run the template-variable post-processor on the whole block before
  // splitting, so a pattern that spans the boundary of two hljs spans
  // is still recognised as one unit.
  html = wrapTemplateVars(html);
  return splitHighlightedByNewline(html);
}

// splitHighlightedByNewline walks hljs's output character-by-character
// tracking a stack of open <span> tags. At each '\n' it closes every
// open span to terminate the current line, emits the result, and then
// reopens them at the start of the next line so nesting is preserved.
// This keeps each per-line fragment a standalone, balanced HTML string
// we can feed to dangerouslySetInnerHTML.
function splitHighlightedByNewline(html: string): string[] {
  const lines: string[] = [];
  const openStack: string[] = [];
  let cur = '';
  let i = 0;
  while (i < html.length) {
    const ch = html[i];
    if (ch === '<') {
      const end = html.indexOf('>', i);
      if (end < 0) {
        // Truncated tag — bail; emit what we have.
        cur += html.slice(i);
        break;
      }
      const tag = html.slice(i, end + 1);
      if (tag.startsWith('</')) {
        // hljs only emits </span>; popping unconditionally is safe.
        openStack.pop();
        cur += tag;
      } else {
        openStack.push(tag);
        cur += tag;
      }
      i = end + 1;
    } else if (ch === '\n') {
      const closers = '</span>'.repeat(openStack.length);
      lines.push(cur + closers);
      cur = openStack.join('');
      i++;
    } else {
      cur += ch;
      i++;
    }
  }
  lines.push(cur);
  return lines;
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

// wrapTemplateVars adds colour to variable-ish patterns that individual
// language lexers don't always catch:
//   - $NAME, ${NAME}, ${NAME:-default}  — POSIX-shell / Dockerfile vars
//   - ${{ inputs.foo }}                  — GitHub Actions expressions
//
// Many hljs languages already handle the first two (bash, sh, …), but
// Dockerfiles, Makefiles, TOML, YAML, raw text etc. don't. Rather than
// condition per-language, we run the post-processor unconditionally:
// if hljs already classified a `$VAR` inside a span, our walker copies
// that span verbatim (the match sits inside the span tag so we never
// see it as plain text), so no double-wrapping occurs.
//
// GitHub-Actions `${{ }}` gets its own class (hljs-subst is the usual
// "template substitution" class hljs themes colour distinctly).
const varPatterns = [
  { re: /\$\{\{[^}]*\}\}/g, cls: 'hljs-subst' },
  { re: /\$\{[^}\n]+\}/g, cls: 'hljs-variable' },
  { re: /\$[A-Za-z_][A-Za-z0-9_]*/g, cls: 'hljs-variable' },
];

function wrapTemplateVars(html: string): string {
  let out = '';
  let i = 0;
  while (i < html.length) {
    if (html[i] === '<') {
      // Tag: copy verbatim so we never inject markup inside an
      // existing span's attribute list or double-wrap tokens hljs
      // already classified.
      const end = html.indexOf('>', i);
      if (end < 0) {
        out += html.slice(i);
        break;
      }
      out += html.slice(i, end + 1);
      i = end + 1;
      continue;
    }
    // Text chunk up to the next tag.
    let j = i;
    while (j < html.length && html[j] !== '<') j++;
    let chunk = html.slice(i, j);
    for (const { re, cls } of varPatterns) {
      chunk = chunk.replace(re, (m) => `<span class="${cls}">${m}</span>`);
    }
    out += chunk;
    i = j;
  }
  return out;
}
