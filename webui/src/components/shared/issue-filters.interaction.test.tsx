import { render, screen, fireEvent } from "@testing-library/react";
import { useState } from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";

import { useAuth } from "@/lib/auth";
import type { SortValue } from "@/lib/query-utils";

import type { IdentityItem, LabelItem } from "./issue-filters";
import { IssueFilters } from "./issue-filters";

// ── Auth mock ─────────────────────────────────────────────────────────────────

type MockUser = {
  id: string;
  humanId: string;
  displayName: string;
  avatarUrl: string | null;
} | null;
vi.mock("@/lib/auth", () => ({
  useAuth: vi.fn<() => { user: MockUser }>(() => ({
    user: { id: "current-user-id", humanId: "cu1", displayName: "Current User", avatarUrl: null },
  })),
}));

// ── Helpers ───────────────────────────────────────────────────────────────────

function makeLabel(name: string, R = 255, G = 0, B = 0): LabelItem {
  return { name, color: { R, G, B } } as LabelItem;
}

function makeIdentity(
  overrides: Partial<IdentityItem> & { id: string; humanId: string; displayName: string },
): IdentityItem {
  return { name: null, email: null, login: null, avatarUrl: null, ...overrides };
}

const DEFAULT_LABELS = [makeLabel("zebra"), makeLabel("alpha"), makeLabel("bug")];

const DEFAULT_IDENTITIES: IdentityItem[] = [
  makeIdentity({ id: "u1", humanId: "h1", displayName: "Alice" }),
  makeIdentity({ id: "u2", humanId: "h2", displayName: "Bob" }),
  makeIdentity({ id: "u3", humanId: "h3", displayName: "Carol" }),
];

interface RenderProps {
  labels?: LabelItem[];
  identities?: IdentityItem[];
  selectedLabels?: string[];
  onLabelsChange?: (labels: string[]) => void;
  selectedAuthorId?: string | null;
  onAuthorChange?: (humanId: string | null, queryValue: string | null) => void;
  recentAuthorIds?: string[];
  sort?: SortValue;
  onSortChange?: (sort: SortValue) => void;
}

function renderFilters(props: RenderProps = {}) {
  const onLabelsChange = props.onLabelsChange ?? vi.fn<(labels: string[]) => void>();
  const onAuthorChange =
    props.onAuthorChange ?? vi.fn<(humanId: string | null, queryValue: string | null) => void>();
  const onSortChange = props.onSortChange ?? vi.fn<(sort: SortValue) => void>();

  render(
    <IssueFilters
      labels={props.labels ?? DEFAULT_LABELS}
      identities={props.identities ?? DEFAULT_IDENTITIES}
      selectedLabels={props.selectedLabels ?? []}
      onLabelsChange={onLabelsChange}
      selectedAuthorId={props.selectedAuthorId ?? null}
      onAuthorChange={onAuthorChange}
      recentAuthorIds={props.recentAuthorIds ?? []}
      sort={props.sort ?? "creation-desc"}
      onSortChange={onSortChange}
    />,
  );

  return { onLabelsChange, onAuthorChange, onSortChange };
}

// floating-ui's useRole override leaves accessible names empty in happy-dom,
// so we select trigger buttons by DOM order: Labels[0], Author[1], Sort[2].
function openLabels() {
  fireEvent.click(screen.getAllByRole("combobox")[0]!);
}

function openAuthor() {
  fireEvent.click(screen.getAllByRole("combobox")[1]!);
}

function openSort() {
  fireEvent.click(screen.getAllByRole("combobox")[2]!);
}

function optionNames() {
  return screen.getAllByRole("option").map((el) => el.textContent?.trim());
}

// ── IssueFilters — top-level sorting ─────────────────────────────────────────

describe("IssueFilters — label sorting", () => {
  it("sorts labels alphabetically before passing to LabelFilter", () => {
    renderFilters({ labels: [makeLabel("zebra"), makeLabel("alpha"), makeLabel("bug")] });
    openLabels();
    const names = optionNames();
    expect(names).toEqual(["alpha", "bug", "zebra"]);
  });
});

