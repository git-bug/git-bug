import {
  useFloating,
  useClick,
  useDismiss,
  useRole,
  useListNavigation,
  useTypeahead,
  useInteractions,
  offset,
  flip,
  FloatingPortal,
  FloatingFocusManager,
} from "@floating-ui/react";
import { ArrowUpDown, ChevronDown, Tag, User, X } from "lucide-react";
import { useMemo, useRef, useState, useCallback, useEffect } from "react";

import type { FragmentType } from "@/__generated__/fragment-masking";
import { LabelBadge, LABEL_FIELDS_FRAGMENT } from "@/components/shared/label-badge";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import * as Listbox from "@/components/ui/listbox";
import { useAuth } from "@/lib/auth";
import { SORT_OPTIONS, type SortValue } from "@/lib/query-utils";
import { cn } from "@/lib/utils";

// Max authors shown in the non-searching state. We intentionally cap this to
// avoid a giant list — the current-user + recently-seen pattern covers the
// common case; typing to search handles the rest.
const INITIAL_AUTHOR_LIMIT = 8;

// Returns the value passed to author:... in the query string.
// Preference order: login (never has spaces, safest) > name > humanId.
// We avoid humanId-as-query where possible because it's opaque to the user;
// the backend Match() also accepts login/name substring matches.
//
// Uses || (not ??) so that empty-string login/name fall through to the next
// option. git-bug identities can have login="" (empty, not null) when the
// login field was never set; ?? would return "" and the filter would silently
// produce author:"" which buildQueryString then drops, making the filter a no-op.
function authorQueryValue(i: {
  login?: string | null;
  name?: string | null;
  humanId: string;
}): string {
  return i.login || i.name || i.humanId;
}

export type { SortValue } from "@/lib/query-utils";

export type LabelItem = {
  name: string;
  color: { R: number; G: number; B: number };
} & FragmentType<typeof LABEL_FIELDS_FRAGMENT>;

export interface IdentityItem {
  id: string;
  humanId: string;
  name?: string | null;
  email?: string | null;
  login?: string | null;
  displayName: string;
  avatarUrl?: string | null;
}

interface IssueFiltersProps {
  labels: readonly LabelItem[];
  identities: readonly IdentityItem[];
  selectedLabels: string[];
  onLabelsChange: (labels: string[]) => void;
  selectedAuthorId: string | null;
  onAuthorChange: (humanId: string | null, queryValue: string | null) => void;
  /** humanIds of authors appearing in the current bug list, used to rank the initial suggestions */
  recentAuthorIds?: string[];
  sort: SortValue;
  onSortChange: (sort: SortValue) => void;
}

export function IssueFilters({
  labels,
  identities,
  selectedLabels,
  onLabelsChange,
  selectedAuthorId,
  onAuthorChange,
  recentAuthorIds = [],
  sort,
  onSortChange,
}: IssueFiltersProps) {
  const { user } = useAuth();

  const validLabels = useMemo(
    () => labels.toSorted((a, b) => a.name.localeCompare(b.name)),
    [labels],
  );

  const allIdentities = useMemo(
    () => identities.toSorted((a, b) => a.displayName.localeCompare(b.displayName)),
    [identities],
  );

  const selectedAuthorIdentity = allIdentities.find((i) => i.humanId === selectedAuthorId);

  return (
    <div className="flex shrink-0 items-center gap-1">
      <LabelFilter
        validLabels={validLabels}
        selectedLabels={selectedLabels}
        onLabelsChange={onLabelsChange}
      />
      <AuthorFilter
        allIdentities={allIdentities}
        selectedAuthorId={selectedAuthorId}
        selectedAuthorIdentity={selectedAuthorIdentity}
        onAuthorChange={onAuthorChange}
        recentAuthorIds={recentAuthorIds}
        currentUserId={user?.id ?? null}
      />
      <SortFilter sort={sort} onSortChange={onSortChange} />
    </div>
  );
}

// ── Sort filter ──────────────────────────────────────────────────────────────

