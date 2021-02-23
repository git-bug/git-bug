import { ApolloProvider } from '@apollo/client';
import React from 'react';
import ReactDOM from 'react-dom';

import App from './App';
import apolloClient from './apollo';

ReactDOM.render(
  <ApolloProvider client={apolloClient}>
    <App />
  </ApolloProvider>,
  document.getElementById('root')
);
