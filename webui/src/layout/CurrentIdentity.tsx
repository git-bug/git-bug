import React from 'react';

import Avatar from '@material-ui/core/Avatar';
import { makeStyles } from '@material-ui/core/styles';

import CurrentIdentityContext from './CurrentIdentityContext';

const useStyles = makeStyles(theme => ({
  displayName: {
    marginLeft: theme.spacing(2),
  },
}));

const CurrentIdentity = () => {
  const classes = useStyles();

  return (
    <CurrentIdentityContext.Consumer>
      {context => {
        if (!context) return null;
        const { loading, error, data } = context as any;

        if (error || loading || !data?.repository?.userIdentity) return null;

        const user = data.repository.userIdentity;
        return (
          <>
            <Avatar src={user.avatarUrl ? user.avatarUrl : undefined}>
              {user.displayName.charAt(0).toUpperCase()}
            </Avatar>
            <div className={classes.displayName}>{user.displayName}</div>
          </>
        );
      }}
    </CurrentIdentityContext.Consumer>
  );
};

export default CurrentIdentity;