describe("IssueFilters — identity sorting", () => {
  it("sorts identities by displayName before passing to AuthorFilter", () => {
    vi.mocked(useAuth).mockReturnValueOnce({ user: null });
    renderFilters({
      identities: [
        makeIdentity({ id: "u3", humanId: "h3", displayName: "Zara" }),
        makeIdentity({ id: "u1", humanId: "h1", displayName: "Alice" }),
        makeIdentity({ id: "u2", humanId: "h2", displayName: "Bob" }),
      ],
    });
    openAuthor();
    // textContent includes avatar fallback chars, so use includes() for matching
    const names = optionNames();
    expect(names[0]).toContain("Alice");
    expect(names[1]).toContain("Bob");
    expect(names[2]).toContain("Zara");
  });
});

// ── LabelFilter ───────────────────────────────────────────────────────────────

describe("LabelFilter — display", () => {
  it("shows label count badge when labels are selected", () => {
    renderFilters({ selectedLabels: ["bug", "alpha"] });
    expect(screen.getByText("2")).toBeInTheDocument();
  });

  it("hides label count badge when no labels are selected", () => {
    renderFilters({ selectedLabels: [] });
    expect(screen.queryByText(/^\d+$/)).toBeNull();
  });

  it("shows 'No labels found' when search matches nothing", () => {
    renderFilters();
    openLabels();
    fireEvent.change(screen.getByPlaceholderText("Search labels…"), {
      target: { value: "zzznomatch" },
    });
    expect(screen.getByText("No labels found")).toBeInTheDocument();
  });
});

describe("LabelFilter — filtering", () => {
  it("filters labels case-insensitively", () => {
    renderFilters({ labels: [makeLabel("Bug"), makeLabel("feature"), makeLabel("BugFix")] });
    openLabels();
    fireEvent.change(screen.getByPlaceholderText("Search labels…"), {
      target: { value: "bug" },
    });
    const names = optionNames();
    expect(names).toEqual(["Bug", "BugFix"]);
  });

  it("shows all labels when search is cleared", () => {
    renderFilters({ labels: [makeLabel("alpha"), makeLabel("bug")] });
    openLabels();
    const input = screen.getByPlaceholderText("Search labels…");
    fireEvent.change(input, { target: { value: "alpha" } });
    fireEvent.change(input, { target: { value: "" } });
    expect(optionNames()).toHaveLength(2);
  });
});

describe("LabelFilter — selected labels float to top", () => {
  it("places selected labels before unselected ones", () => {
    renderFilters({
      labels: [makeLabel("alpha"), makeLabel("bug"), makeLabel("zebra")],
      selectedLabels: ["zebra"],
    });
    openLabels();
    const names = optionNames();
    expect(names[0]).toBe("zebra");
    expect(names).toContain("alpha");
    expect(names).toContain("bug");
  });

  it("multiple selected labels all appear first", () => {
    renderFilters({
      labels: [makeLabel("alpha"), makeLabel("bug"), makeLabel("zebra")],
      selectedLabels: ["zebra", "bug"],
    });
    openLabels();
    const names = optionNames();
    expect(names.indexOf("zebra")).toBeLessThan(names.indexOf("alpha"));
    expect(names.indexOf("bug")).toBeLessThan(names.indexOf("alpha"));
  });
});

describe("LabelFilter — callbacks", () => {
  it("calls onLabelsChange with the new label added on click", () => {
    const { onLabelsChange } = renderFilters({ selectedLabels: ["bug"] });
    openLabels();
    // "alpha" is not yet selected — click it to add
    fireEvent.click(screen.getByRole("option", { name: /alpha/i }));
    expect(onLabelsChange).toHaveBeenCalledWith(expect.arrayContaining(["bug", "alpha"]));
  });

  it("calls onLabelsChange with the label removed on click", () => {
    const { onLabelsChange } = renderFilters({ selectedLabels: ["bug"] });
    openLabels();
    fireEvent.click(screen.getByRole("option", { name: /bug/i }));
    expect(onLabelsChange).toHaveBeenCalledWith([]);
  });

  it("shows Clear labels button only when labels are selected", () => {
    renderFilters({ selectedLabels: ["bug"] });
    openLabels();
    expect(screen.getByRole("button", { name: /clear labels/i })).toBeInTheDocument();
  });

  it("hides Clear labels button when no labels are selected", () => {
    renderFilters({ selectedLabels: [] });
    openLabels();
    expect(screen.queryByRole("button", { name: /clear labels/i })).toBeNull();
  });

  it("calls onLabelsChange([]) when Clear labels is clicked", () => {
    const { onLabelsChange } = renderFilters({ selectedLabels: ["bug"] });
    openLabels();
    fireEvent.click(screen.getByRole("button", { name: /clear labels/i }));
    expect(onLabelsChange).toHaveBeenCalledWith([]);
  });
});

