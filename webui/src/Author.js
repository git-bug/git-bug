import gql from 'graphql-tag';
import Tooltip from '@material-ui/core/Tooltip/Tooltip';
import MAvatar from '@material-ui/core/Avatar';
import React from 'react';

const Author = ({ author, ...props }) => {
  if (!author.email) {
    return <span {...props}>{author.displayName}</span>;
  }

  return (
    <Tooltip title={author.email}>
      <span {...props}>{author.displayName}</span>
    </Tooltip>
  );
};

Author.fragment = gql`
  fragment authored on Authored {
    author {
      name
      email
      displayName
      avatarUrl
    }
  }
`;

export const Avatar = ({ author, ...props }) => {
  if (author.avatarUrl) {
    return <MAvatar src={author.avatarUrl} {...props} />;
  }

  return <MAvatar {...props}>{author.displayName[0]}</MAvatar>;
};

export default Author;
