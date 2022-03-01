import CssBaseline from '@mui/material/CssBaseline';
import * as React from 'react';

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
