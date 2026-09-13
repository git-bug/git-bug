import { InMemoryCache } from "@apollo/client/cache";
import { ApolloClient, ApolloLink, Observable } from "@apollo/client/core";
import { ApolloProvider } from "@apollo/client/react";
import type { TypedDocumentNode } from "@graphql-typed-document-node/core";
import type { Decorator } from "@storybook/react-vite";
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router";
import { Suspense } from "react";

import { typePolicies } from "../src/lib/apollo";

// Catch-all route so any <Link to="..."> resolves without errors.
const rootRoute = createRootRoute();
const catchAll = createRoute({
  getParentRoute: () => rootRoute,
  path: "$",
});
rootRoute.addChildren([catchAll]);

const router = createRouter({
  routeTree: rootRoute,
  history: createMemoryHistory({ initialEntries: ["/"] }),
});

// Router with a /$repo route, starting at /_
// Used for components that call useParams() and need params.repo to be set.
const repoRoot = createRootRoute();
const repoRoute = createRoute({ getParentRoute: () => repoRoot, path: "/$repo" });
const repoCatchAll = createRoute({ getParentRoute: () => repoRoot, path: "$" });
repoRoot.addChildren([repoRoute, repoCatchAll]);
const repoRouter = createRouter({
  routeTree: repoRoot,
  history: createMemoryHistory({ initialEntries: ["/_"] }),
});

// Wraps a story in a TanStack Router context so components using <Link> render.
export const withRouter: Decorator = (Story) => (
  <RouterProvider router={router} defaultComponent={() => <Story />} />
);

// Like withRouter but starts at /_  so useParams() returns { repo: "_" }.
export const withRepoRouter: Decorator = (Story) => (
  <RouterProvider router={repoRouter} defaultComponent={() => <Story />} />
);

// Mock Apollo client for stories.
// - useSuspenseFragment reads from this cache
// - useSuspenseQuery for useAuth() hits the mock link
const mockApolloClient = new ApolloClient({
  link: new ApolloLink(
    (operation) =>
      new Observable((observer) => {
        const data: Record<string, unknown> = {};
        // Provide mock data for the UserIdentity query used by useAuth()
        if (operation.operationName === "UserIdentity") {
          data.repository = {
            __typename: "Repository",
            // Required by the cache key policy; null is the default repo.
            name: null,
            userIdentity: {
              __typename: "Identity",
              id: "mock-user",
              humanId: "mock1",
              name: "Mock User",
              displayName: "Mock User",
              avatarUrl: null,
              email: null,
              login: null,
            },
          };
        }
        observer.next({ data });
        observer.complete();
      }),
  ),
  cache: new InMemoryCache({
    typePolicies: {
      // The app's real policies, so mocks that break them fail here too.
      ...typePolicies,
      // Types without `id` need explicit keyFields so useSuspenseFragment
      // can normalize and cache the mock data passed via `from`.
      Label: { keyFields: ["name"] },
      GitBlob: { keyFields: ["hash"] },
      GitRefConnection: { keyFields: [], fields: { nodes: { merge: false } } },
      BugTimelineItemConnection: { keyFields: [], fields: { nodes: { merge: false } } },
    },
  }),
  dataMasking: false,
});

// Wraps a story in an ApolloProvider. Components using useSuspenseFragment
// need their data pre-written to the cache via withCachedFragments().
export const withApollo: Decorator = (Story) => (
  <ApolloProvider client={mockApolloClient}>
    <Suspense fallback={<div style={{ padding: 16, color: "orange" }}>Loading…</div>}>
      <Story />
    </Suspense>
  </ApolloProvider>
);

// Pre-writes fragment data into the Apollo cache so that useSuspenseFragment
// can read it without suspending. Call this in the story's decorators list
// AFTER withApollo.
//
// Usage:
//   decorators: [withApollo, withCachedFragments(
//     [MY_FRAGMENT, "MyFragment", myMockData],
//     [MY_FRAGMENT, "MyFragment", anotherMockData],
//   )]
export function withCachedFragments(
  ...entries: Array<
    readonly [fragment: TypedDocumentNode, fragmentName: string, data: Record<string, unknown>]
  >
): Decorator {
  return (Story) => {
    for (const [fragment, fragmentName, data] of entries) {
      mockApolloClient.cache.writeFragment({ fragment, fragmentName, data });
    }
    return <Story />;
  };
}
