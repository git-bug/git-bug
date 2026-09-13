import { InMemoryCache } from "@apollo/client";
import type { MockedResponse } from "@apollo/client/testing";
import { MockedProvider } from "@apollo/client/testing/react";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";

import { CommitListDocument } from "@/__generated__/graphql";
import { typePolicies } from "@/lib/apollo";

import { CommitList } from "./commit-list";

// Link uses TanStack Router — replace with a plain anchor for these tests.
vi.mock("@tanstack/react-router", async (importOriginal) => {
  const mod = await importOriginal<typeof import("@tanstack/react-router")>();
  return {
    ...mod,
    Link: ({
      children,
      className,
      to,
    }: {
      children: React.ReactNode;
      className?: string;
      to: string;
    }) => (
      <a href={to} className={className}>
        {children}
      </a>
    ),
  };
});

const PAGE_SIZE = 30;

// Mocked results carry __typename because the generated document selects it:
// without it the fetchMore result cannot be written to the cache, the original
// query stops being satisfiable and Apollo silently re-issues it.
function makeCommit(i: number) {
  return {
    __typename: "GitCommit" as const,
    hash: `hash${i}`,
    shortHash: `h${i}`,
    message: `Commit message ${i}`,
    authorName: "Alice",
    date: "2024-06-01T10:00:00Z",
  };
}

function makeQueryMock(
  commits: ReturnType<typeof makeCommit>[],
  hasNextPage: boolean,
  endCursor: string | null,
  variablesMatcher = vi.fn().mockReturnValue(true),
) {
  return {
    variablesMatcher,
    mock: {
      request: { query: CommitListDocument, variables: variablesMatcher },
      result: {
        data: {
          repository: {
            __typename: "Repository" as const,
            commits: {
              __typename: "GitCommitConnection" as const,
              nodes: commits,
              pageInfo: { __typename: "PageInfo" as const, hasNextPage, endCursor },
            },
          },
        },
      },
    },
  };
}

function renderList(
  props: { repo?: string | null; ref_?: string; path?: string } = {},
  mocks: MockedResponse<any, any>[] = [],
) {
  return render(
    // The cache has to be configured like the app's: without the Repository
    // type policy Apollo cannot resolve the entity across pages, and re-issues
    // the first query after every fetchMore.
    <MockedProvider mocks={mocks} showWarnings={false} cache={new InMemoryCache({ typePolicies })}>
      <CommitList repo={props.repo ?? "myrepo"} ref_={props.ref_ ?? "main"} path={props.path} />
    </MockedProvider>,
  );
}

// ── Loading / error states ────────────────────────────────────────────────────

describe("CommitList — loading", () => {
  it("renders a skeleton while the query is in flight", async () => {
    const { mock } = makeQueryMock([], false, null);
    renderList({}, [mock]);
    // Skeleton elements are present while loading
    expect(document.querySelectorAll(".animate-pulse").length).toBeGreaterThan(0);
    // Wait for skeleton to disappear (query resolved) so cleanup doesn't throw
    await waitFor(() => expect(document.querySelectorAll(".animate-pulse").length).toBe(0));
  });
});

describe("CommitList — error", () => {
  it("shows the error message on query failure", async () => {
    const mock = {
      request: { query: CommitListDocument, variables: vi.fn().mockReturnValue(true) },
      error: new Error("connection refused"),
    };
    renderList({}, [mock]);
    await waitFor(() => expect(screen.getByText(/connection refused/i)).toBeInTheDocument());
  });
});

// ── Commit rendering ──────────────────────────────────────────────────────────

describe("CommitList — commits", () => {
  it("renders commit messages", async () => {
    const { mock } = makeQueryMock([makeCommit(1), makeCommit(2)], false, null);
    renderList({}, [mock]);
    await waitFor(() => expect(screen.getByText("Commit message 1")).toBeInTheDocument());
    expect(screen.getByText("Commit message 2")).toBeInTheDocument();
  });

  it("groups commits under a date heading", async () => {
    const { mock } = makeQueryMock([makeCommit(1)], false, null);
    renderList({}, [mock]);
    await waitFor(() => expect(screen.getByText(/Commits on/i)).toBeInTheDocument());
  });

  it("renders short hashes as links", async () => {
    const { mock } = makeQueryMock([makeCommit(1)], false, null);
    renderList({}, [mock]);
    await waitFor(() => expect(screen.getByText("h1")).toBeInTheDocument());
  });
});

// ── Pagination ────────────────────────────────────────────────────────────────

