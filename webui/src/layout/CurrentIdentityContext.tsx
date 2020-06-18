import React from 'react';

import { CurrentIdentityQueryResult } from './CurrentIdentity.generated';

const Context = React.createContext(null as CurrentIdentityQueryResult | null);
export default Context;
