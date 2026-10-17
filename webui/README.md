# webui

Web interface for git-bug. Built with Vite 8 + React 19 + TypeScript 6 + Tailwind v4 + shadcn/ui + TanStack Router + Apollo Client 4.

## Quickstart

You need two processes running:

```bash
# 1. Go backend (from repo root)
go run . webui --no-open --port 3000

# 2. Vite dev server (from this directory)
pnpm install
pnpm dev
```

Open http://localhost:5173. Vite proxies `/graphql`, `/gitfile`, and `/upload` to the Go server on port 3000.

Node 22 is required. If you use asdf, `.tool-versions` pins the right version automatically.

## Routes

| Path                           | Page                                         |
| ------------------------------ | -------------------------------------------- |
| `/`                            | Repo picker — auto-redirects for single repo |
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
│   ├── $repo/
│   │   ├── index.tsx   # Redirect to tree/{defaultRef}
│   │   ├── _code.tsx   # Code browser layout (breadcrumb, ref selector)
│   │   ├── _code/      # tree/$ref/$, blob/$ref/$, commits/$ref
│   │   ├── _issues.tsx # Issues layout — preloads labels + identities
│   │   ├── _issues/    # issues/, issues/new, issues/$id, user/$id
│   │   └── commit/     # commit/$hash
├── components/
│   ├── ui/             # shadcn/ui primitives (button, input, avatar, ...)
│   ├── shared/         # Reusable app components with composition APIs
│   ├── bugs/           # Bug-feature components with data fetching
│   ├── code/           # Code browser components
│   ├── content/        # Markdown renderer
│   └── layout/         # Header + Shell
├── __generated__/      # Generated typed hooks — do not edit
├── assets/             # Logo SVG
├── lib/                # apollo.ts, auth.tsx, theme.tsx, utils.ts, query-utils.ts, shiki.ts
├── routeTree.gen.ts    # Auto-generated route tree — do not edit
└── App.tsx             # Router instance + context
```

### Component layers

Components are organized in three layers:

- **`ui/`** — Generic primitives managed by shadcn CLI (`npx shadcn add`) or hand-written. No domain knowledge. Examples: button, input, avatar, badge, listbox (presentational compound components for dropdown menus), popover, separator, skeleton, textarea, toast (on Base UI's Toast; shadcn only offers one through sonner). Interactive dropdowns use `@floating-ui/react` hooks wired per-consumer with `Listbox.*` presentational primitives.

- **`shared/`** — App-level reusable components. These know about the domain (bug status, labels, identities) but contain no data fetching. They use **composition APIs** (compound components) and are typed against **colocated GraphQL fragments**. Examples: issue-row, label-badge, status-badge, status-tabs, comment-card, pagination, query-input, write-preview, empty-state, section-heading, issue-filters.

- **`bugs/`**, **`code/`** — Feature components with GraphQL mutations, `useAuth`, and other side effects. These compose `shared/` and `ui/` components.

### GraphQL fragments

Fragments are defined inline using the `graphql()` tagged template in the component file that consumes them:

```tsx
// src/components/shared/label-badge.tsx
import { graphql } from "@/__generated__/gql";

