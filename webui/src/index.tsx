import { ApolloProvider } from '@apollo/client';
import React from 'react';
import ReactDOM from 'react-dom';
import { BrowserRouter } from 'react-router-dom';

import App from './App';
import apolloClient from './apollo';
import Themer from './components/Themer';
import { defaultLightTheme, defaultDarkTheme } from './themes/index';

ReactDOM.render(
  <ApolloProvider client={apolloClient}>
    <BrowserRouter>
      <Themer lightTheme={defaultLightTheme} darkTheme={defaultDarkTheme}>
        <App />
      </Themer>
    </BrowserRouter>
  </ApolloProvider>,
  document.getElementById('root')
);
