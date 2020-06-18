import React from 'react';

import CssBaseline from '@material-ui/core/CssBaseline';

import { useCurrentIdentityQuery } from './CurrentIdentity.generated';
import CurrentIdentityContext from './CurrentIdentityContext';
import Header from './Header';

type Props = { children: React.ReactNode };
function Layout({ children }: Props) {
  return (
    <CurrentIdentityContext.Provider value={useCurrentIdentityQuery()}>
      <CssBaseline />
      <Header />
      {children}
    </CurrentIdentityContext.Provider>
  );
}

export default Layout;
