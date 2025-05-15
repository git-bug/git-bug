import { ApolloProvider } from '@apollo/client';
import React from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';

import App from './App';
import apolloClient from './apollo';
import Themer from './components/Themer';
import { defaultLightTheme, defaultDarkTheme } from './themes/index';

const root = createRoot(document.getElementById('root') as HTMLElement);
root.render(
  <ApolloProvider client={apolloClient}>
    <BrowserRouter>
      <Themer lightTheme={defaultLightTheme} darkTheme={defaultDarkTheme}>
        <App />
      </Themer>
    </BrowserRouter>
  </ApolloProvider>
);
