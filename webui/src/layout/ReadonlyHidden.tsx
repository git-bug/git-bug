import React from 'react';

import CurrentIdentityContext from './CurrentIdentityContext';

type Props = { children: React.ReactNode };
const ReadonlyHidden = ({ children }: Props) => (
  <CurrentIdentityContext.Consumer>
    {context => {
      if (!context) return null;
      const { loading, error, data } = context;

      if (error || loading || !data?.repository?.userIdentity) return null;

      return <>{children}</>;
    }}
  </CurrentIdentityContext.Consumer>
);

export default ReadonlyHidden;