// ── AuthorFilter — visibleIdentities priority ─────────────────────────────────

describe("AuthorFilter — priority pinning", () => {
  const CURRENT = makeIdentity({
    id: "current-user-id",
    humanId: "cu1",
    displayName: "Current User",
  });
  const SELECTED = makeIdentity({ id: "sel-id", humanId: "sel1", displayName: "Selected Author" });
  const RECENT = makeIdentity({ id: "rec-id", humanId: "rec1", displayName: "Recent Author" });
  const OTHERS = Array.from({ length: 6 }, (_, i) =>
    makeIdentity({ id: `other-${i}`, humanId: `o${i}`, displayName: `Other ${i}` }),
  );

  it("pins the current user first", () => {
    renderFilters({ identities: [OTHERS[0]!, CURRENT, OTHERS[1]!] });
    openAuthor();
    expect(optionNames()[0]).toContain("Current User");
  });

  it("pins the selected author second when different from current user", () => {
    renderFilters({
      identities: [SELECTED, CURRENT, OTHERS[0]!],
      selectedAuthorId: "sel1",
    });
    openAuthor();
    const names = optionNames();
    expect(names[0]).toContain("Current User");
    expect(names[1]).toContain("Selected Author");
  });

  it("does not duplicate the current user if also the selected author", () => {
    renderFilters({
      identities: [CURRENT, OTHERS[0]!],
      selectedAuthorId: CURRENT.humanId,
    });
    openAuthor();
    const names = optionNames();
    const currentCount = names.filter((n) => n?.includes("Current User")).length;
    expect(currentCount).toBe(1);
  });

  it("pins recent authors after current user and selected", () => {
    renderFilters({
      identities: [RECENT, CURRENT, OTHERS[0]!],
      recentAuthorIds: [RECENT.humanId],
    });
    openAuthor();
    const names = optionNames();
    expect(names[0]).toContain("Current User");
    expect(names[1]).toContain("Recent Author");
  });

  it("does not duplicate a recent author already pinned as current user", () => {
    renderFilters({
      identities: [CURRENT],
      recentAuthorIds: [CURRENT.humanId],
    });
    openAuthor();
    const names = optionNames();
    const currentCount = names.filter((n) => n?.includes("Current User")).length;
    expect(currentCount).toBe(1);
  });

  it("caps the visible list at INITIAL_AUTHOR_LIMIT (8)", () => {
    const many = Array.from({ length: 12 }, (_, i) =>
      makeIdentity({ id: `u${i}`, humanId: `h${i}`, displayName: `User ${i}` }),
    );
    renderFilters({ identities: many });
    openAuthor();
    expect(screen.getAllByRole("option")).toHaveLength(8);
  });

  it("shows 'X more — type to search' hint when list is capped", () => {
    const many = Array.from({ length: 12 }, (_, i) =>
      makeIdentity({ id: `u${i}`, humanId: `h${i}`, displayName: `User ${i}` }),
    );
    renderFilters({ identities: many });
    openAuthor();
    expect(screen.getByText(/more — type to search/i)).toBeInTheDocument();
  });

  it("hides the hint when the full list fits within the limit", () => {
    renderFilters({ identities: DEFAULT_IDENTITIES }); // only 3
    openAuthor();
    expect(screen.queryByText(/more — type to search/i)).toBeNull();
  });
});

// ── AuthorFilter — search ─────────────────────────────────────────────────────

