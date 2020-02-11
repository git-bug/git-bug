import React from 'react';
import Avatar from '@material-ui/core/Avatar';
import { makeStyles } from '@material-ui/core/styles';

import { useCurrentIdentityQuery } from './CurrentIdentity.generated';

const useStyles = makeStyles(theme => ({
  displayName: {
    marginLeft: theme.spacing(2),
  },
}));

const CurrentIdentity = () => {
  const classes = useStyles();
  const { loading, error, data } = useCurrentIdentityQuery();

  if (error || loading || !data?.defaultRepository?.userIdentity) return null;

  const user = data.defaultRepository.userIdentity;
  return (
    <>
      <Avatar src={user.avatarUrl ? user.avatarUrl : undefined}>
        {user.displayName.charAt(0).toUpperCase()}
      </Avatar>
      <div className={classes.displayName}>{user.displayName}</div>
    </>
  );
};

export default CurrentIdentity;
