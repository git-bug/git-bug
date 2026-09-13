import { ApolloClient, InMemoryCache, HttpLink, type InMemoryCacheConfig } from "@apollo/client";
import { createQueryPreloader } from "@apollo/client/react";

const httpLink = new HttpLink({
  uri: "/graphql",
  // include credentials so future httpOnly auth cookies are sent automatically
  credentials: "include",
});

// Repository has no `id`; `name` identifies it, null being the default repo.
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
  link: httpLink,
  // Data masking is off: fragment colocation is enforced at the type level
  // via codegen's $fragmentRefs branding (inlineFragmentTypes: "mask").
  // Components use codegen's useFragment (a zero-cost cast) to unmask.
  // When @defer is needed, individual components can switch to Apollo's
  // useSuspenseFragment — it works without dataMasking.
  dataMasking: false,

  cache: new InMemoryCache({ typePolicies }),
});

// Preloader for use in TanStack Router loaders. Returns a QueryRef
// that components read with useReadQuery() for suspense-based rendering.
export const preloadQuery = createQueryPreloader(client);
