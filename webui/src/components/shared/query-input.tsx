// Generic syntax-highlighted search input with pluggable autocomplete providers.
//
// Architecture: two layers share the same font/padding so they appear identical:
//   1. A "backdrop" div (aria-hidden) renders colored <span>s for each token.
//   2. The real <input> floats on top with transparent text and bg, so the caret
//      is visible but the text itself is hidden in favour of the backdrop.

import {
  useFloating,
  useDismiss,
  useListNavigation,
  useInteractions,
  offset,
  flip,
  size,
  FloatingPortal,
} from "@floating-ui/react";
import {
  createContext,
  useContext,
  useState,
  useRef,
  useMemo,
  useEffect,
  type ChangeEvent,
  type ReactNode,
} from "react";

import * as Listbox from "@/components/ui/listbox";
import { cn } from "@/lib/utils";

// ── Public types ────────��──────────────────────��──────────────────────────────

export interface Suggestion {
  /** What gets inserted into the input (already quoted if needed). */
  value: string;
  /** Display label shown in the dropdown. */
  label: string;
  /** Optional leading decoration (icon, color dot, etc.). */
  icon?: ReactNode;
  /** Optional right-aligned secondary text. */
  description?: string;
}

export interface CompletionProvider {
  /** The prefix this provider handles, including the colon (e.g. "label:"). */
  prefix: string;
  /** Tailwind classes for syntax-highlighting this prefix in the backdrop. */
  highlightClass: string;
  /** Return suggestions for the partial query typed after the prefix. */
  getSuggestions(query: string): Suggestion[] | Promise<Suggestion[]>;
}

/** Static syntax rules for tokens that aren't completable but should be colored. */
export interface SyntaxRule {
  /** Exact token match (e.g. "status:open") or a prefix match (e.g. "sort:"). */
  match: string | ((token: string) => boolean);
  /** Tailwind classes for the prefix/key portion. */
  highlightClass: string;
}

// ── Defaults ────���────────────────────────────���────────────────────────────────

const DEFAULT_SYNTAX_RULES: SyntaxRule[] = [
  { match: "status:open", highlightClass: "text-green-600 dark:text-green-400" },
  { match: "status:closed", highlightClass: "text-purple-600 dark:text-purple-400" },
  { match: (t) => t.startsWith("sort:"), highlightClass: "text-orange-600 dark:text-orange-400" },
];

// ── Segment parsing ────────���──────────────────────────────��───────────────────

interface Segment {
  text: string;
  highlightClass: string | null;
}

function parseSegments(
  input: string,
  providers: CompletionProvider[],
  syntaxRules: SyntaxRule[],
): Segment[] {
  const segments: Segment[] = [];
  let i = 0;

  while (i < input.length) {
    // Whitespace runs
    if (input[i] === " ") {
      let j = i;
      while (j < input.length && input[j] === " ") j++;
      segments.push({ text: input.slice(i, j), highlightClass: null });
      i = j;
      continue;
    }

    // Token — consume until an unquoted space
    let j = i;
    let inQuote = false;
    while (j < input.length) {
      if (input[j] === '"') {
        inQuote = !inQuote;
        j++;
        continue;
      }
      if (!inQuote && input[j] === " ") break;
      j++;
    }

    const token = input.slice(i, j);

    // Check providers first (they also define syntax highlighting)
    let highlightClass: string | null = null;
    for (const p of providers) {
      if (token.startsWith(p.prefix)) {
        highlightClass = p.highlightClass;
        break;
      }
    }

    // Then check static syntax rules
    if (!highlightClass) {
      for (const rule of syntaxRules) {
        const matches = typeof rule.match === "string" ? token === rule.match : rule.match(token);
        if (matches) {
          highlightClass = rule.highlightClass;
          break;
        }
      }
    }

    segments.push({ text: token, highlightClass });
    i = j;
  }

  return segments;
}

function renderSegment(seg: Segment, i: number): ReactNode {
  if (!seg.highlightClass) {
    return <span key={i}>{seg.text}</span>;
  }
  const colon = seg.text.indexOf(":");
  if (colon === -1) {
    return (
      <span key={i} className={seg.highlightClass}>
        {seg.text}
      </span>
    );
  }
  const key = seg.text.slice(0, colon + 1);
  const val = seg.text.slice(colon + 1);
  return (
    <span key={i}>
      <span className={seg.highlightClass}>{key}</span>
      <span>{val}</span>
    </span>
  );
}

// ── Cursor / token utilities ──────────────────────────────────────────────────

interface CompletionInfo {
  provider: CompletionProvider;
  query: string;
  tokenStart: number;
}

