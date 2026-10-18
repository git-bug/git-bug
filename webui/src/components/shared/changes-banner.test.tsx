import { composeStories } from "@storybook/react-vite";
import { expect, test } from "vitest";

import * as stories from "./changes-banner.stories";

const composed = composeStories(stories);

for (const [name, Story] of Object.entries(composed)) {
  test(`ChangesBanner/${name} matches snapshot`, async () => {
    await Story.run();
    expect(document.body.firstChild).toMatchSnapshot();
  });
}
