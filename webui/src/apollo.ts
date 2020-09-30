import { ApolloClient, InMemoryCache } from '@apollo/client';

import introspectionResult from './fragmentTypes';

const client = new ApolloClient({
  uri: '/graphql',
  cache: new InMemoryCache({
    possibleTypes: introspectionResult.possibleTypes,
  }),
});

export default client;
