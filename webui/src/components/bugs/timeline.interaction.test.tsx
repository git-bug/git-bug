import type { MockedResponse } from "@apollo/client/testing";
import { MockedProvider } from "@apollo/client/testing/react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { Suspense } from "react";
import { beforeEach, describe, it, expect, onTestFinished, vi } from "vitest";

import { makeFragmentData } from "@/__generated__/fragment-masking";
import { BugEditCommentDocument } from "@/__generated__/graphql";
import type { TimelineItemsFragment } from "@/__generated__/graphql";
import { toastManager, Toasts } from "@/components/ui/toast";
import { useAuth } from "@/lib/auth";

import { Timeline, TIMELINE_ITEMS_FRAGMENT } from "./timeline";

// Replace TanStack Router's Link with a plain <a> — timeline uses it only for
// author profile links, not for any logic under test.
vi.mock("@tanstack/react-router", async (importOriginal) => {
  const mod = await importOriginal<typeof import("@tanstack/react-router")>();
  return {
    ...mod,
    Link: ({ children, className }: { children: React.ReactNode; className?: string }) => (
      <a className={className}>{children}</a>
    ),
  };
});

// ── Auth mock ─────────────────────────────────────────────────────────────────

type MockUser = {
  id: string;
  humanId: string;
  displayName: string;
  avatarUrl: string | null;
} | null;
vi.mock("@/lib/auth", () => ({
  useAuth: vi.fn<() => { user: MockUser }>(() => ({
    user: { id: "user-1", humanId: "u1", displayName: "Test User", avatarUrl: null },
  })),
}));

// ── Toasts ────────────────────────────────────────────────────────────────────

const addToast = vi.spyOn(toastManager, "add");

beforeEach(() => {
  addToast.mockClear();
});

// ── Data helpers ──────────────────────────────────────────────────────────────

const AUTHOR_ME = {
  __typename: "Identity" as const,
  id: "user-1",
  humanId: "u1",
  displayName: "Test User",
  avatarUrl: null,
};

const AUTHOR_OTHER = {
  __typename: "Identity" as const,
  id: "user-2",
  humanId: "u2",
  displayName: "Other User",
  avatarUrl: null,
};

type AddCommentOverrides = Partial<{
  id: string;
  author: typeof AUTHOR_ME | typeof AUTHOR_OTHER;
  // the authors of the versions, by default the comment author alone
  editors: (typeof AUTHOR_ME | typeof AUTHOR_OTHER)[];
  message: string;
  createdAt: string;
  lastEdit: string;
  edited: boolean;
}>;

function makeAddComment({ editors, ...overrides }: AddCommentOverrides = {}) {
  const author = overrides.author ?? AUTHOR_ME;
  return {
    __typename: "BugAddCommentTimelineItem" as const,
    id: "comment-1",
    author,
    message: "Original message",
    createdAt: "2024-01-01T00:00:00Z",
    lastEdit: "2024-01-01T00:00:00Z",
    edited: (editors?.length ?? 1) > 1,
    history: (editors ?? [author]).map((a) => ({
      __typename: "BugCommentHistoryStep" as const,
      author: a,
    })),
    ...overrides,
  };
}

function makeTimeline(nodes: TimelineItemsFragment["nodes"]) {
  const data = { __typename: "BugTimelineItemConnection" as const, nodes };
  return makeFragmentData(data, TIMELINE_ITEMS_FRAGMENT);
}

