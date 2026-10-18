import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { useDraft } from "./draft";

const DAY_MS = 24 * 60 * 60 * 1000;

describe("useDraft", () => {
  it("restores the text typed before a reload", () => {
    const first = renderHook(() => useDraft("repo:bug1:comment"));
    act(() => first.result.current[1]("half a comment"));
    first.unmount();

    const second = renderHook(() => useDraft("repo:bug1:comment"));
    expect(second.result.current[0]).toBe("half a comment");
  });

  it("forgets the draft once emptied, like after sending it", () => {
    const { result } = renderHook(() => useDraft("repo:bug1:comment"));
    act(() => result.current[1]("sent soon"));
    act(() => result.current[1](""));

    expect(localStorage.length).toBe(0);
  });

  it("keeps the newer draft of another tab when emptied", () => {
    const { result } = renderHook(() => useDraft("repo:new-issue:title"));
    act(() => result.current[1]("from this tab"));
    localStorage.setItem(
      "git-bug:draft:repo:new-issue:title",
      JSON.stringify({ value: "from another tab", savedAt: Date.now() }),
    );
    act(() => result.current[1](""));

    const reloaded = renderHook(() => useDraft("repo:new-issue:title"));
    expect(reloaded.result.current[0]).toBe("from another tab");
  });

  it("switches to the draft of the new key", () => {
    localStorage.setItem(
      "git-bug:draft:repo:bug2:comment",
      JSON.stringify({ value: "on bug 2", savedAt: Date.now() }),
    );
    const { result, rerender } = renderHook(({ key }) => useDraft(key), {
      initialProps: { key: "repo:bug1:comment" },
    });
    act(() => result.current[1]("on bug 1"));

    rerender({ key: "repo:bug2:comment" });
    expect(result.current[0]).toBe("on bug 2");
    rerender({ key: "repo:bug1:comment" });
    expect(result.current[0]).toBe("on bug 1");
  });

  it("ignores and prunes the drafts left alone too long", async () => {
    localStorage.setItem(
      "git-bug:draft:repo:old:comment",
      JSON.stringify({ value: "forgotten", savedAt: Date.now() - 31 * DAY_MS }),
    );
    localStorage.setItem(
      "git-bug:draft:repo:recent:comment",
      JSON.stringify({ value: "recent", savedAt: Date.now() - DAY_MS }),
    );
    localStorage.setItem("theme", "dark");

    // a fresh module, as pruning happens once per page load
    vi.resetModules();
    const fresh = await import("./draft");
    const { result } = renderHook(() => fresh.useDraft("repo:old:comment"));

    expect(result.current[0]).toBe("");
    expect(localStorage.getItem("git-bug:draft:repo:old:comment")).toBeNull();
    expect(localStorage.getItem("git-bug:draft:repo:recent:comment")).not.toBeNull();
    expect(localStorage.getItem("theme")).toBe("dark");
  });

  it("still works as plain state without storage", () => {
    const setItem = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("unavailable");
    });
    onTestFinished(() => setItem.mockRestore());

    const { result } = renderHook(() => useDraft("repo:bug1:comment"));
    act(() => result.current[1]("not kept"));

    expect(result.current[0]).toBe("not kept");
  });
});
