import { install } from '@material-ui/styles';
import ThemeProvider from '@material-ui/styles/ThemeProvider';
import { createMuiTheme } from '@material-ui/core/styles';
import ApolloClient from 'apollo-boost';
import React from 'react';
import { ApolloProvider } from 'react-apollo';
import ReactDOM from 'react-dom';
import { BrowserRouter } from 'react-router-dom';

install();

// TODO(sandhose): this is temporary until Material-UI v4 goes out
const App = React.lazy(() => import('./App'));

const theme = createMuiTheme();

const client = new ApolloClient({
  uri: '/graphql',
});

ReactDOM.render(
  <ApolloProvider client={client}>
    <BrowserRouter>
      <ThemeProvider theme={theme}>
        <React.Suspense fallback={'Loading…'}>
          <App />
        </React.Suspense>
      </ThemeProvider>
    </BrowserRouter>
  </ApolloProvider>,
  document.getElementById('root')
);
