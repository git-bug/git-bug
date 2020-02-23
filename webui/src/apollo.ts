import ApolloClient from 'apollo-boost';
import {
  IntrospectionFragmentMatcher,
  InMemoryCache,
} from 'apollo-cache-inmemory';

import introspectionQueryResultData from './fragmentTypes';

const client = new ApolloClient({
  uri: '/graphql',
  cache: new InMemoryCache({
    fragmentMatcher: new IntrospectionFragmentMatcher({
      introspectionQueryResultData,
    }),
  }),
});

export default client;
