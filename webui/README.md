# webui

Web interface for git-bug. Built with Vite 8 + React 19 + TypeScript 7 + Tailwind v4 + shadcn/ui + TanStack Router + Apollo Client 4.

## Quickstart

Use the Go, Node.js 22 and pnpm versions pinned in the root [`.tool-versions`](../.tool-versions). In one terminal, build and start the backend from the repository root.

```bash
make build
# If this repository needs a git-bug identity, run ./git-bug user new first.
./git-bug webui --dev --no-open --port 3000
```

To browse without an identity, add `--read-only` to the server command. In a second terminal at the repository root, start Vite.

```bash
cd webui
pnpm dev
```

Open http://localhost:5173. Vite proxies `/graphql`, `/gitfile`, and `/upload` to the Go server on port 3000. The `--dev` flag lets GraphQL subscriptions connect from Vite's port. Vite picks up frontend edits while both processes are running.

## Routes

| Path                           | Page                                         |
| ------------------------------ | -------------------------------------------- |
| `/`                            | Repo picker — auto-redirects for single repo |
| `/$repo`                       | Redirect to the tree at the default ref      |
| `/$repo/tree/$ref/...path`     | Code browser — directory listing             |
| `/$repo/blob/$ref/...path`     | Code browser — file viewer                   |
| `/$repo/commits/$ref?path=...` | Commit history (optionally scoped to a path) |
| `/$repo/commit/$hash`          | Commit detail with collapsible file diffs    |
| `/$repo/issues`                | Issue list with search, filters, pagination  |
| `/$repo/issues/new`            | New issue form                               |
| `/$repo/issues/$id`            | Issue detail and timeline                    |
| `/$repo/user/$id`              | User profile with their issues               |

`_` is the URL segment for the default (unnamed) repository. Named repositories use their registered name.

## Code structure

```
src/
├── routes/             # File-based routing (TanStack Router)
│   ├── __root.tsx      # Root layout (Shell + error boundary)
│   ├── index.tsx       # Repo picker (/)
│   ├── $repo.tsx       # Repo layout — normalizes slug, preloads refs
│   └── $repo/
│       ├── index.tsx   # Redirect to tree/{defaultRef}
│       ├── _code.tsx   # Code browser layout (breadcrumb, ref selector)
│       ├── _code/      # tree/$ref/$, blob/$ref/$, commits/$ref
│       ├── _issues.tsx # Issues layout — preloads labels + identities
│       ├── _issues/    # issues/, issues/new, issues/$id, user/$id
│       └── commit/     # commit/$hash
├── components/
│   ├── ui/             # shadcn/ui primitives (button, input, avatar, ...)
│   ├── shared/         # Reusable app components with composition APIs
│   ├── bugs/           # Bug-feature components with data fetching
│   ├── code/           # Code browser components
│   ├── content/        # Markdown renderer
│   └── layout/         # Header, Shell and logo.tsx
├── __generated__/      # Typed GraphQL documents, types and fragment helpers
├── lib/                # Apollo client, auth, live updates, drafts, theming and utilities
├── vendor/shadcn/      # Vendored Tailwind CSS and license
├── routeTree.gen.ts    # Auto-generated route tree — do not edit
└── App.tsx             # Router instance + context
```

### Component layers

Components are organized in three layers:

