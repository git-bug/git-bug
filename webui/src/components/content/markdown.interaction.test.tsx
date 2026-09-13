import { render, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Markdown } from "./markdown";

// A scrollable <pre> has to be reachable by keyboard (axe rule
// scrollable-region-focusable).  Shiki adds tabindex="0" to the blocks it
// highlights; everything else gets it from the pre override in Markdown.
describe("Markdown — code block focusability", () => {
  it("gives a code block without a language a tabindex", async () => {
    const { container } = render(<Markdown content={"```\nplain text\n```"} />);
    await waitFor(() => expect(container.querySelector("pre")).toBeInTheDocument());
    expect(container.querySelector("pre")).toHaveAttribute("tabindex", "0");
  });

  it("overrides a tabindex=-1 carried by a raw <pre> in the document", async () => {
    const { container } = render(
      <Markdown content={'<pre tabindex="-1">unreachable by keyboard</pre>'} />,
    );
    await waitFor(() => expect(container.querySelector("pre")).toBeInTheDocument());
    expect(container.querySelector("pre")).toHaveAttribute("tabindex", "0");
  });
});
