// Query string utilities for the bug filter system.
// Handles building and parsing structured filter queries like:
//   "status:open label:bug author:janedoe sort:creation-desc some free text"

export type StatusFilter = "open" | "closed";

export type SortValue = "creation-desc" | "creation-asc" | "edit-desc" | "edit-asc";

export const SORT_OPTIONS: { value: SortValue; label: string }[] = [
  { value: "creation-desc", label: "Newest" },
  { value: "creation-asc", label: "Oldest" },
  { value: "edit-desc", label: "Recently updated" },
  { value: "edit-asc", label: "Least recently updated" },
];

const VALID_SORTS = new Set<string>(["creation-desc", "creation-asc", "edit-desc", "edit-asc"]);

function isValidSort(val: string): val is SortValue {
  return VALID_SORTS.has(val);
}

// Quote a qualifier value when it contains whitespace or a colon, which the
// backend query lexer would otherwise treat as a separator (e.g. label "type:bug").
export function quoteQueryValue(value: string): string {
  return /[\s:]/.test(value) ? `"${value}"` : value;
}

// Tokenize a query string, keeping quoted spans as single tokens.
export function tokenizeQuery(input: string): string[] {
  const tokens: string[] = [];
  let current = "";
  let inQuote = false;
  for (const ch of input.trim()) {
    if (ch === '"') {
      inQuote = !inQuote;
      current += ch;
    } else if (ch === " " && !inQuote) {
      if (current) {
        tokens.push(current);
        current = "";
      }
    } else current += ch;
  }
  if (current) tokens.push(current);
  return tokens;
}

// Parse a query string back into structured filter state.
export function parseQueryString(input: string): {
  status: StatusFilter | null;
  labels: string[];
  author: string | null;
  freeText: string;
  sort: SortValue;
} {
  let status: StatusFilter | null = null;
  const labels: string[] = [];
  let author: string | null = null;
  let sort: SortValue = "creation-desc";
  const free: string[] = [];

  for (const token of tokenizeQuery(input)) {
    if (token === "status:open") status = "open";
    else if (token === "status:closed") status = "closed";
    else if (token.startsWith("label:")) labels.push(token.slice(6).replace(/^"|"$/g, ""));
    else if (token.startsWith("author:")) author = token.slice(7).replace(/^"|"$/g, "");
    else if (token.startsWith("sort:")) {
      const val = token.slice(5);
      if (isValidSort(val)) sort = val;
    } else free.push(token);
  }

  return { status, labels, author, freeText: free.join(" "), sort };
}

// Returns the filter parts (labels, author, freeText) without the status prefix,
// so it can be combined with "status:open" / "status:closed".
export function buildBaseQuery(labels: string[], author: string | null, freeText: string): string {
  const parts: string[] = [];
  for (const label of labels) {
    parts.push(`label:${quoteQueryValue(label)}`);
  }
  if (author) {
    parts.push(`author:${quoteQueryValue(author)}`);
  }
  if (freeText.trim()) parts.push(freeText.trim());
  return parts.join(" ");
}

// Build the structured query string sent to the GraphQL allBugs(query:) argument.
export function buildQueryString(
  status: StatusFilter | null,
  labels: string[],
  author: string | null,
  freeText: string,
  sort: SortValue = "creation-desc",
): string {
  const parts: string[] = [];
  if (status) parts.push(`status:${status}`);
  const base = buildBaseQuery(labels, author, freeText);
  if (base) parts.push(base);
  if (sort !== "creation-desc") parts.push(`sort:${sort}`);
  return parts.join(" ");
}