describe("CommitList — pagination", () => {
  it("shows 'Load more commits' button when a full page is returned with a cursor", async () => {
    const commits = Array.from({ length: PAGE_SIZE }, (_, i) => makeCommit(i));
    const { mock } = makeQueryMock(commits, true, "cursor-1");
    renderList({}, [mock]);
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /load more commits/i })).toBeInTheDocument(),
    );
  });

  it("hides 'Load more commits' on an exactly-full last page (hasNextPage false)", async () => {
    // A repo with exactly PAGE_SIZE commits still returns an endCursor, so the
    // button must key off hasNextPage rather than the page being full.
    const commits = Array.from({ length: PAGE_SIZE }, (_, i) => makeCommit(i));
    const { mock } = makeQueryMock(commits, false, "cursor-1");
    renderList({}, [mock]);
    await waitFor(() => expect(screen.getByText("Commit message 0")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: /load more commits/i })).toBeNull();
  });

  it("hides 'Load more commits' when fewer than a full page returned", async () => {
    const { mock } = makeQueryMock([makeCommit(1), makeCommit(2)], false, null);
    renderList({}, [mock]);
    await waitFor(() => expect(screen.getByText("Commit message 1")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: /load more commits/i })).toBeNull();
  });

  it("fires fetchMore with the cursor when 'Load more' is clicked", async () => {
    const commits = Array.from({ length: PAGE_SIZE }, (_, i) => makeCommit(i));
    const { mock: firstMock } = makeQueryMock(commits, true, "cursor-1");
    const { variablesMatcher, mock: moreMock } = makeQueryMock([makeCommit(99)], false, null);

    renderList({}, [firstMock, moreMock]);

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /load more commits/i })).toBeInTheDocument(),
    );
    // Wrap click in act so React flushes setLoadingMore(true) before we drain
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: /load more commits/i }));
    });

    await waitFor(() => expect(variablesMatcher).toHaveBeenCalled());
    expect(variablesMatcher).toHaveBeenCalledWith(expect.objectContaining({ after: "cursor-1" }));
    // Drain: wait for fetchMore to fully settle (response delivered + React flushed)
    await act(async () => {
      await new Promise((r) => setTimeout(r, 20));
    });
  });

  it("appends the fetched page below the first one", async () => {
    const commits = Array.from({ length: PAGE_SIZE }, (_, i) => makeCommit(i));
    const { mock: firstMock } = makeQueryMock(commits, true, "cursor-1");
    const { mock: moreMock } = makeQueryMock([makeCommit(99)], false, null);

    renderList({}, [firstMock, moreMock]);

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /load more commits/i })).toBeInTheDocument(),
    );
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: /load more commits/i }));
    });

    // The fetched node is rendered, and the first page is still there with it
    await waitFor(() => expect(screen.getByText("Commit message 99")).toBeInTheDocument());
    expect(screen.getByText("Commit message 0")).toBeInTheDocument();
    expect(screen.getByText(`Commit message ${PAGE_SIZE - 1}`)).toBeInTheDocument();
  });

  it("stops offering 'Load more' when the fetched page reports no next page", async () => {
    const commits = Array.from({ length: PAGE_SIZE }, (_, i) => makeCommit(i));
    const { mock: firstMock } = makeQueryMock(commits, true, "cursor-1");
    // A full second page, but hasNextPage false: the button must follow pageInfo
    const secondPage = Array.from({ length: PAGE_SIZE }, (_, i) => makeCommit(100 + i));
    const { mock: moreMock } = makeQueryMock(secondPage, false, "cursor-2");

    renderList({}, [firstMock, moreMock]);

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /load more commits/i })).toBeInTheDocument(),
    );
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: /load more commits/i }));
    });

    await waitFor(() => expect(screen.getByText("Commit message 100")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: /load more commits/i })).toBeNull();
  });

  it("fetches the third page with the cursor the second page returned", async () => {
    const commits = Array.from({ length: PAGE_SIZE }, (_, i) => makeCommit(i));
    const { mock: firstMock } = makeQueryMock(commits, true, "cursor-1");
    const secondPage = Array.from({ length: PAGE_SIZE }, (_, i) => makeCommit(100 + i));
    const { mock: moreMock } = makeQueryMock(secondPage, true, "cursor-2");
    const { variablesMatcher: thirdMatcher, mock: thirdMock } = makeQueryMock(
      [makeCommit(200)],
      false,
      null,
    );

    renderList({}, [firstMock, moreMock, thirdMock]);

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /load more commits/i })).toBeInTheDocument(),
    );
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: /load more commits/i }));
    });

    // Still offered, because the second page reported another page after it
    await waitFor(() => expect(screen.getByText("Commit message 100")).toBeInTheDocument());
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: /load more commits/i }));
    });

    // The cursor must come from the accumulated page, not the first one
    await waitFor(() => expect(thirdMatcher).toHaveBeenCalled());
    expect(thirdMatcher).toHaveBeenCalledWith(expect.objectContaining({ after: "cursor-2" }));
    await waitFor(() => expect(screen.getByText("Commit message 200")).toBeInTheDocument());
    expect(screen.getByText("Commit message 0")).toBeInTheDocument();
  });
});