describe("AuthorFilter — search", () => {
  const IDENTITIES = [
    makeIdentity({
      id: "u1",
      humanId: "h1",
      displayName: "Alice Smith",
      name: "Alice Smith",
      login: "asmith",
      email: "alice@example.com",
    }),
    makeIdentity({
      id: "u2",
      humanId: "h2",
      displayName: "Bob Jones",
      name: "Bob Jones",
      login: "bjones",
      email: "bob@example.com",
    }),
    makeIdentity({
      id: "u3",
      humanId: "h3",
      displayName: "Carol White",
      name: "Carol White",
      login: null,
      email: null,
    }),
  ];

  beforeEach(() => {
    vi.mocked(useAuth).mockReturnValue({ user: null });
  });

  it("filters by displayName", () => {
    renderFilters({ identities: IDENTITIES });
    openAuthor();
    fireEvent.change(screen.getByPlaceholderText("Search authors…"), {
      target: { value: "alice" },
    });
    expect(optionNames().some((n) => n?.includes("Alice"))).toBe(true);
    expect(optionNames().some((n) => n?.includes("Bob"))).toBe(false);
  });

  it("filters by login", () => {
    renderFilters({ identities: IDENTITIES });
    openAuthor();
    fireEvent.change(screen.getByPlaceholderText("Search authors…"), {
      target: { value: "bjones" },
    });
    expect(optionNames().some((n) => n?.includes("Bob"))).toBe(true);
    expect(optionNames().some((n) => n?.includes("Alice"))).toBe(false);
  });

  it("filters by email", () => {
    renderFilters({ identities: IDENTITIES });
    openAuthor();
    fireEvent.change(screen.getByPlaceholderText("Search authors…"), {
      target: { value: "alice@example" },
    });
    expect(optionNames().some((n) => n?.includes("Alice"))).toBe(true);
    expect(optionNames().some((n) => n?.includes("Bob"))).toBe(false);
  });

  it("bypasses the 8-item cap during search", () => {
    vi.mocked(useAuth).mockReturnValue({ user: null });
    const many = Array.from({ length: 12 }, (_, i) =>
      makeIdentity({
        id: `u${i}`,
        humanId: `h${i}`,
        displayName: `User ${String(i).padStart(2, "0")}`,
      }),
    );
    renderFilters({ identities: many });
    openAuthor();
    fireEvent.change(screen.getByPlaceholderText("Search authors…"), { target: { value: "User" } });
    expect(screen.getAllByRole("option")).toHaveLength(12);
  });

  it("shows 'No authors found' when search matches nothing", () => {
    renderFilters({ identities: IDENTITIES });
    openAuthor();
    fireEvent.change(screen.getByPlaceholderText("Search authors…"), {
      target: { value: "zzznomatch" },
    });
    expect(screen.getByText("No authors found")).toBeInTheDocument();
  });
});

// ── AuthorFilter — onAuthorChange callback ────────────────────────────────────

describe("AuthorFilter — onAuthorChange", () => {
  beforeEach(() => {
    vi.mocked(useAuth).mockReturnValue({ user: null });
  });

  it("selects an author and passes their humanId and query value", () => {
    const { onAuthorChange } = renderFilters({
      identities: [makeIdentity({ id: "u1", humanId: "h1", displayName: "Alice", login: "alice" })],
    });
    openAuthor();
    fireEvent.click(screen.getByRole("option", { name: /alice/i }));
    expect(onAuthorChange).toHaveBeenCalledWith("h1", "alice");
  });

  it("deselects the author (null, null) when clicking the already-selected author", () => {
    const { onAuthorChange } = renderFilters({
      identities: [makeIdentity({ id: "u1", humanId: "h1", displayName: "Alice" })],
      selectedAuthorId: "h1",
    });
    openAuthor();
    fireEvent.click(screen.getByRole("option", { name: /alice/i }));
    expect(onAuthorChange).toHaveBeenCalledWith(null, null);
  });

  it("Clear author button calls onAuthorChange(null, null)", () => {
    const { onAuthorChange } = renderFilters({
      identities: [makeIdentity({ id: "u1", humanId: "h1", displayName: "Alice" })],
      selectedAuthorId: "h1",
    });
    openAuthor();
    fireEvent.click(screen.getByRole("button", { name: /clear author/i }));
    expect(onAuthorChange).toHaveBeenCalledWith(null, null);
  });

  it("prefers login over name as the query value", () => {
    const { onAuthorChange } = renderFilters({
      identities: [
        makeIdentity({
          id: "u1",
          humanId: "h1",
          displayName: "Alice",
          login: "alice",
          name: "Alice Smith",
        }),
      ],
    });
    openAuthor();
    fireEvent.click(screen.getByRole("option", { name: /alice/i }));
    expect(onAuthorChange).toHaveBeenCalledWith("h1", "alice");
  });

  it("skips empty-string login and falls through to name", () => {
    // This is the || vs ?? bug: empty login="" must not be returned as the query value
    const { onAuthorChange } = renderFilters({
      identities: [
        makeIdentity({
          id: "u1",
          humanId: "h1",
          displayName: "Alice",
          login: "",
          name: "Alice Smith",
        }),
      ],
    });
    openAuthor();
    fireEvent.click(screen.getByRole("option", { name: /alice/i }));
    expect(onAuthorChange).toHaveBeenCalledWith("h1", "Alice Smith");
  });

  it("falls through to humanId when login and name are both empty/null", () => {
    const { onAuthorChange } = renderFilters({
      identities: [
        makeIdentity({ id: "u1", humanId: "h1", displayName: "Alice", login: null, name: null }),
      ],
    });
    openAuthor();
    fireEvent.click(screen.getByRole("option", { name: /alice/i }));
    expect(onAuthorChange).toHaveBeenCalledWith("h1", "h1");
  });
});

