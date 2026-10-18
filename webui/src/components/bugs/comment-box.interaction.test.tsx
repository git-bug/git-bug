import { MockedProvider } from "@apollo/client/testing/react";
import type { TypedDocumentNode } from "@graphql-typed-document-node/core";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { Suspense } from "react";
import { describe, it, expect, vi } from "vitest";

import { Status } from "@/__generated__/enums";
import {
  BugAddCommentDocument,
  BugAddCommentAndCloseDocument,
  BugAddCommentAndReopenDocument,
  BugStatusCloseDocument,
  BugStatusOpenDocument,
} from "@/__generated__/graphql";
import { useAuth } from "@/lib/auth";

import { CommentBox } from "./comment-box";

type MockUser = {
  id: string;
  humanId: string;
  displayName: string;
  avatarUrl: string | null;
} | null;
vi.mock("@/lib/auth", () => ({
  useAuth: vi.fn<() => { user: MockUser }>(() => ({
    user: {
      id: "user-1",
      humanId: "u1",
      displayName: "Test User",
      avatarUrl: null,
    },
  })),
}));

function renderCommentBox(props: { bugPrefix?: string; bugStatus?: Status; repo?: string | null }) {
  const { bugPrefix = "bug1", bugStatus = Status.Open, repo = "myrepo" } = props;
  return render(
    <MockedProvider showWarnings={false}>
      <Suspense>
        <CommentBox bugPrefix={bugPrefix} bugStatus={bugStatus} repo={repo} />
      </Suspense>
    </MockedProvider>,
  );
}

// Helper: fill in the comment textarea.
function typeComment(text: string) {
  fireEvent.change(screen.getByPlaceholderText("Leave a comment…"), {
    target: { value: text },
  });
}

describe("CommentBox — auth gate", () => {
  it("renders nothing when there is no logged-in user", async () => {
    vi.mocked(useAuth).mockReturnValueOnce({ user: null });

    const { container } = renderCommentBox({});
    expect(container.firstChild).toBeNull();
  });
});

describe("CommentBox — button labels", () => {
  it('shows "Close issue" when the bug is open', () => {
    renderCommentBox({ bugStatus: Status.Open });
    expect(screen.getByRole("button", { name: "Close issue" })).toBeInTheDocument();
  });

  it('shows "Reopen issue" when the bug is closed', () => {
    renderCommentBox({ bugStatus: Status.Closed });
    expect(screen.getByRole("button", { name: "Reopen issue" })).toBeInTheDocument();
  });

  it('"Comment" button is disabled when the textarea is empty', () => {
    renderCommentBox({});
    expect(screen.getByRole("button", { name: "Comment" })).toBeDisabled();
  });

  it('"Comment" button becomes enabled once the user types', () => {
    renderCommentBox({});
    typeComment("hello");
    expect(screen.getByRole("button", { name: "Comment" })).toBeEnabled();
  });
});

