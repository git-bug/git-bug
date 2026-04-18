import { gql, useQuery } from '@apollo/client';
import SearchIcon from '@mui/icons-material/Search';
import { ClickAwayListener, InputBase, MenuItem, MenuList, Paper, Popper } from '@mui/material';
import { alpha } from '@mui/material/styles';
import makeStyles from '@mui/styles/makeStyles';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useNavigate } from 'react-router';

const useStyles = makeStyles((theme) => ({
  wrap: {
    position: 'relative',
    borderRadius: theme.shape.borderRadius,
    backgroundColor: alpha(theme.palette.common.white, 0.15),
    '&:hover': {
      backgroundColor: alpha(theme.palette.common.white, 0.25),
    },
    marginLeft: theme.spacing(2),
    marginRight: theme.spacing(2),
    width: '100%',
    maxWidth: 420,
    display: 'flex',
    alignItems: 'center',
    padding: theme.spacing(0, 1),
  },
  iconBox: {
    color: alpha(theme.palette.primary.contrastText, 0.7),
    marginRight: theme.spacing(1),
    display: 'flex',
  },
  input: {
    color: 'inherit',
    width: '100%',
    fontSize: '0.9rem',
  },
  popper: {
    zIndex: theme.zIndex.modal + 1,
    minWidth: 280,
  },
  menuItem: {
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    fontSize: '0.85rem',
  },
}));

// Static key and value catalogs. Dynamic values (repo, org) come from the
// GraphQL repository list; author/label suggestions are not wired up yet
// because they need per-repo queries that are out of scope here.
const KEY_SUGGESTIONS = [
  'status:',
  'kind:',
  'repo:',
  'org:',
  'author:',
  'label:',
  'title:',
  'no:',
  'sort:',
];

const STATIC_VALUE_SUGGESTIONS: Record<string, string[]> = {
  status: ['open', 'closed', 'merged', 'draft'],
  kind: ['issue', 'pr'],
  sort: ['creation', 'creation-asc', 'edit', 'edit-asc', 'id'],
  no: ['label'],
};

const REPO_NAMES_QUERY = gql`
  query RepoNamesForSearch {
    repositories {
      nodes {
        name
      }
    }
  }
`;

interface RepoNamesData {
  repositories: { nodes: Array<{ name: string | null }> };
}

/** GlobalSearch is the always-visible search field in the top app bar. It
 *  auto-completes token-by-token: an empty or pre-colon token suggests
 *  filter keys, a `key:` token suggests known values for that key. Repo
 *  and org suggestions are sourced from the registered repository list.
 *
 *  The input tracks the current URL by default (so clicking filter chips
 *  on the list page updates the search text) but backs off while the user
 *  is actively typing. Submitting navigates to /search.
 *
 *  Mounted under /r/:name/ with no explicit q= it pre-fills the repo:
 *  token so users see — and can delete — the current scope. */
export default function GlobalSearch() {
  const classes = useStyles();
  const navigate = useNavigate();
  const location = useLocation();

  // The Header sits outside the /r/:repoName/* route; extract from pathname.
  const repoName = repoNameFromPath(location.pathname);
  const urlValue = derivedValue(location, repoName);

  const [value, setValue] = useState(urlValue);
  const [cursor, setCursor] = useState(0);
  const [highlight, setHighlight] = useState(0);
  const [open, setOpen] = useState(false);
  const isDirtyRef = useRef(false);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const anchorRef = useRef<HTMLFormElement | null>(null);

  useEffect(() => {
    if (!isDirtyRef.current) {
      setValue(urlValue);
    }
  }, [urlValue]);

  const { data: repoData } = useQuery<RepoNamesData>(REPO_NAMES_QUERY);
  const repoNames = useMemo(() => {
    const names = (repoData?.repositories.nodes ?? [])
      .map((n) => n.name)
      .filter((n): n is string => !!n && n !== '__default');
    return names;
  }, [repoData]);
  const orgs = useMemo(() => {
    const s = new Set<string>();
    for (const n of repoNames) {
      const i = n.indexOf('/');
      if (i > 0) s.add(n.slice(0, i));
    }
    return Array.from(s).sort();
  }, [repoNames]);

  const suggestions = useMemo(
    () => computeSuggestions(value, cursor, repoNames, orgs),
    [value, cursor, repoNames, orgs]
  );

  const applySuggestion = (s: string) => {
    const next = replaceCurrentToken(value, cursor, s);
    isDirtyRef.current = true;
    setValue(next.value);
    setOpen(false);
    // Move cursor to the end of the inserted token on next tick.
    window.queueMicrotask(() => {
      const el = inputRef.current;
      if (el) {
        el.focus();
        el.setSelectionRange(next.cursor, next.cursor);
        setCursor(next.cursor);
      }
    });
  };

  return (
    <form
      ref={anchorRef}
      className={classes.wrap}
      onSubmit={(e) => {
        e.preventDefault();
        // Enter always submits. Tab completes the highlighted suggestion.
        // Accepting a suggestion with Enter would force a double press to
        // actually run the search, which is surprising.
        const q = value.trim();
        isDirtyRef.current = false;
        setOpen(false);
        navigate(q === '' ? '/search' : `/search?q=${encodeURIComponent(q)}`);
      }}
    >
      <span className={classes.iconBox}>
        <SearchIcon fontSize="small" />
      </span>
      <InputBase
        inputRef={inputRef}
        placeholder="search — e.g. kind:pr status:open repo:…"
        className={classes.input}
        value={value}
        onChange={(e) => {
          isDirtyRef.current = true;
          setValue(e.target.value);
          setCursor(e.target.selectionStart ?? e.target.value.length);
          setOpen(true);
          setHighlight(0);
        }}
        onSelect={(e) => {
          const el = e.target as HTMLInputElement;
          setCursor(el.selectionStart ?? el.value.length);
        }}
        onFocus={() => setOpen(true)}
        onBlur={() => {
          isDirtyRef.current = false;
        }}
        onKeyDown={(e) => {
          if (!open || suggestions.length === 0) {
            if (e.key === 'Escape') setOpen(false);
            return;
          }
          if (e.key === 'ArrowDown') {
            e.preventDefault();
            setHighlight((i) => (i + 1) % suggestions.length);
          } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            setHighlight(
              (i) => (i - 1 + suggestions.length) % suggestions.length
            );
          } else if (e.key === 'Tab') {
            e.preventDefault();
            applySuggestion(suggestions[highlight]);
          } else if (e.key === 'Escape') {
            setOpen(false);
          }
        }}
        inputProps={{ 'aria-label': 'global search' }}
      />
      <Popper
        open={open && suggestions.length > 0}
        anchorEl={anchorRef.current}
        placement="bottom-start"
        className={classes.popper}
      >
        <ClickAwayListener onClickAway={() => setOpen(false)}>
          <Paper>
            <MenuList autoFocusItem={false}>
              {suggestions.slice(0, 8).map((s, i) => (
                <MenuItem
                  key={s}
                  className={classes.menuItem}
                  selected={i === highlight}
                  onMouseDown={(e) => {
                    // Prevent the input from losing focus before we apply.
                    e.preventDefault();
                  }}
                  onClick={() => applySuggestion(s)}
                >
                  {s}
                </MenuItem>
              ))}
            </MenuList>
          </Paper>
        </ClickAwayListener>
      </Popper>
    </form>
  );
}

