import { describe, expect, it } from "vitest";

import {
  buildBaseQuery,
  buildQueryString,
  parseQueryString,
  quoteQueryValue,
  tokenizeQuery,
} from "./query-utils";

// ─── tokenizeQuery ────────────────────────────────────────────────────────────

describe("tokenizeQuery", () => {
  it("returns empty array for empty string", () => {
    expect(tokenizeQuery("")).toEqual([]);
  });

  it("returns empty array for whitespace-only string", () => {
    expect(tokenizeQuery("   ")).toEqual([]);
  });

  it("splits on spaces", () => {
    expect(tokenizeQuery("foo bar baz")).toEqual(["foo", "bar", "baz"]);
  });

  it("collapses multiple spaces", () => {
    expect(tokenizeQuery("foo  bar   baz")).toEqual(["foo", "bar", "baz"]);
  });

  it("trims leading and trailing whitespace", () => {
    expect(tokenizeQuery("  foo bar  ")).toEqual(["foo", "bar"]);
  });

  it("keeps a double-quoted span as a single token", () => {
    expect(tokenizeQuery('label:"bug fix"')).toEqual(['label:"bug fix"']);
  });

  it("keeps spaces inside quotes", () => {
    expect(tokenizeQuery('"hello world" foo')).toEqual(['"hello world"', "foo"]);
  });

  it("handles an unterminated quote by treating the rest as one token", () => {
    expect(tokenizeQuery('label:"unclosed')).toEqual(['label:"unclosed']);
  });

  it("handles adjacent quoted and unquoted tokens", () => {
    expect(tokenizeQuery('status:open label:"needs triage" foo')).toEqual([
      "status:open",
      'label:"needs triage"',
      "foo",
    ]);
  });
});

// ─── parseQueryString ─────────────────────────────────────────────────────────

describe("parseQueryString", () => {
  it("returns defaults for empty input", () => {
    expect(parseQueryString("")).toEqual({
      status: null,
      labels: [],
      author: null,
      freeText: "",
      sort: "creation-desc",
    });
  });

  it("parses status:open", () => {
    expect(parseQueryString("status:open").status).toBe("open");
  });

  it("parses status:closed", () => {
    expect(parseQueryString("status:closed").status).toBe("closed");
  });

  it("ignores an invalid status value", () => {
    expect(parseQueryString("status:pending").status).toBe(null);
  });

  it("last status value wins when duplicated", () => {
    expect(parseQueryString("status:open status:closed").status).toBe("closed");
  });

  it("parses a single unquoted label", () => {
    expect(parseQueryString("label:bug").labels).toEqual(["bug"]);
  });

  it("parses multiple labels", () => {
    expect(parseQueryString("label:bug label:urgent").labels).toEqual(["bug", "urgent"]);
  });

  it("strips quotes from a quoted label", () => {
    // label:"bug fix" should produce the label value 'bug fix', not '"bug fix"'
    expect(parseQueryString('label:"bug fix"').labels).toEqual(["bug fix"]);
  });

  it("round-trips a label containing a colon", () => {
    const q = buildBaseQuery(["type:bug"], null, "");
    expect(q).toBe('label:"type:bug"');
    expect(parseQueryString(q).labels).toEqual(["type:bug"]);
  });

  it("parses an unquoted author", () => {
    expect(parseQueryString("author:jane").author).toBe("jane");
  });

  it("strips quotes from a quoted author", () => {
    expect(parseQueryString('author:"jane doe"').author).toBe("jane doe");
  });

  it("last author value wins when duplicated", () => {
    expect(parseQueryString("author:alice author:bob").author).toBe("bob");
  });

  it("parses a valid sort value", () => {
    expect(parseQueryString("sort:creation-asc").sort).toBe("creation-asc");
  });

  it("ignores an invalid sort value, keeping default", () => {
    expect(parseQueryString("sort:bogus").sort).toBe("creation-desc");
  });

  it("collects free text tokens", () => {
    expect(parseQueryString("foo bar").freeText).toBe("foo bar");
  });

  it("does not treat unknown prefixed tokens as structured filters", () => {
    expect(parseQueryString("foo:bar").freeText).toBe("foo:bar");
  });

  it("parses a combined query correctly", () => {
    expect(
      parseQueryString('status:open label:bug author:"jane doe" sort:edit-desc some text'),
    ).toEqual({
      status: "open",
      labels: ["bug"],
      author: "jane doe",
      freeText: "some text",
      sort: "edit-desc",
    });
  });
});

// ─── buildBaseQuery ───────────────────────────────────────────────────────────