function getCompletionInfo(
  value: string,
  cursor: number,
  providers: CompletionProvider[],
): CompletionInfo | null {
  let tokenStart = 0;
  for (let i = cursor - 1; i >= 0; i--) {
    if (value[i] === " ") {
      tokenStart = i + 1;
      break;
    }
  }

  const partial = value.slice(tokenStart, cursor);
  for (const provider of providers) {
    if (partial.startsWith(provider.prefix)) {
      const query = partial.slice(provider.prefix.length).replace(/^"/, "");
      return { provider, query, tokenStart };
    }
  }
  return null;
}

function getTokenEnd(value: string, tokenStart: number): number {
  let inQuote = false;
  for (let i = tokenStart; i < value.length; i++) {
    if (value[i] === '"') {
      inQuote = !inQuote;
      continue;
    }
    if (!inQuote && value[i] === " ") return i;
  }
  return value.length;
}

// ── Context ───────��───────────────────────────────────���───────────────────────

interface QueryInputContextValue {
  value: string;
  segments: Segment[];
  inputRef: React.RefObject<HTMLInputElement | null>;
  suggestions: Suggestion[];
  activeIndex: number | null;
  showDropdown: boolean;
  loading: boolean;
  handleChange: (e: ChangeEvent<HTMLInputElement>) => void;
  handleKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
  handleSelect: (e: React.SyntheticEvent<HTMLInputElement>) => void;
  selectSuggestion: (index: number) => void;
  // floating-ui
  floatingRef: (node: HTMLElement | null) => void;
  floatingStyles: React.CSSProperties;
  getFloatingProps: () => Record<string, unknown>;
  getItemProps: (userProps?: Record<string, unknown>) => Record<string, unknown>;
  elementsRef: React.MutableRefObject<(HTMLElement | null)[]>;
}

const QueryInputContext = createContext<QueryInputContextValue | null>(null);

function useQueryInput() {
  const ctx = useContext(QueryInputContext);
  if (!ctx) throw new Error("QueryInput sub-components must be used within QueryInput.Root");
  return ctx;
}

// ── Components ───────────���─────────────────────────────��──────────────────────

interface RootProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
  providers?: CompletionProvider[];
  syntaxRules?: SyntaxRule[];
  className?: string;
  children: ReactNode;
}

export function Root({
  value,
  onChange,
  onSubmit,
  providers = [],
  syntaxRules = DEFAULT_SYNTAX_RULES,
  className,
  children,
}: RootProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [completion, setCompletion] = useState<CompletionInfo | null>(null);
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [loading, setLoading] = useState(false);

  const elementsRef = useRef<(HTMLElement | null)[]>([]);

  const showDropdown = suggestions.length > 0;

  const segments = useMemo(
    () => parseSegments(value, providers, syntaxRules),
    [value, providers, syntaxRules],
  );

  // floating-ui for dropdown positioning
  const { refs, floatingStyles, context } = useFloating({
    open: showDropdown || loading,
    onOpenChange(nextOpen) {
      if (!nextOpen) {
        setCompletion(null);
        setSuggestions([]);
        setLoading(false);
      }
    },
    placement: "bottom-start",
    middleware: [
      offset(4),
      flip(),
      size({
        apply({ rects, elements }) {
          Object.assign(elements.floating.style, {
            minWidth: `${rects.reference.width}px`,
          });
        },
      }),
    ],
  });

  const dismiss = useDismiss(context, { escapeKey: true, outsidePress: true });
  const listNav = useListNavigation(context, {
    listRef: elementsRef,
    activeIndex,
    onNavigate: setActiveIndex,
    loop: true,
    virtual: true,
    focusItemOnOpen: false,
  });

  const { getReferenceProps, getFloatingProps, getItemProps } = useInteractions([dismiss, listNav]);

  // Fetch suggestions when completion changes.  Clearing them when there is no
  // completion left is done wherever the completion is cleared, so this effect
  // only ever has fetching to do.
  useEffect(() => {
    if (!completion) return;

    let cancelled = false;
    const result = completion.provider.getSuggestions(completion.query);

    if (result instanceof Promise) {
      // Entering the loading state is what synchronising with an async provider
      // looks like: there is no event to hang it off, and the request has to be
      // in flight before we know anything to derive it from.
      // oxlint-disable-next-line react/set-state-in-effect
      setLoading(true);
      void result.then((items) => {
        if (!cancelled) {
          setSuggestions(items);
          setActiveIndex(items.length > 0 ? 0 : null);
          setLoading(false);
        }
      });
    } else {
      setSuggestions(result);
      setActiveIndex(result.length > 0 ? 0 : null);
      setLoading(false);
    }

    return () => {
      cancelled = true;
    };
  }, [completion]);

  function updateCompletion(newValue: string, cursor: number) {
    const info = getCompletionInfo(newValue, cursor, providers);
    setCompletion(info);
    setActiveIndex(null);
    if (!info) {
      // Nothing completable at the cursor any more — drop what was showing.
      setSuggestions([]);
      setLoading(false);
    }
  }

  function handleChange(e: ChangeEvent<HTMLInputElement>) {
    const newValue = e.target.value;
    const cursor = e.target.selectionStart ?? newValue.length;
    onChange(newValue);
    updateCompletion(newValue, cursor);
  }

  function handleSelect(e: React.SyntheticEvent<HTMLInputElement>) {
    updateCompletion(value, e.currentTarget.selectionStart ?? value.length);
  }

  function applySuggestion(s: Suggestion) {
    if (!completion) return;
    const tokenEnd = getTokenEnd(value, completion.tokenStart);
    const completedToken = `${completion.provider.prefix}${s.value}`;
    const newValue =
      value.slice(0, completion.tokenStart) +
      completedToken +
      " " +
      value.slice(tokenEnd).trimStart();
    onChange(newValue);
    setCompletion(null);
    setSuggestions([]);
    setLoading(false);

    const newCursor = completion.tokenStart + completedToken.length + 1;
    requestAnimationFrame(() => {
      inputRef.current?.focus();
      inputRef.current?.setSelectionRange(newCursor, newCursor);
    });
  }

  function selectSuggestion(index: number) {
    const s = suggestions[index];
    if (s) applySuggestion(s);
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      if (activeIndex != null && suggestions.length > 0) {
        e.preventDefault();
        selectSuggestion(activeIndex);
      } else if (!completion) {
        e.preventDefault();
        onSubmit();
      }
      return;
    }

    if (e.key === "Tab" && activeIndex != null && suggestions.length > 0) {
      e.preventDefault();
      selectSuggestion(activeIndex);
    }
  }

  const ctx: QueryInputContextValue = {
    value,
    segments,
    inputRef,
    suggestions,
    activeIndex,
    showDropdown,
    loading,
    handleChange,
    handleKeyDown,
    handleSelect,
    selectSuggestion,
    floatingRef: refs.setFloating,
    floatingStyles,
    getFloatingProps,
    getItemProps,
    elementsRef,
  };

  return (
    <QueryInputContext value={ctx}>
      <div
        ref={refs.setReference}
        className={cn(
          "relative flex flex-1 items-center rounded-md border border-input bg-background",
          "ring-offset-background focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2",
          className,
        )}
        onClick={() => inputRef.current?.focus()}
        {...getReferenceProps()}
      >
        {children}
      </div>
    </QueryInputContext>
  );
}

