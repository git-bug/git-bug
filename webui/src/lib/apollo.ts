import {
  ApolloClient,
  ApolloLink,
  InMemoryCache,
  HttpLink,
  type InMemoryCacheConfig,
} from "@apollo/client";
import { GraphQLWsLink } from "@apollo/client/link/subscriptions";
import { createQueryPreloader } from "@apollo/client/react";
import { isSubscriptionOperation } from "@apollo/client/utilities";
import { createClient } from "graphql-ws";

import introspection from "@/__generated__/possible-types";

// The concrete types of each interface, without which the cache never matches
// a fragment on an interface (`... on Authored`).  Exported, like typePolicies,
// for the Storybook mock client.
export const possibleTypes = introspection.possibleTypes;

const httpLink = new HttpLink({
  uri: "/graphql",
  // include credentials so future httpOnly auth cookies are sent automatically
  credentials: "include",
});

// Subscriptions go over a single websocket, opened on the first subscription
// and retried when it drops.  Same origin as the page, so it carries the same
// cookies as the HTTP requests.
const wsLink = new GraphQLWsLink(
  createClient({
    url: () => {
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      return `${protocol}//${window.location.host}/graphql`;
    },
    retryAttempts: Infinity,
    on: {
      // the changes made while disconnected were never told: refetch what the
      // views display
      connected: (_socket, _payload, wasRetry) => {
        if (wasRetry) client.refetchQueries({ include: "active" }).catch(() => {});
      },
    },
  }),
);

const link = ApolloLink.split(({ query }) => isSubscriptionOperation(query), wsLink, httpLink);

// Repository has no `id`; `name` identifies it, empty for the default repo.
// Every query selecting `repository` must also select `name`, or Apollo throws
// `Missing field 'name' while extracting keyFields` and the query returns
// nothing.  apollo.test.ts enforces that, and covers why `[]` and `false` don't
// work here.  Exported so tests and the Storybook mock client reuse it.
export const typePolicies: InMemoryCacheConfig["typePolicies"] = {
  Repository: {
    keyFields: ["name"],
  },
};

export const client = new ApolloClient({
  link,
  // Data masking is off: fragment colocation is enforced at the type level
  // via codegen's $fragmentRefs branding (inlineFragmentTypes: "mask").
  // Components use codegen's useFragment (a zero-cost cast) to unmask.
  // When @defer is needed, individual components can switch to Apollo's
  // useSuspenseFragment — it works without dataMasking.
  dataMasking: false,

  cache: new InMemoryCache({ typePolicies, possibleTypes }),
});

// Preloader for use in TanStack Router loaders. Returns a QueryRef
// that components read with useReadQuery() for suspense-based rendering.
export const preloadQuery = createQueryPreloader(client);