- **`ui/`** — Generic primitives managed by shadcn CLI (`npx shadcn add`) or hand-written. No domain knowledge. Examples: button, input, avatar, badge, listbox (presentational compound components for dropdown menus), popover, separator, skeleton, textarea, toast (on Base UI's Toast; shadcn only offers one through sonner). Interactive dropdowns use `@floating-ui/react` hooks wired per-consumer with `Listbox.*` presentational primitives.

- **`shared/`** — App-level reusable components. These know about the domain (bug status, labels, identities) but contain no data fetching. They use **composition APIs** (compound components) and are typed against **colocated GraphQL fragments**. Examples: issue-row, label-badge, status-badge, status-tabs, comment-card, pagination, query-input, write-preview, empty-state, section-heading, issue-filters.

- **`bugs/`**, **`code/`** — Feature components with GraphQL mutations, `useAuth`, and other side effects. These compose `shared/` and `ui/` components.

### GraphQL fragments

Define each fragment with `graphql()` in the component file that reads it.

```tsx
// src/components/shared/label-badge.tsx
import { graphql } from "@/__generated__/gql";

export const LABEL_FIELDS_FRAGMENT = graphql(`
  fragment LabelFields on Label {
    name
    color {
      R
      G
      B
    }
  }
`);
```

The parent query spreads `...LabelFields` and passes the resulting label to the component. `LabelBadge` types its `label` prop as `FragmentType<typeof LABEL_FIELDS_FRAGMENT>` and reads the fields with `useFragment(LABEL_FIELDS_FRAGMENT, label)`.

```tsx
<LabelBadge label={label} />
```

After changing a GraphQL operation, fragment or schema, regenerate the typed documents and types from `webui/`.

```bash
pnpm codegen
```

## Routing

Routes use [TanStack Router](https://tanstack.com/router) with file-based routing and automatic code splitting. The `@tanstack/router-plugin` Vite plugin generates `routeTree.gen.ts` from the `src/routes/` directory.

Pathless layout routes (`_code.tsx`, `_issues.tsx`) group child routes that share data loading or layout without adding URL segments.

The router context provides:

- `preloadQuery` — Apollo `createQueryPreloader` for data loading in route loaders
- `ref` — normalized repo slug (null for default repo), set by `$repo.tsx` `beforeLoad`
- `refsRef` — preloaded Git refs shared by the repo's code pages
- `labelsRef`, `identitiesRef` — preloaded shared queries, set by `_issues.tsx` `beforeLoad`

Custom link components:

- `ButtonLink` — `createLink()`-wrapped anchor with button styling and preload-on-intent
- `BackLink` — uses `router.history.back()` when possible, falls back to a typed Link
- `LabelBadgeLink` — `createLink()`-wrapped label badge for filter navigation
- `StatusTabs.Tab` — `createLink()`-wrapped status toggle tab
- `Pagination.Previous/Next` — `createLink()`-wrapped pagination buttons

## Data loading

Data is loaded in route loaders using Apollo's `preloadQuery` + `useReadQuery` pattern:

```ts
export const Route = createFileRoute("/$repo/_issues/issues/$id")({
  loader: async ({ context: { preloadQuery, ref }, params: { id } }) => {
    const bugDetailRef = preloadQuery(BUG_DETAIL_QUERY, {
      fetchPolicy: "network-only",
      variables: { ref, prefix: id },
    });
    return { bugDetailRef: await preloadQuery.toPromise(bugDetailRef) };
  },
});
```

The router waits for `toPromise()` before transitioning, then the component reads data with `useReadQuery()`. Cascading queries (e.g. last commits after tree loads) stay as component-level `useQuery`.

Search params that affect data loading use `loaderDeps` so the loader re-runs when they change (e.g. issue filters, pagination cursors).

Issue pages subscribe to repository changes over the `/graphql` WebSocket. The detail page refetches when the issue changes. The issue list shows a refresh banner so the reader can choose when to update the results.

## Storybook

[Storybook 10](https://storybook.js.org/) is set up for component development and testing:

```bash
pnpm storybook        # Dev server on http://localhost:6006
pnpm build-storybook  # Production build
```

Component stories live alongside the components in `*.stories.tsx` files. They use CSF3 with `satisfies Meta<typeof Component>` for type inference.

## Testing

Vitest 5 runs two test projects.

| Project       | Environment           | What it does                                                                    |
| ------------- | --------------------- | ------------------------------------------------------------------------------- |
| **storybook** | Chromium (Playwright) | Story smoke tests, accessibility checks (axe-core) and `play` interaction tests |
| **snapshot**  | happy-dom             | Unit tests and DOM snapshots from portable stories in `src/**/*.test.{ts,tsx}`  |

Before the first browser run, install Chromium with `pnpm exec playwright install chromium` from `webui/`. The browser project skips `file-viewer.stories.tsx` because Shiki's WebAssembly engine fails in Vitest browser mode.

```bash
pnpm test run                       # Run all tests once
pnpm test run --project=storybook   # Run browser tests
pnpm test run --project=snapshot    # Run unit and snapshot tests
pnpm test run --project=snapshot -u # Update DOM snapshots
```

### Adding tests

Storybook runs smoke and accessibility checks for included stories. To add a DOM snapshot, create a `*.test.tsx` file next to the story.

```tsx
import { composeStories } from "@storybook/react-vite";
import { expect, test } from "vitest";

import * as stories from "./my-component.stories";

const composed = composeStories(stories);

for (const [name, Story] of Object.entries(composed)) {
  test(`MyComponent/${name} matches snapshot`, async () => {
    await Story.run();
    expect(document.body.firstChild).toMatchSnapshot();
  });
}
```

For interaction tests, add `play` functions to stories:

```tsx
export const MyInteraction: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(canvas.getByRole("button"));
    await expect(canvas.getByText("Result")).toBeVisible();
  },
};
```

## Tooling

| Tool                                                                           | Purpose                                                 |
| ------------------------------------------------------------------------------ | ------------------------------------------------------- |
| [oxlint](https://oxc.rs)                                                       | Linter with type-aware rules + storybook/router plugins |
| [oxfmt](https://oxc.rs)                                                        | Formatter with import + Tailwind class sorting          |
| [Storybook 10](https://storybook.js.org)                                       | Component development + visual testing                  |
| [Vitest 5](https://vitest.dev)                                                 | Test runner (browser + happy-dom)                       |
| [@storybook/addon-a11y](https://storybook.js.org/addons/@storybook/addon-a11y) | Accessibility testing via axe-core                      |
| [valibot](https://valibot.dev)                                                 | Runtime validation for route search parameters          |
| [@tsconfig/bases](https://github.com/tsconfig/bases)                           | Shared tsconfig presets (vite-react + strictest)        |

```bash
pnpm lint        # oxlint (type-aware, 0 warnings target)
pnpm lint:fix    # oxlint with auto-fix
pnpm fmt         # oxfmt format
pnpm fmt:check   # oxfmt check only
pnpm check       # lint, check formatting and verify vendored shadcn files
pnpm vendor:sync # refresh vendored CSS and license after a shadcn upgrade
pnpm test        # vitest (all projects)
pnpm storybook   # storybook dev server
pnpm codegen     # regenerate GraphQL types
```

## Auth

In writable mode, the server uses the repository's configured git-bug identity for all requests. In `--read-only` mode, requests are anonymous and `useAuth()` returns `{ user: null }`. The root route preloads the `UserIdentity` query, which `useAuth()` reads with `useSuspenseQuery`.

## Build for production

From the repository root, run the same build used in the quickstart.

```bash
make build
```

`make build` installs frontend dependencies, builds `webui/dist/`, and compiles `./git-bug` with `go build -tags webui`. The build tag enables `//go:embed all:dist` in [assets.go](assets.go).

## Theming

`ThemeProvider` saves the selected theme in local storage. On first use it follows the system preference, choosing Light or Dark soft. The header also offers Light soft and Dark. In `src/index.css`, `:root` defines Light and the `.light-soft`, `.dark-soft` and `.dark` classes on `<html>` override its variables. Tailwind's `@theme inline` maps those variables to utilities.