function renderTimeline(
  nodes: TimelineItemsFragment["nodes"],
  mocks: MockedResponse<any, any>[] = [],
  repo: string | null = "myrepo",
) {
  return render(
    <MockedProvider mocks={mocks} showWarnings={false}>
      <Suspense>
        <Timeline repo={repo} timeline={makeTimeline(nodes)} />
      </Suspense>
    </MockedProvider>,
  );
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe("Timeline — edit button auth gate", () => {
  it("shows Edit button when the logged-in user authored the comment", () => {
    renderTimeline([makeAddComment({ author: AUTHOR_ME })]);
    expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
  });

  it("hides Edit button when a different user authored the comment", () => {
    renderTimeline([makeAddComment({ author: AUTHOR_OTHER })]);
    expect(screen.queryByRole("button", { name: "Edit" })).toBeNull();
  });

  it("hides Edit button when there is no logged-in user", async () => {
    // for every call: the timeline and each comment read the user
    vi.mocked(useAuth).mockReturnValue({ user: null });
    onTestFinished(() => {
      vi.mocked(useAuth).mockReset();
    });
    renderTimeline([makeAddComment()]);
    expect(screen.queryByRole("button", { name: "Edit" })).toBeNull();
  });
});

describe("Timeline — edited badge", () => {
  it("shows 'edited' badge when comment has been edited", () => {
    renderTimeline([makeAddComment({ edited: true })]);
    expect(screen.getByText("edited")).toBeInTheDocument();
  });

  it("hides 'edited' badge when comment has not been edited", () => {
    renderTimeline([makeAddComment({ edited: false })]);
    expect(screen.queryByText("edited")).toBeNull();
  });
});

describe("Timeline — edit flow", () => {
  it("opens textarea with current message on Edit click", () => {
    renderTimeline([makeAddComment({ message: "Original message" })]);
    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    expect(screen.getByRole("textbox")).toHaveValue("Original message");
  });

  it("Escape cancels and returns to read mode without mutating", async () => {
    renderTimeline([makeAddComment({ message: "Original" })]);
    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "Changed" } });
    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Escape" });

    expect(screen.queryByRole("textbox")).toBeNull();
    expect(screen.getByText("Original")).toBeInTheDocument();
  });

  it("Cancel button returns to read mode without mutating", () => {
    renderTimeline([makeAddComment({ message: "Original" })]);
    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

    expect(screen.queryByRole("textbox")).toBeNull();
  });

  it("does NOT fire mutation when the message is unchanged", async () => {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: BugEditCommentDocument, variables: matchVars },
      result: { data: { bugEditComment: { bug: { id: "1" } } } },
    };

    renderTimeline([makeAddComment({ message: "Same message" })], [mock]);
    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await new Promise((r) => setTimeout(r, 50));
    expect(matchVars).not.toHaveBeenCalled();
  });

  it("exits edit mode after successful save", async () => {
    const mock = {
      request: { query: BugEditCommentDocument, variables: vi.fn().mockReturnValue(true) },
      result: { data: { bugEditComment: { bug: { id: "1" } } } },
    };

    renderTimeline([makeAddComment({ message: "Original" })], [mock]);
    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "Updated" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(screen.queryByRole("textbox")).toBeNull());
  });
});

describe("Timeline — mutation variables", () => {
  const COMMENT_ID = "comment-1";
  const REPO = "myrepo";

  it("passes targetPrefix, trimmed message, and repoRef on Save click", async () => {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: BugEditCommentDocument, variables: matchVars },
      result: { data: { bugEditComment: { bug: { id: "1" } } } },
    };

    renderTimeline([makeAddComment({ id: COMMENT_ID, message: "Old message" })], [mock]);

    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "  New message  " } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { targetPrefix: COMMENT_ID, message: "New message", repoRef: REPO },
    });
    await waitFor(() => expect(screen.queryByRole("textbox")).toBeNull());
  });

  it("passes correct variables on Ctrl+Enter", async () => {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: BugEditCommentDocument, variables: matchVars },
      result: { data: { bugEditComment: { bug: { id: "1" } } } },
    };

    renderTimeline([makeAddComment({ id: COMMENT_ID, message: "Old" })], [mock]);

    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "New" } });
    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter", ctrlKey: true });

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { targetPrefix: COMMENT_ID, message: "New", repoRef: REPO },
    });
    await waitFor(() => expect(screen.queryByRole("textbox")).toBeNull());
  });

  it("works with null repoRef (default repo)", async () => {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: BugEditCommentDocument, variables: matchVars },
      result: { data: { bugEditComment: { bug: { id: "1" } } } },
    };

    renderTimeline([makeAddComment({ id: COMMENT_ID, message: "Old" })], [mock], null);

    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "New" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { targetPrefix: COMMENT_ID, message: "New", repoRef: null },
    });
    await waitFor(() => expect(screen.queryByRole("textbox")).toBeNull());
  });
});

