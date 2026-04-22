import { ApolloClient, HttpLink, InMemoryCache } from '@apollo/client';
import { setContext } from '@apollo/client/link/context';

import introspectionResult from './fragmentTypes';

// currentRepoName is updated by the router when a user navigates into
// /r/:name/*. Apollo's auth-link (setContext) reads it on each request and
// sets X-Repo-Name, which the git-bug webui middleware uses to scope the
// request to the right repository. When unset (root path, single-repo
// mode), the server falls back to the sole registered repo.
let currentRepoName: string | null = null;

export function setCurrentRepoName(name: string | null) {
  if (currentRepoName === name) return;
  currentRepoName = name;
  // Whenever the effective repo changes, drop the Apollo cache so queries
  // don't hand back results from the previous repo. This is a hammer, but
  // the alternative is threading repoName as a variable through every
  // operation (~12 .graphql files). Since the X-Repo-Name header isn't part
  // of the cache key, a per-repo clear is the simplest correctness fix.
  if (client) {
    // resetStore refetches active queries too; clearStore just drops data.
    client.resetStore().catch(() => {});
  }
}

const httpLink = new HttpLink({ uri: '/graphql' });

const authLink = setContext((_, { headers }) => {
  if (!currentRepoName) return { headers };
  return {
    headers: {
      ...(headers || {}),
      'X-Repo-Name': currentRepoName,
    },
  };
});

const client = new ApolloClient({
  link: authLink.concat(httpLink),
  cache: new InMemoryCache({
    possibleTypes: introspectionResult.possibleTypes,
    typePolicies: {
      Repository: {
        keyFields: ['name'],
      },
    },
  }),
});

export default client;