function SortFilter({
  sort,
  onSortChange,
}: {
  sort: SortValue;
  onSortChange: (sort: SortValue) => void;
}) {
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const selectedIndex = SORT_OPTIONS.findIndex((o) => o.value === sort);

  const elementsRef = useRef<(HTMLElement | null)[]>([]);
  const labelsRef = useRef<(string | null)[]>([]);

  const { refs, floatingStyles, context } = useFloating({
    open,
    onOpenChange: setOpen,
    placement: "bottom-end",
    middleware: [offset(4), flip()],
  });

  const click = useClick(context);
  const dismiss = useDismiss(context);
  const role = useRole(context, { role: "listbox" });
  const listNav = useListNavigation(context, {
    listRef: elementsRef,
    activeIndex,
    selectedIndex,
    onNavigate: setActiveIndex,
    loop: true,
  });
  const typeahead = useTypeahead(context, {
    listRef: labelsRef,
    activeIndex,
    onMatch: setActiveIndex,
  });

  const { getReferenceProps, getFloatingProps, getItemProps } = useInteractions([
    click,
    dismiss,
    role,
    listNav,
    typeahead,
  ]);

  function handleSelect(index: number) {
    const opt = SORT_OPTIONS[index];
    if (opt) {
      onSortChange(opt.value);
      setOpen(false);
    }
  }

  return (
    <>
      <button
        ref={refs.setReference}
        className={cn(
          "flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium whitespace-nowrap transition-colors",
          sort !== "creation-desc"
            ? "bg-accent text-accent-foreground"
            : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
        )}
        {...getReferenceProps()}
      >
        <ArrowUpDown className="size-3.5" />
        {SORT_OPTIONS.find((o) => o.value === sort)?.label ?? "Sort"}
        <ChevronDown className="size-3" />
      </button>

      {open && (
        <FloatingPortal>
          <FloatingFocusManager context={context} modal={false}>
            <Listbox.Content ref={refs.setFloating} style={floatingStyles} {...getFloatingProps()}>
              <Listbox.ScrollArea>
                {SORT_OPTIONS.map((opt, i) => {
                  labelsRef.current[i] = opt.label;
                  return (
                    <Listbox.Item
                      key={opt.value}
                      ref={(el) => {
                        elementsRef.current[i] = el;
                      }}
                      active={activeIndex === i}
                      selected={sort === opt.value}
                      tabIndex={activeIndex === i ? 0 : -1}
                      className="whitespace-nowrap"
                      {...getItemProps({
                        onClick: () => handleSelect(i),
                      })}
                    >
                      {opt.label}
                    </Listbox.Item>
                  );
                })}
              </Listbox.ScrollArea>
            </Listbox.Content>
          </FloatingFocusManager>
        </FloatingPortal>
      )}
    </>
  );
}

// ── Label filter ─────────────────────────────────────────────────────────────

function LabelFilter({
  validLabels,
  selectedLabels,
  onLabelsChange,
}: {
  validLabels: readonly LabelItem[];
  selectedLabels: string[];
  onLabelsChange: (labels: string[]) => void;
}) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [activeIndex, setActiveIndex] = useState<number | null>(null);

  const elementsRef = useRef<(HTMLElement | null)[]>([]);
  const searchRef = useRef<HTMLInputElement>(null);

  const { refs, floatingStyles, context } = useFloating({
    open,
    onOpenChange(nextOpen) {
      setOpen(nextOpen);
      if (!nextOpen) setSearch("");
    },
    placement: "bottom-end",
    middleware: [offset(4), flip()],
  });

  const click = useClick(context);
  const dismiss = useDismiss(context);
  const role = useRole(context, { role: "listbox" });
  const listNav = useListNavigation(context, {
    listRef: elementsRef,
    activeIndex,
    onNavigate: setActiveIndex,
    loop: true,
    virtual: true,
    focusItemOnOpen: false,
  });

  const { getReferenceProps, getFloatingProps, getItemProps } = useInteractions([
    click,
    dismiss,
    role,
    listNav,
  ]);

  const filteredLabels = search.trim()
    ? validLabels.filter((l) => l.name.toLowerCase().includes(search.toLowerCase()))
    : [...validLabels];

  // Selected labels float to top
  const sortedLabels = [
    ...filteredLabels.filter((l) => selectedLabels.includes(l.name)),
    ...filteredLabels.filter((l) => !selectedLabels.includes(l.name)),
  ];

  // Reset active index when filtered list changes
  useEffect(() => {
    setActiveIndex(sortedLabels.length > 0 ? 0 : null);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- reset on search change
  }, [search]);

  function toggleLabel(name: string) {
    if (selectedLabels.includes(name)) {
      onLabelsChange(selectedLabels.filter((l) => l !== name));
    } else {
      onLabelsChange([...selectedLabels, name]);
    }
  }

  function handleSearchKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter" && activeIndex != null) {
      e.preventDefault();
      const label = sortedLabels[activeIndex];
      if (label) toggleLabel(label.name);
    }
  }

  return (
    <>
      <button
        ref={refs.setReference}
        className={cn(
          "flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
          selectedLabels.length > 0
            ? "bg-accent text-accent-foreground"
            : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
        )}
        {...getReferenceProps()}
      >
        <Tag className="size-3.5" />
        Labels
        {selectedLabels.length > 0 && (
          <span className="bg-muted rounded-full px-1.5 py-0.5 text-xs leading-none">
            {selectedLabels.length}
          </span>
        )}
        <ChevronDown className="size-3" />
      </button>

      {open && (
        <FloatingPortal>
          <FloatingFocusManager context={context} modal={false} initialFocus={searchRef}>
            <Listbox.Content ref={refs.setFloating} style={floatingStyles} {...getFloatingProps()}>
              <Listbox.Search
                ref={searchRef}
                placeholder="Search labels…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                onKeyDown={handleSearchKeyDown}
                aria-activedescendant={
                  activeIndex != null ? `label-option-${activeIndex}` : undefined
                }
              />
              <Listbox.ScrollArea>
                {sortedLabels.length === 0 && <Listbox.Empty>No labels found</Listbox.Empty>}
                {sortedLabels.map((label, i) => {
                  const active = selectedLabels.includes(label.name);
                  return (
                    <Listbox.Item
                      key={label.name}
                      id={`label-option-${i}`}
                      ref={(el) => {
                        elementsRef.current[i] = el;
                      }}
                      active={activeIndex === i}
                      selected={active}
                      {...getItemProps({
                        onClick: () => toggleLabel(label.name),
                      })}
                    >
                      <span
                        className="size-2 shrink-0 rounded-full"
                        style={{
                          backgroundColor: `rgb(${label.color.R},${label.color.G},${label.color.B})`,
                          opacity: active ? 1 : 0.35,
                        }}
                      />
                      <LabelBadge label={label} />
                    </Listbox.Item>
                  );
                })}
              </Listbox.ScrollArea>
              {selectedLabels.length > 0 && (
                <Listbox.Footer>
                  <Listbox.FooterButton onClick={() => onLabelsChange([])}>
                    <X className="size-3" />
                    Clear labels
                  </Listbox.FooterButton>
                </Listbox.Footer>
              )}
            </Listbox.Content>
          </FloatingFocusManager>
        </FloatingPortal>
      )}
    </>
  );
}

