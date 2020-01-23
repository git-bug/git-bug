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
      {({ error, data }) => {
        if (
          error ||
          !data ||
          !data.defaultRepository ||
          !data.defaultRepository.userIdentity ||
          !data.defaultRepository.userIdentity.displayName
        )
          return <></>;
        const displayName =
          data.defaultRepository.userIdentity.displayName || '';
        const avatar = data.defaultRepository.userIdentity.avatarUrl;
        return (
          <>
            <Avatar src={avatar}>{displayName.charAt(0).toUpperCase()}</Avatar>
            <div className={classes.displayName}>{displayName}</div>
          </>
        );
      }}
    </Query>
  );
};

export default CurrentIdentity;