describe("quoteQueryValue", () => {
  it("leaves plain values unquoted", () => {
    expect(quoteQueryValue("bug")).toBe("bug");
  });

  it("quotes values containing a colon", () => {
    expect(quoteQueryValue("type:bug")).toBe('"type:bug"');
  });

  it("quotes values containing whitespace", () => {
    expect(quoteQueryValue("bug fix")).toBe('"bug fix"');
  });

  it("quotes values containing both", () => {
    expect(quoteQueryValue("type: needs triage")).toBe('"type: needs triage"');
  });
});

describe("buildBaseQuery", () => {
  it("returns empty string when all inputs are empty", () => {
    expect(buildBaseQuery([], null, "")).toBe("");
  });

  it("builds an unquoted label when no spaces", () => {
    expect(buildBaseQuery(["bug"], null, "")).toBe("label:bug");
  });

  it("quotes a label containing spaces", () => {
    expect(buildBaseQuery(["bug fix"], null, "")).toBe('label:"bug fix"');
  });

  it("builds multiple labels", () => {
    expect(buildBaseQuery(["bug", "needs triage"], null, "")).toBe(
      'label:bug label:"needs triage"',
    );
  });

  it("builds an unquoted author when no spaces", () => {
    expect(buildBaseQuery([], "jane", "")).toBe("author:jane");
  });

  it("quotes a label containing a colon", () => {
    expect(buildBaseQuery(["type:bug", "bug"], null, "")).toBe('label:"type:bug" label:bug');
  });

  it("quotes an author containing a colon", () => {
    expect(buildBaseQuery([], "a:b", "")).toBe('author:"a:b"');
  });

  it("quotes an author containing spaces", () => {
    expect(buildBaseQuery([], "jane doe", "")).toBe('author:"jane doe"');
  });

  it("appends free text", () => {
    expect(buildBaseQuery([], null, "some text")).toBe("some text");
  });

  it("ignores whitespace-only free text", () => {
    expect(buildBaseQuery([], null, "   ")).toBe("");
  });

  it("combines labels, author, and free text in order", () => {
    expect(buildBaseQuery(["bug"], "jane", "crash")).toBe("label:bug author:jane crash");
  });
});

// ─── buildQueryString ─────────────────────────────────────────────────────────

describe("buildQueryString", () => {
  it("returns empty string for all-null/empty inputs with default sort", () => {
    expect(buildQueryString(null, [], null, "")).toBe("");
  });

  it("includes status prefix when provided", () => {
    expect(buildQueryString("open", [], null, "")).toBe("status:open");
  });

  it("omits sort when it equals the default (creation-desc)", () => {
    expect(buildQueryString(null, [], null, "", "creation-desc")).not.toContain("sort:");
  });

  it("includes sort for non-default values", () => {
    expect(buildQueryString(null, [], null, "", "edit-asc")).toBe("sort:edit-asc");
  });

  it("combines all parts", () => {
    expect(buildQueryString("open", ["bug"], "jane", "crash", "creation-asc")).toBe(
      "status:open label:bug author:jane crash sort:creation-asc",
    );
  });
});

// ─── roundtrip: buildQueryString → parseQueryString ──────────────────────────

describe("roundtrip", () => {
  it("round-trips a simple open query", () => {
    const q = buildQueryString("open", [], null, "");
    expect(parseQueryString(q).status).toBe("open");
  });

  it("round-trips a closed query with a label", () => {
    const q = buildQueryString("closed", ["bug"], null, "");
    const parsed = parseQueryString(q);
    expect(parsed.status).toBe("closed");
    expect(parsed.labels).toEqual(["bug"]);
  });

  it("round-trips a multi-word label", () => {
    const q = buildQueryString(null, ["needs triage"], null, "");
    // buildBaseQuery produces label:"needs triage"
    // parseQueryString must strip the quotes to recover 'needs triage'
    const parsed = parseQueryString(q);
    expect(parsed.labels).toEqual(["needs triage"]);
  });

  it("round-trips a multi-word author", () => {
    const q = buildQueryString(null, [], "jane doe", "");
    expect(parseQueryString(q).author).toBe("jane doe");
  });

  it("round-trips a non-default sort", () => {
    const q = buildQueryString(null, [], null, "", "edit-desc");
    expect(parseQueryString(q).sort).toBe("edit-desc");
  });

  it("round-trips a complex query with all fields", () => {
    const input = {
      status: "open" as const,
      labels: ["bug", "needs triage"],
      author: "jane doe",
      freeText: "crash on login",
      sort: "edit-asc" as const,
    };
    const q = buildQueryString(
      input.status,
      input.labels,
      input.author,
      input.freeText,
      input.sort,
    );
    const parsed = parseQueryString(q);
    expect(parsed).toEqual(input);
  });
});
