import React from 'react';

import CssBaseline from '@material-ui/core/CssBaseline';

import Header from './Header';

type Props = { children: React.ReactNode };
function Layout({ children }: Props) {
  return (
    <>
      <CssBaseline />
      <Header />
      {children}
    </>
  );
}

export default Layout;
