import { ApolloClient, InMemoryCache } from '@apollo/client';

import introspectionResult from './fragmentTypes';

const client = new ApolloClient({
  uri: '/graphql',
  cache: new InMemoryCache({
    possibleTypes: introspectionResult.possibleTypes,
    typePolicies: {
      // TODO: For now, we only query the default repository, so consider it as a singleton
      Repository: {
        keyFields: ['name'],
      },
    },
  }),
});

export default client;
