import makeStyles from '@mui/styles/makeStyles';
import { useContext, useEffect, useState } from 'react';

import { ThemeContext } from 'src/components/Themer';

import { highlightBlockLines, languageForPath } from './highlight';

const useStyles = makeStyles((theme) => ({
  empty: {
    color: theme.palette.text.secondary,
    fontStyle: 'italic',
    padding: theme.spacing(2),
  },
  error: {
    color: theme.palette.error.main,
    padding: theme.spacing(2),
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    fontSize: '0.85rem',
  },
  summary: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    padding: theme.spacing(1),
    background: theme.palette.background.default,
    borderBottom: `1px solid ${theme.palette.divider}`,
    fontSize: '0.85rem',
    color: theme.palette.text.secondary,
  },
  add: { color: '#2da44e', fontWeight: 600 },
  del: { color: '#cf222e', fontWeight: 600 },
  file: {
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: theme.shape.borderRadius,
    marginBottom: theme.spacing(2),
    background: theme.palette.background.paper,
    // Long lines matter more than fitting the panel width — let each
    // file box scroll horizontally instead of wrapping or truncating.
    overflowX: 'auto',
  },
  fileHeader: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    padding: theme.spacing(1),
    borderBottom: `1px solid ${theme.palette.divider}`,
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    fontSize: '0.85rem',
  },
  path: {
    flex: 1,
    wordBreak: 'break-all',
  },
  // Dir segment muted so the filename reads as the primary thing; the
  // reader still sees where in the tree the file lives.
  pathDir: {
    color: theme.palette.text.secondary,
  },
  pathFile: {
    fontWeight: 600,
  },
  badge: {
    padding: '2px 6px',
    borderRadius: 4,
    fontSize: '0.7rem',
    textTransform: 'uppercase',
    color: 'white',
  },
  added: { background: '#2da44e' },
  removed: { background: '#cf222e' },
  renamed: { background: '#0969da' },
  binary: {
    padding: theme.spacing(1),
    color: theme.palette.text.secondary,
    fontStyle: 'italic',
  },
  hunk: {
    background: theme.palette.background.default,
  },
  hunkHeader: {
    background: theme.palette.action.hover,
    color: theme.palette.text.secondary,
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    fontSize: '0.75rem',
    padding: '2px 8px',
    borderTop: `1px solid ${theme.palette.divider}`,
  },
  line: {
    display: 'grid',
    // Line-number columns are fixed width; the code column takes as much
    // room as it needs. `minWidth: max-content` pushes the grid to fit
    // the longest token so overflowX on the parent can scroll without
    // clipping.
    gridTemplateColumns: '4ch 4ch max-content',
    minWidth: 'max-content',
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    fontSize: '0.78rem',
    lineHeight: 1.45,
    whiteSpace: 'pre',
  },
  lineAdd: {
    background: 'rgba(45, 164, 78, 0.12)',
    color: '#1a7f37',
  },
  lineDel: {
    background: 'rgba(207, 34, 46, 0.12)',
    color: '#82071e',
  },
  lineNumber: {
    color: theme.palette.text.disabled,
    padding: '0 4px',
    textAlign: 'right',
    userSelect: 'none',
  },
  lineContent: {
    padding: '0 8px',
  },
}));

type DiffLine = {
  type: 'add' | 'del' | 'ctx';
  old?: number;
  new?: number;
  content: string;
};

type DiffHunk = {
  header: string;
  oldStart: number;
  oldLines: number;
  newStart: number;
  newLines: number;
  lines: DiffLine[];
};

type FileDiffEntry = {
  path: string;
  oldPath?: string;
  isNew?: boolean;
  isDelete?: boolean;
  isBinary?: boolean;
  additions: number;
  deletions: number;
  hunks?: DiffHunk[];
};

type Props = {
  repoName: string;
  baseRef: string;
  headRef: string;
  originUrl?: string | null;
};

function prNumberFromOrigin(url?: string | null): number | null {
  if (!url) return null;
  const m = url.match(/\/pull\/(\d+)(?:\D|$)/);
  return m ? Number(m[1]) : null;
}

// splitPath splits a posix-style path into [dir, filename]. Dir keeps
// its trailing slash so callers can concatenate without worrying
// about separators. For paths without a directory, dir is "".
function splitPath(p: string): [string, string] {
  const slash = p.lastIndexOf('/');
  if (slash < 0) return ['', p];
  return [p.slice(0, slash + 1), p.slice(slash + 1)];
}

// PathCrumb renders a file path with the dir segment muted and the
// filename bolded — lets the reader find the file at a glance while
// keeping enough of the tree context to disambiguate e.g. two README.md.
function PathCrumb({
  path,
  classes,
}: {
  path: string;
  classes: ReturnType<typeof useStyles>;
}) {
  const [dir, file] = splitPath(path);
  return (
    <>
      {dir && <span className={classes.pathDir}>{dir}</span>}
      <span className={classes.pathFile}>{file}</span>
    </>
  );
}