interface IconProps {
  children: ReactNode;
}

export function Icon({ children }: IconProps) {
  return (
    <div className="text-muted-foreground pointer-events-none absolute left-3 flex size-4 shrink-0 items-center justify-center [&>svg]:size-4">
      {children}
    </div>
  );
}

interface InputProps {
  placeholder?: string;
  className?: string;
}

export function Input({ placeholder, className }: InputProps) {
  const {
    value,
    segments,
    inputRef,
    handleChange,
    handleKeyDown,
    handleSelect,
    activeIndex,
    suggestions,
  } = useQueryInput();

  return (
    <>
      {/* Colored backdrop */}
      <div
        aria-hidden
        className="text-foreground pointer-events-none absolute inset-0 flex items-center overflow-hidden pr-3 pl-9 font-mono text-sm whitespace-pre"
      >
        {value === "" ? null : segments.map((seg, i) => renderSegment(seg, i))}
      </div>

      {/* Actual input */}
      <input
        ref={inputRef}
        type="text"
        value={value}
        placeholder={placeholder}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        onSelect={handleSelect}
        role="combobox"
        aria-expanded={suggestions.length > 0}
        aria-activedescendant={activeIndex != null ? `query-option-${activeIndex}` : undefined}
        className={cn(
          "caret-foreground placeholder:text-muted-foreground relative w-full bg-transparent py-2 pr-3 pl-9 font-mono text-sm text-transparent outline-hidden placeholder:font-sans",
          className,
        )}
        spellCheck={false}
        autoComplete="off"
      />
    </>
  );
}

export function Completions() {
  const {
    suggestions,
    activeIndex,
    showDropdown,
    loading,
    selectSuggestion,
    floatingRef,
    floatingStyles,
    getFloatingProps,
    getItemProps,
    elementsRef,
  } = useQueryInput();

  if (!showDropdown && !loading) return null;

  return (
    <FloatingPortal>
      <Listbox.Content ref={floatingRef} style={floatingStyles} {...getFloatingProps()}>
        {loading && suggestions.length === 0 && (
          <div className="text-muted-foreground px-3 py-2 text-sm">Loading…</div>
        )}
        {suggestions.map((s, i) => (
          <Listbox.Item
            key={`${s.value}-${s.label}`}
            id={`query-option-${i}`}
            ref={(el) => {
              elementsRef.current[i] = el;
            }}
            active={activeIndex === i}
            onMouseDown={(e) => {
              e.preventDefault();
              selectSuggestion(i);
            }}
            {...getItemProps()}
          >
            {s.icon}
            <span className="font-mono">{s.label}</span>
            {s.description && (
              <span className="text-muted-foreground ml-auto text-xs">{s.description}</span>
            )}
          </Listbox.Item>
        ))}
      </Listbox.Content>
    </FloatingPortal>
  );
}