/** Given the full input value and cursor position, returns the suggestion
 *  strings appropriate for the token at the cursor. Keys return `key:`,
 *  values return bare values. */
function computeSuggestions(
  value: string,
  cursor: number,
  repoNames: string[],
  orgs: string[]
): string[] {
  const tok = tokenAtCursor(value, cursor);
  const colon = tok.indexOf(':');
  if (colon < 0) {
    // Key suggestion: match any key that starts with the current token.
    const prefix = tok.toLowerCase();
    return KEY_SUGGESTIONS.filter((k) =>
      k.toLowerCase().startsWith(prefix)
    ).sort();
  }
  const key = tok.slice(0, colon).toLowerCase();
  const partial = tok.slice(colon + 1).toLowerCase();
  let pool: string[] | null = null;
  if (STATIC_VALUE_SUGGESTIONS[key]) {
    pool = STATIC_VALUE_SUGGESTIONS[key];
  } else if (key === 'repo') {
    // Accept both short basename and org/name forms.
    const seen = new Set<string>();
    const out: string[] = [];
    for (const n of repoNames) {
      const base = n.includes('/') ? n.slice(n.indexOf('/') + 1) : n;
      if (!seen.has(base)) {
        seen.add(base);
        out.push(base);
      }
      out.push(n);
    }
    pool = out;
  } else if (key === 'org') {
    pool = orgs;
  }
  if (!pool) return [];
  return pool.filter((v) => v.toLowerCase().startsWith(partial)).sort();
}

/** The token being edited: the word containing the cursor, where whitespace
 *  is the separator. */
function tokenAtCursor(value: string, cursor: number): string {
  const before = value.slice(0, cursor);
  const after = value.slice(cursor);
  const start = before.search(/\S*$/);
  const end = after.search(/\s|$/);
  return value.slice(start, cursor + end);
}

/** Replace the current token with `replacement`. When the token already
 *  contains a colon, only the value portion is replaced — so picking `pr`
 *  from the dropdown after typing `kind:` yields `kind:pr`, not bare `pr`.
 *  For key suggestions (no colon in the current token) the whole token is
 *  replaced with the suggestion (which already ends with `:`). */
function replaceCurrentToken(
  value: string,
  cursor: number,
  replacement: string
): { value: string; cursor: number } {
  const before = value.slice(0, cursor);
  const after = value.slice(cursor);
  const start = before.search(/\S*$/);
  const end = after.search(/\s|$/);
  const token = value.slice(start, cursor + end);
  const colon = token.indexOf(':');
  let newToken: string;
  if (colon >= 0) {
    newToken = token.slice(0, colon + 1) + replacement;
  } else {
    newToken = replacement;
  }
  const next = value.slice(0, start) + newToken + value.slice(cursor + end);
  return { value: next, cursor: start + newToken.length };
}

// repoNameFromPath extracts "<org>/<repo>" from a /r/<org%2Frepo>/... URL.
function repoNameFromPath(pathname: string): string | null {
  const m = pathname.match(/^\/r\/([^/]+)/);
  if (!m) return null;
  return safeDecode(m[1]);
}

function derivedValue(
  location: { search: string; pathname: string },
  repoName: string | null
): string {
  const sp = new URLSearchParams(location.search);
  const q = sp.get('q') ?? '';
  if (repoName && location.pathname.startsWith('/r/')) {
    const base = baseRepoName(repoName);
    const prefix = `repo:${base}`;
    if (!q) return `${prefix} status:open`;
    if (q.includes(prefix)) return q;
    return `${prefix} ${q}`;
  }
  return q;
}

function baseRepoName(full: string): string {
  const slash = full.lastIndexOf('/');
  return slash >= 0 ? full.slice(slash + 1) : full;
}

function safeDecode(s: string): string {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
}