// ── Author filter ────────────────────────────────────────────────────────────

function AuthorFilter({
  allIdentities,
  selectedAuthorId,
  selectedAuthorIdentity,
  onAuthorChange,
  recentAuthorIds,
  currentUserId,
}: {
  allIdentities: readonly IdentityItem[];
  selectedAuthorId: string | null;
  selectedAuthorIdentity: IdentityItem | undefined;
  onAuthorChange: (humanId: string | null, queryValue: string | null) => void;
  recentAuthorIds: string[];
  currentUserId: string | null;
}) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [activeIndex, setActiveIndex] = useState<number | null>(null);

  const elementsRef = useRef<(HTMLElement | null)[]>([]);
  const searchRef = useRef<HTMLInputElement>(null);

  const { refs, floatingStyles, context } = useFloating({
    open,
    onOpenChange(nextOpen) {
      setOpen(nextOpen);
      if (!nextOpen) setSearch("");
    },
    placement: "bottom-end",
    middleware: [offset(4), flip()],
  });

  const click = useClick(context);
  const dismiss = useDismiss(context);
  const role = useRole(context, { role: "listbox" });
  const listNav = useListNavigation(context, {
    listRef: elementsRef,
    activeIndex,
    onNavigate: setActiveIndex,
    loop: true,
    virtual: true,
    focusItemOnOpen: false,
  });

  const { getReferenceProps, getFloatingProps, getItemProps } = useInteractions([
    click,
    dismiss,
    role,
    listNav,
  ]);

  const isSearching = search.trim() !== "";

  const matchesSearch = useCallback(
    (i: IdentityItem) => {
      const q = search.toLowerCase();
      return (
        i.displayName.toLowerCase().includes(q) ||
        (i.name ?? "").toLowerCase().includes(q) ||
        (i.login ?? "").toLowerCase().includes(q) ||
        (i.email ?? "").toLowerCase().includes(q)
      );
    },
    [search],
  );

  const visibleIdentities = useMemo(() => {
    if (isSearching) {
      return allIdentities.filter(matchesSearch);
    }

    const pinned = new Set<string>();
    const result: IdentityItem[] = [];

    // 1. Current user
    if (currentUserId) {
      const me = allIdentities.find((i) => i.id === currentUserId);
      if (me) {
        result.push(me);
        pinned.add(me.id);
      }
    }
    // 2. Selected author
    if (selectedAuthorId) {
      const sel = allIdentities.find((i) => i.humanId === selectedAuthorId);
      if (sel && !pinned.has(sel.id)) {
        result.push(sel);
        pinned.add(sel.id);
      }
    }
    // 3. Recently seen
    for (const humanId of recentAuthorIds) {
      const match = allIdentities.find((i) => i.humanId === humanId);
      if (match && !pinned.has(match.id)) {
        result.push(match);
        pinned.add(match.id);
      }
    }
    // 4. Fill to limit
    for (const i of allIdentities) {
      if (result.length >= INITIAL_AUTHOR_LIMIT) break;
      if (!pinned.has(i.id)) result.push(i);
    }
    return result;
  }, [allIdentities, isSearching, matchesSearch, currentUserId, selectedAuthorId, recentAuthorIds]);

  // Reset the active index when the filtered list changes.  Adjusting state
  // during render avoids the extra render pass an effect would cause.
  const [lastVisible, setLastVisible] = useState(visibleIdentities);
  if (visibleIdentities !== lastVisible) {
    setLastVisible(visibleIdentities);
    setActiveIndex(visibleIdentities.length > 0 ? 0 : null);
  }

  function handleSearchKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter" && activeIndex != null) {
      e.preventDefault();
      const identity = visibleIdentities[activeIndex];
      if (identity) {
        const isActive = selectedAuthorId === identity.humanId;
        onAuthorChange(
          isActive ? null : identity.humanId,
          isActive ? null : authorQueryValue(identity),
        );
        setOpen(false);
      }
    }
  }

  return (
    <>
      <button
        ref={refs.setReference}
        className={cn(
          "flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
          selectedAuthorId
            ? "bg-accent text-accent-foreground"
            : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
        )}
        {...getReferenceProps()}
      >
        {selectedAuthorIdentity ? (
          <>
            <Avatar className="size-4">
              <AvatarImage
                src={selectedAuthorIdentity.avatarUrl ?? undefined}
                alt={selectedAuthorIdentity.displayName}
              />
              <AvatarFallback className="text-[8px]">
                {selectedAuthorIdentity.displayName.slice(0, 2).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            {selectedAuthorIdentity.displayName}
          </>
        ) : (
          <>
            <User className="size-3.5" />
            Author
          </>
        )}
        <ChevronDown className="size-3" />
      </button>

      {open && (
        <FloatingPortal>
          <FloatingFocusManager context={context} modal={false} initialFocus={searchRef}>
            <Listbox.Content ref={refs.setFloating} style={floatingStyles} {...getFloatingProps()}>
              <Listbox.Search
                ref={searchRef}
                placeholder="Search authors…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                onKeyDown={handleSearchKeyDown}
                aria-activedescendant={
                  activeIndex != null ? `author-option-${activeIndex}` : undefined
                }
              />
              <Listbox.ScrollArea>
                {visibleIdentities.length === 0 && <Listbox.Empty>No authors found</Listbox.Empty>}
                {visibleIdentities.map((identity, i) => {
                  const active = selectedAuthorId === identity.humanId;
                  return (
                    <Listbox.Item
                      key={identity.id}
                      id={`author-option-${i}`}
                      ref={(el) => {
                        elementsRef.current[i] = el;
                      }}
                      active={activeIndex === i}
                      selected={active}
                      {...getItemProps({
                        onClick: () => {
                          onAuthorChange(
                            active ? null : identity.humanId,
                            active ? null : authorQueryValue(identity),
                          );
                          setOpen(false);
                        },
                      })}
                    >
                      <Avatar className="size-5 shrink-0">
                        <AvatarImage
                          src={identity.avatarUrl ?? undefined}
                          alt={identity.displayName}
                        />
                        <AvatarFallback className="text-[8px]">
                          {identity.displayName.slice(0, 2).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      <div className="min-w-0 flex-1 text-left">
                        <div className="truncate">{identity.displayName}</div>
                        {identity.login && identity.login !== identity.displayName && (
                          <div className="text-muted-foreground truncate text-xs">
                            @{identity.login}
                          </div>
                        )}
                      </div>
                    </Listbox.Item>
                  );
                })}
                {!isSearching && allIdentities.length > INITIAL_AUTHOR_LIMIT && (
                  <div className="text-muted-foreground px-2 py-1.5 text-center text-xs">
                    {allIdentities.length - visibleIdentities.length} more — type to search
                  </div>
                )}
              </Listbox.ScrollArea>
              {selectedAuthorId && (
                <Listbox.Footer>
                  <Listbox.FooterButton
                    onClick={() => {
                      onAuthorChange(null, null);
                      setOpen(false);
                    }}
                  >
                    <X className="size-3" />
                    Clear author
                  </Listbox.FooterButton>
                </Listbox.Footer>
              )}
            </Listbox.Content>
          </FloatingFocusManager>
        </FloatingPortal>
      )}
    </>
  );
}

// ── Sort filter ── (extracted above)
