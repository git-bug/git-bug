import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";

import { GitRefType } from "@/__generated__/enums";
import type { FragmentType } from "@/__generated__/fragment-masking";
import { makeFragmentData } from "@/__generated__/fragment-masking";

import { RefSelector, REF_SELECTOR_REFS_FRAGMENT } from "./ref-selector";

function makeRefs(nodes: { name: string; shortName: string; type: GitRefType }[]) {
  return makeFragmentData(
    {
      __typename: "GitRefConnection" as const,
      nodes: nodes.map((n) => ({ __typename: "GitRef" as const, ...n })),
    },
    REF_SELECTOR_REFS_FRAGMENT,
  );
}

const DEFAULT_REFS = makeRefs([
  { name: "refs/heads/main", shortName: "main", type: GitRefType.Branch },
  { name: "refs/heads/develop", shortName: "develop", type: GitRefType.Branch },
  { name: "refs/tags/v1.0", shortName: "v1.0", type: GitRefType.Tag },
  { name: "refs/tags/v2.0", shortName: "v2.0", type: GitRefType.Tag },
]);

function renderSelector(
  props: {
    refs?: FragmentType<typeof REF_SELECTOR_REFS_FRAGMENT>;
    currentRef?: string;
    onSelect?: (shortName: string) => void;
  } = {},
) {
  const onSelect = props.onSelect ?? vi.fn<(shortName: string) => void>();
  render(
    <RefSelector
      refs={props.refs ?? DEFAULT_REFS}
      currentRef={props.currentRef ?? "main"}
      onSelect={onSelect}
    />,
  );
  return { onSelect };
}

function openSelector() {
  fireEvent.click(screen.getByRole("combobox"));
}

// ── Trigger button ────────────────────────────────────────────────────────────

describe("RefSelector — trigger", () => {
  it("shows the currentRef on the button", () => {
    renderSelector({ currentRef: "develop" });
    expect(screen.getByRole("combobox")).toHaveTextContent("develop");
  });
});

// ── Popover contents ──────────────────────────────────────────────────────────

describe("RefSelector — popover", () => {
  it("opens on click and shows a Branches group", () => {
    renderSelector();
    openSelector();
    expect(screen.getByText("Branches")).toBeInTheDocument();
  });

  it("shows a Tags group", () => {
    renderSelector();
    openSelector();
    expect(screen.getByText("Tags")).toBeInTheDocument();
  });

  it("lists branches as options", () => {
    renderSelector();
    openSelector();
    expect(screen.getByRole("option", { name: /main/ })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: /develop/ })).toBeInTheDocument();
  });

  it("lists tags as options", () => {
    renderSelector();
    openSelector();
    expect(screen.getByRole("option", { name: /v1\.0/ })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: /v2\.0/ })).toBeInTheDocument();
  });

  it("marks the current ref as selected", () => {
    renderSelector({ currentRef: "main" });
    openSelector();
    expect(screen.getByRole("option", { name: /main/ })).toHaveAttribute("aria-selected", "true");
  });

  it("hides the Branches group when all branches are filtered out", () => {
    renderSelector();
    openSelector();
    fireEvent.change(screen.getByPlaceholderText("Filter…"), { target: { value: "v1" } });
    expect(screen.queryByText("Branches")).toBeNull();
    expect(screen.getByText("Tags")).toBeInTheDocument();
  });

  it("hides the Tags group when all tags are filtered out", () => {
    renderSelector();
    openSelector();
    fireEvent.change(screen.getByPlaceholderText("Filter…"), { target: { value: "main" } });
    expect(screen.queryByText("Tags")).toBeNull();
    expect(screen.getByText("Branches")).toBeInTheDocument();
  });

  it("filters case-insensitively across both groups", () => {
    renderSelector();
    openSelector();
    fireEvent.change(screen.getByPlaceholderText("Filter…"), { target: { value: "MAIN" } });
    expect(screen.getByRole("option", { name: /main/ })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /develop/ })).toBeNull();
  });

  it("shows empty state when nothing matches the filter", () => {
    renderSelector();
    openSelector();
    fireEvent.change(screen.getByPlaceholderText("Filter…"), { target: { value: "zzznomatch" } });
    expect(screen.queryAllByRole("option")).toHaveLength(0);
  });
});

// ── Selection callback ────────────────────────────────────────────────────────

describe("RefSelector — onSelect", () => {
  it("calls onSelect with the branch shortName when clicked", () => {
    const { onSelect } = renderSelector();
    openSelector();
    fireEvent.click(screen.getByRole("option", { name: /develop/ }));
    expect(onSelect).toHaveBeenCalledWith("develop");
  });

  it("calls onSelect with the tag shortName when clicked", () => {
    const { onSelect } = renderSelector();
    openSelector();
    fireEvent.click(screen.getByRole("option", { name: /v1\.0/ }));
    expect(onSelect).toHaveBeenCalledWith("v1.0");
  });

  it("does not show the popover after selection (closes on click)", async () => {
    renderSelector();
    openSelector();
    fireEvent.click(screen.getByRole("option", { name: /develop/ }));
    expect(screen.queryByText("Branches")).toBeNull();
  });
});
