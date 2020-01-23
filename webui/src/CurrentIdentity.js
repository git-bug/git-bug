import React from 'react';
import gql from 'graphql-tag';
import { Query } from 'react-apollo';
import Avatar from '@material-ui/core/Avatar';
import { makeStyles } from '@material-ui/styles';

const useStyles = makeStyles(theme => ({
  displayName: {
    marginLeft: theme.spacing(2),
  },
}));

const QUERY = gql`
  {
    defaultRepository {
      userIdentity {
        displayName
        avatarUrl
      }
    }
  }
`;

const CurrentIdentity = () => {
  const classes = useStyles();
  return (
    <Query query={QUERY}>
      {({ loading, error, data }) => {
        if (error || loading || !data.defaultRepository.userIdentity)
          return null;
        const user = data.defaultRepository.userIdentity;
        return (
          <>
            <Avatar src={user.avatarUrl}>
              {user.displayName.charAt(0).toUpperCase()}
            </Avatar>
            <div className={classes.displayName}>{user.displayName}</div>
          </>
        );
      }}
    </Query>
  );
};

export default CurrentIdentity;