// ── SortFilter ────────────────────────────────────────────────────────────────

describe("SortFilter", () => {
  it("shows the current sort label on the button", () => {
    renderFilters({ sort: "edit-desc" });
    expect(screen.getAllByRole("combobox")[2]).toHaveTextContent("Recently updated");
  });

  it("calls onSortChange with the selected value", () => {
    const { onSortChange } = renderFilters({ sort: "creation-desc" });
    openSort();
    fireEvent.click(screen.getByRole("option", { name: /oldest/i }));
    expect(onSortChange).toHaveBeenCalledWith("creation-asc");
  });

  it("marks the current sort option as selected", () => {
    renderFilters({ sort: "creation-asc" });
    openSort();
    const oldest = screen.getByRole("option", { name: /oldest/i });
    expect(oldest).toHaveAttribute("aria-selected", "true");
  });
});

// ── AuthorFilter — stability across parent renders ───────────────────────────
//
// The issues page passes a freshly built `recentAuthorIds` array every render.
// Comparing that by identity resets the keyboard highlight mid-navigation.

describe("AuthorFilter — parent re-renders", () => {
  function Harness() {
    const [tick, setTick] = useState(0);
    return (
      <>
        <button data-testid="rerender" onClick={() => setTick((t) => t + 1)}>
          {tick}
        </button>
        <IssueFilters
          labels={DEFAULT_LABELS}
          identities={DEFAULT_IDENTITIES}
          selectedLabels={[]}
          onLabelsChange={() => {}}
          selectedAuthorId={null}
          onAuthorChange={() => {}}
          // rebuilt every render, like the issues page does
          recentAuthorIds={DEFAULT_IDENTITIES.map((i) => i.humanId)}
          sort="creation-desc"
          onSortChange={() => {}}
        />
      </>
    );
  }

  it("keeps the keyboard highlight when the parent re-renders", () => {
    render(<Harness />);
    openAuthor();
    const listbox = screen.getAllByRole("listbox")[0]!;
    fireEvent.keyDown(listbox, { key: "ArrowDown" });
    fireEvent.keyDown(listbox, { key: "ArrowDown" });
    const active = listbox.getAttribute("aria-activedescendant");
    expect(active).toBe("author-option-1");

    fireEvent.click(screen.getByTestId("rerender"));

    expect(screen.getAllByRole("listbox")[0]).toHaveAttribute("aria-activedescendant", active!);
  });

  it("still resets the highlight when the search actually filters the list", () => {
    render(<Harness />);
    openAuthor();
    const listbox = screen.getAllByRole("listbox")[0]!;
    fireEvent.keyDown(listbox, { key: "ArrowDown" });
    fireEvent.keyDown(listbox, { key: "ArrowDown" });
    expect(listbox).toHaveAttribute("aria-activedescendant", "author-option-1");

    fireEvent.change(screen.getByPlaceholderText(/search/i), { target: { value: "Bob" } });

    expect(screen.getAllByRole("listbox")[0]).toHaveAttribute(
      "aria-activedescendant",
      "author-option-0",
    );
  });
});