describe("CommentBox — mutation variables", () => {
  const PREFIX = "bug1";
  const REPO = "myrepo";
  const MESSAGE = "my comment";

  function mockMutation(
    document: TypedDocumentNode<any, any>,
    resultData: Record<string, unknown>,
  ) {
    const matchVars = vi.fn().mockReturnValue(true);
    const mock = {
      request: { query: document, variables: matchVars },
      result: { data: resultData },
    };
    return { matchVars, mock };
  }

  it("addComment — passes prefix, message, and repoRef", async () => {
    const { matchVars, mock } = mockMutation(BugAddCommentDocument, {
      bugAddComment: { bug: { id: "1" } },
    });

    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <CommentBox bugPrefix={PREFIX} bugStatus={Status.Open} repo={REPO} />
        </Suspense>
      </MockedProvider>,
    );

    typeComment(MESSAGE);
    fireEvent.click(screen.getByRole("button", { name: "Comment" }));

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, message: MESSAGE, repoRef: REPO },
    });
    await waitFor(() => expect(screen.getByPlaceholderText("Leave a comment…")).toHaveValue(""));
  });

  it("addComment — works with null repoRef (default repo)", async () => {
    const { matchVars, mock } = mockMutation(BugAddCommentDocument, {
      bugAddComment: { bug: { id: "1" } },
    });

    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <CommentBox bugPrefix={PREFIX} bugStatus={Status.Open} repo={null} />
        </Suspense>
      </MockedProvider>,
    );

    typeComment(MESSAGE);
    fireEvent.click(screen.getByRole("button", { name: "Comment" }));

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, message: MESSAGE, repoRef: null },
    });
    await waitFor(() => expect(screen.getByPlaceholderText("Leave a comment…")).toHaveValue(""));
  });

  it("addComment — trims whitespace from message", async () => {
    const { matchVars, mock } = mockMutation(BugAddCommentDocument, {
      bugAddComment: { bug: { id: "1" } },
    });

    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <CommentBox bugPrefix={PREFIX} bugStatus={Status.Open} repo={REPO} />
        </Suspense>
      </MockedProvider>,
    );

    typeComment("  my comment  ");
    fireEvent.click(screen.getByRole("button", { name: "Comment" }));

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, message: "my comment", repoRef: REPO },
    });
    await waitFor(() => expect(screen.getByPlaceholderText("Leave a comment…")).toHaveValue(""));
  });

  it("statusClose (open, no message) — passes prefix and repoRef", async () => {
    const { matchVars, mock } = mockMutation(BugStatusCloseDocument, {
      bugStatusClose: { bug: { id: "1", status: Status.Closed } },
    });

    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <CommentBox bugPrefix={PREFIX} bugStatus={Status.Open} repo={REPO} />
        </Suspense>
      </MockedProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Close issue" }));

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, repoRef: REPO },
    });
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Close issue" })).not.toBeDisabled(),
    );
  });

  it("addAndClose (open, with message) — passes prefix, message, and repoRef", async () => {
    const { matchVars, mock } = mockMutation(BugAddCommentAndCloseDocument, {
      bugAddCommentAndClose: { bug: { id: "1" } },
    });

    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <CommentBox bugPrefix={PREFIX} bugStatus={Status.Open} repo={REPO} />
        </Suspense>
      </MockedProvider>,
    );

    typeComment(MESSAGE);
    fireEvent.click(screen.getByRole("button", { name: "Close issue" }));

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, message: MESSAGE, repoRef: REPO },
    });
    await waitFor(() => expect(screen.getByPlaceholderText("Leave a comment…")).toHaveValue(""));
  });

  it("statusOpen (closed, no message) — passes prefix and repoRef", async () => {
    const { matchVars, mock } = mockMutation(BugStatusOpenDocument, {
      bugStatusOpen: { bug: { id: "1", status: Status.Open } },
    });

    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <CommentBox bugPrefix={PREFIX} bugStatus={Status.Closed} repo={REPO} />
        </Suspense>
      </MockedProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Reopen issue" }));

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, repoRef: REPO },
    });
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Reopen issue" })).not.toBeDisabled(),
    );
  });

  it("addAndReopen (closed, with message) — passes prefix, message, and repoRef", async () => {
    const { matchVars, mock } = mockMutation(BugAddCommentAndReopenDocument, {
      bugAddCommentAndReopen: { bug: { id: "1" } },
    });

    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <CommentBox bugPrefix={PREFIX} bugStatus={Status.Closed} repo={REPO} />
        </Suspense>
      </MockedProvider>,
    );

    typeComment(MESSAGE);
    fireEvent.click(screen.getByRole("button", { name: "Reopen issue" }));

    await waitFor(() => expect(matchVars).toHaveBeenCalled());
    expect(matchVars).toHaveBeenCalledWith({
      input: { prefix: PREFIX, message: MESSAGE, repoRef: REPO },
    });
    await waitFor(() => expect(screen.getByPlaceholderText("Leave a comment…")).toHaveValue(""));
  });
});

describe("CommentBox — state transitions", () => {
  it("clears the message after a successful addComment", async () => {
    const mock = {
      request: {
        query: BugAddCommentDocument,
        variables: vi.fn().mockReturnValue(true),
      },
      result: { data: { bugAddComment: { bug: { id: "1" } } } },
    };

    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <CommentBox bugPrefix="bug1" bugStatus={Status.Open} repo="myrepo" />
        </Suspense>
      </MockedProvider>,
    );

    const textarea = screen.getByPlaceholderText("Leave a comment…");
    typeComment("my comment");
    expect(textarea).toHaveValue("my comment");

    fireEvent.click(screen.getByRole("button", { name: "Comment" }));

    await waitFor(() => expect(textarea).toHaveValue(""));
  });
});

describe("CommentBox — failures", () => {
  it("keeps the message and shows the error when the comment fails", async () => {
    const mock = {
      request: { query: BugAddCommentDocument, variables: () => true },
      error: new Error("network down"),
    };
    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <CommentBox bugPrefix="bug1" bugStatus={Status.Open} repo="myrepo" />
        </Suspense>
      </MockedProvider>,
    );

    typeComment("my comment");
    fireEvent.click(screen.getByRole("button", { name: "Comment" }));

    expect(await screen.findByText("Failed to save: network down")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Leave a comment…")).toHaveValue("my comment");
  });
});

describe("CommentBox — drafts", () => {
  it("keeps an unsent comment across a remount, and forgets it once posted", async () => {
    const mock = {
      request: { query: BugAddCommentDocument, variables: () => true },
      result: { data: { bugAddComment: { bug: { id: "1" } } } },
    };
    const first = renderCommentBox({});
    typeComment("half a comment");
    first.unmount();

    render(
      <MockedProvider mocks={[mock]} showWarnings={false}>
        <Suspense>
          <CommentBox bugPrefix="bug1" bugStatus={Status.Open} repo="myrepo" />
        </Suspense>
      </MockedProvider>,
    );
    expect(screen.getByPlaceholderText("Leave a comment…")).toHaveValue("half a comment");

    fireEvent.click(screen.getByRole("button", { name: "Comment" }));
    await waitFor(() => expect(screen.getByPlaceholderText("Leave a comment…")).toHaveValue(""));
    expect(localStorage.length).toBe(0);
  });
});