/** PrDiff shows the tree-level diff from baseRef to headRef. Computed
 *  locally from gogit — no GitHub API cost. Big diffs can be heavy so
 *  we render without lazy-loading for now; if this becomes painful we
 *  can add per-file collapse + virtualisation later. */
export default function PrDiff({ repoName, baseRef, headRef, originUrl }: Props) {
  const classes = useStyles();
  const theme = useContext(ThemeContext);
  const [files, setFiles] = useState<FileDiffEntry[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    const prNum = prNumberFromOrigin(originUrl);
    const url =
      `/pr/${encodeURIComponent(repoName)}/diff` +
      `?base=${encodeURIComponent(baseRef)}&head=${encodeURIComponent(headRef)}` +
      (prNum ? `&pr=${prNum}` : '');
    fetch(url)
      .then(async (r) => {
        if (!r.ok) throw new Error(await r.text());
        return r.json();
      })
      .then((j: { files: FileDiffEntry[] }) => {
        if (!cancelled) setFiles(j.files ?? []);
      })
      .catch((e: Error) => {
        if (!cancelled) setError(e.message || 'request failed');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [repoName, baseRef, headRef, originUrl]);

  if (loading) return <div className={classes.empty}>Loading diff…</div>;
  if (error) return <div className={classes.error}>⚠ {error}</div>;
  if (!files || files.length === 0) {
    return (
      <div className={classes.empty}>
        No changes between base and head — the PR is either already merged
        or pointing at the same commit.
      </div>
    );
  }

  const totalAdd = files.reduce((s, f) => s + (f.additions || 0), 0);
  const totalDel = files.reduce((s, f) => s + (f.deletions || 0), 0);

  // Wrap the whole thing in the existing highlight-theme classes so hljs
  // tokens (e.g. .hljs-keyword, .hljs-string) pick up their colours from
  // the light/dark palette already wired up by Content/index.tsx.
  return (
    <div className={'highlight-theme'} data-theme={theme.mode}>
      <div className={classes.summary}>
        <span>{files.length} file{files.length === 1 ? '' : 's'} changed</span>
        <span className={classes.add}>+{totalAdd}</span>
        <span className={classes.del}>−{totalDel}</span>
      </div>
      {files.map((f) => {
        const language = languageForPath(f.path);
        return (
        <div key={f.path + (f.oldPath ?? '')} className={classes.file}>
          <div className={classes.fileHeader}>
            <span className={classes.path}>
              {f.oldPath && f.oldPath !== f.path ? (
                <>
                  <PathCrumb path={f.oldPath} classes={classes} /> →{' '}
                  <PathCrumb path={f.path} classes={classes} />
                </>
              ) : (
                <PathCrumb path={f.path} classes={classes} />
              )}
            </span>
            {f.isNew && (
              <span className={`${classes.badge} ${classes.added}`}>new</span>
            )}
            {f.isDelete && (
              <span className={`${classes.badge} ${classes.removed}`}>deleted</span>
            )}
            {f.oldPath && f.oldPath !== f.path && !f.isNew && !f.isDelete && (
              <span className={`${classes.badge} ${classes.renamed}`}>renamed</span>
            )}
            <span className={classes.add}>+{f.additions}</span>
            <span className={classes.del}>−{f.deletions}</span>
          </div>
          {f.isBinary ? (
            <div className={classes.binary}>Binary file — no textual diff.</div>
          ) : (
            (f.hunks ?? []).map((h, hi) => {
              // Highlight the hunk as one block so lexer state (multi-
              // line strings, block comments, template literals, …)
              // carries across line breaks. We still render the lines
              // individually so add/del backgrounds and line numbers
              // stay row-aligned.
              const highlighted = highlightBlockLines(
                h.lines.map((l) => l.content),
                language
              );
              return (
              <div key={hi} className={classes.hunk}>
                <div className={classes.hunkHeader}>{h.header}</div>
                {h.lines.map((l, li) => {
                  const cls =
                    l.type === 'add'
                      ? classes.lineAdd
                      : l.type === 'del'
                      ? classes.lineDel
                      : '';
                  const prefix = l.type === 'add' ? '+' : l.type === 'del' ? '−' : ' ';
                  const body = highlighted[li] ?? '';
                  return (
                    <div key={li} className={`${classes.line} ${cls}`}>
                      <span className={classes.lineNumber}>
                        {l.type !== 'add' && l.old ? l.old : ''}
                      </span>
                      <span className={classes.lineNumber}>
                        {l.type !== 'del' && l.new ? l.new : ''}
                      </span>
                      <span
                        className={classes.lineContent}
                        dangerouslySetInnerHTML={{ __html: prefix + body }}
                      />
                    </div>
                  );
                })}
              </div>
              );
            })
          )}
        </div>
        );
      })}
    </div>
  );
}