graphql(`
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

Components are typed against their fragments:

```tsx
import type { LabelFieldsFragment } from "@/__generated__/graphql";
import { LabelBadge } from "@/components/shared/label-badge";

// Spread fragment data directly onto the component
<LabelBadge {...label} />;
```

Codegen scans all `src/**/*.{ts,tsx}` files for `graphql()` calls. After changing any fragment or query, regenerate typed hooks:

```bash
pnpm codegen
```

## Routing

Routes use [TanStack Router](https://tanstack.com/router) with file-based routing and automatic code splitting. The `@tanstack/router-plugin` Vite plugin generates `routeTree.gen.ts` from the `src/routes/` directory.

Pathless layout routes (`_code.tsx`, `_issues.tsx`) group child routes that share data loading or layout without adding URL segments.

The router context provides:

- `preloadQuery` — Apollo `createQueryPreloader` for data loading in route loaders
- `ref` — normalized repo slug (null for default repo), set by `$repo.tsx` `beforeLoad`
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
export const Route = createFileRoute("/$repo/issues/$id")({
  loader: async ({ context: { preloadQuery, ref }, params: { id } }) => {
    const bugDetailRef = preloadQuery<BugDetailQuery>(BugDetailDocument, {
      variables: { ref, prefix: id },
    });
    return { bugDetailRef: await preloadQuery.toPromise(bugDetailRef) };
  },
});
```

The router waits for `toPromise()` before transitioning, then the component reads data with `useReadQuery()`. Cascading queries (e.g. last commits after tree loads) stay as component-level `useQuery`.

Search params that affect data loading use `loaderDeps` so the loader re-runs when they change (e.g. issue filters, pagination cursors).

## Storybook

[Storybook 10](https://storybook.js.org/) is set up for component development and testing:

```bash
pnpm storybook        # Dev server on http://localhost:6006
pnpm build-storybook  # Production build
```

Every presentational component has stories. Stories use the CSF3 format with `satisfies Meta<typeof Component>` for full type inference. Mock data is typed against GraphQL fragment types.

## Testing

Tests run via [Vitest 4](https://vitest.dev/) with two projects:

| Project       | Environment           | What it does                                                                       |
| ------------- | --------------------- | ---------------------------------------------------------------------------------- |
| **storybook** | Chromium (Playwright) | Smoke tests every story + a11y checks (axe-core) + play function interaction tests |
| **snapshot**  | happy-dom             | DOM snapshot tests via portable stories API                                        |

```bash
pnpm test                          # Run all tests
pnpm test -- --project=storybook   # Storybook tests only
pnpm test -- --project=snapshot    # Snapshot tests only
pnpm test -- -u                    # Update snapshots
```

### Adding tests

Every story automatically becomes a smoke test and an a11y test. For snapshot tests, add a `*.test.tsx` file next to the story:

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

| Tool                                                                           | Purpose                                                  |
| ------------------------------------------------------------------------------ | -------------------------------------------------------- |
| [oxlint](https://oxc.rs)                                                       | Linter with type-aware rules + storybook/router plugins  |
| [oxfmt](https://oxc.rs)                                                        | Formatter with import + Tailwind class sorting           |
| [Storybook 10](https://storybook.js.org)                                       | Component development + visual testing                   |
| [Vitest 4](https://vitest.dev)                                                 | Test runner (browser + happy-dom)                        |
| [@storybook/addon-a11y](https://storybook.js.org/addons/@storybook/addon-a11y) | Accessibility testing via axe-core                       |
| [valibot](https://valibot.dev)                                                 | Runtime validation for search params and fetch responses |
| [@tsconfig/bases](https://github.com/tsconfig/bases)                           | Shared tsconfig presets (vite-react + strictest)         |

```bash
pnpm lint        # oxlint (type-aware, 0 warnings target)
pnpm lint:fix    # oxlint with auto-fix
pnpm fmt         # oxfmt format
pnpm fmt:check   # oxfmt check only
pnpm check       # lint + format check
pnpm test        # vitest (all projects)
pnpm storybook   # storybook dev server
pnpm codegen     # regenerate GraphQL types
```

## Auth

Currently local-only: the server injects the git config identity for every request. `useAuth()` (`src/lib/auth.tsx`) fetches the user identity via a GraphQL `useSuspenseQuery`, preloaded in the root route loader so it's always resolved before components render.

## Build for production

The Go binary embeds the compiled frontend via `//go:embed all:dist` in `webui/handler.go`:

```bash
pnpm build           # outputs to webui/dist/
cd .. && go build .  # embeds dist/ into the binary
```

## Theming

`ThemeProvider` (`src/lib/theme.tsx`) toggles the `dark` class on `<html>`. CSS variables for both modes are defined in `src/index.css` using Tailwind v4's `@theme inline` block. Components pick them up automatically.