describe("Timeline — changes by others", () => {
  function renderChanging(nodes: TimelineItemsFragment["nodes"]) {
    const view = renderTimeline(nodes);
    return {
      ...view,
      change: (next: TimelineItemsFragment["nodes"]) =>
        view.rerender(
          <MockedProvider mocks={[]} showWarnings={false}>
            <Suspense>
              <Timeline repo="myrepo" timeline={makeTimeline(next)} />
            </Suspense>
          </MockedProvider>,
        ),
    };
  }

  const highlighted = (container: HTMLElement) => container.querySelectorAll(".animate-highlight");

  it("announces nothing on the first render", () => {
    const { container } = renderTimeline([makeAddComment({ author: AUTHOR_OTHER })]);
    expect(addToast).not.toHaveBeenCalled();
    expect(highlighted(container)).toHaveLength(0);
  });

  it("announces and highlights a comment added by someone else", () => {
    const first = makeAddComment({ id: "c1" });
    const { container, change } = renderChanging([first]);
    change([first, makeAddComment({ id: "c2", author: AUTHOR_OTHER })]);

    expect(addToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: "Other User commented" }),
    );
    expect(highlighted(container)).toHaveLength(1);
    expect(container.querySelector("#timeline-c2 .animate-highlight")).not.toBeNull();
  });

  it("replays the highlight when the toast action is used", async () => {
    Element.prototype.scrollIntoView = vi.fn();
    render(<Toasts>{null}</Toasts>);
    const first = makeAddComment({ id: "c1" });
    const { container, change } = renderChanging([first]);
    change([first, makeAddComment({ id: "c2", author: AUTHOR_OTHER })]);
    const played = container.querySelector("#timeline-c2 .animate-highlight");

    fireEvent.click(await screen.findByRole("button", { name: "Show" }));

    const replayed = container.querySelector("#timeline-c2 .animate-highlight");
    expect(replayed).not.toBeNull();
    expect(replayed).not.toBe(played);
  });

  it("announces a comment edited by someone else, even within the same second", () => {
    const comment = { id: "c1", author: AUTHOR_OTHER };
    const { change } = renderChanging([makeAddComment(comment)]);
    // lastEdit unchanged: only the history tells the new version
    change([makeAddComment({ ...comment, message: "New", editors: [AUTHOR_OTHER, AUTHOR_OTHER] })]);

    expect(addToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: "Other User edited a comment" }),
    );
  });

  it("announces someone else's edit of the user's comment, naming the editor", () => {
    const comment = { id: "c1", author: AUTHOR_ME };
    const { change } = renderChanging([makeAddComment(comment)]);
    change([makeAddComment({ ...comment, message: "New", editors: [AUTHOR_ME, AUTHOR_OTHER] })]);

    expect(addToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: "Other User edited a comment" }),
    );
  });

  it("doesn't announce the user's edit of someone else's comment", () => {
    const comment = { id: "c1", author: AUTHOR_OTHER };
    const { change } = renderChanging([makeAddComment(comment)]);
    change([makeAddComment({ ...comment, message: "New", editors: [AUTHOR_OTHER, AUTHOR_ME] })]);

    expect(addToast).not.toHaveBeenCalled();
  });

  it("doesn't announce the user's own changes", () => {
    const first = makeAddComment({ id: "c1" });
    const { container, change } = renderChanging([first]);
    change([first, makeAddComment({ id: "c2", author: AUTHOR_ME })]);

    expect(addToast).not.toHaveBeenCalled();
    expect(highlighted(container)).toHaveLength(0);
  });
});

describe("Timeline — failed comment edit", () => {
  it("stays in edit mode with the text and shows the error", async () => {
    const mock = {
      request: { query: BugEditCommentDocument, variables: () => true },
      error: new Error("network down"),
    };
    renderTimeline([makeAddComment({ message: "Old" })], [mock]);

    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "New" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText("Failed to save the comment: network down")).toBeInTheDocument();
    expect(screen.getByRole("textbox")).toHaveValue("New");
  });
});
