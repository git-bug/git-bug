import React from 'react';
import { Link as RouterLink } from 'react-router-dom';

import MAvatar from '@material-ui/core/Avatar';
import Link from '@material-ui/core/Link';
import Tooltip from '@material-ui/core/Tooltip/Tooltip';

import { AuthoredFragment } from '../graphql/fragments.generated';

type Props = AuthoredFragment & {
  className?: string;
  bold?: boolean;
};

const Author = ({ author, ...props }: Props) => {
  return (
    <Tooltip title={`Goto the ${author.displayName}'s profile.`}>
      <Link {...props} component={RouterLink} to={`/user/${author.humanId}`}>
        {author.displayName}
      </Link>
    </Tooltip>
  );
};

export const Avatar = ({ author, ...props }: Props) => {
  if (author.avatarUrl) {
    return <MAvatar src={author.avatarUrl} {...props} />;
  }

  return <MAvatar {...props}>{author.displayName[0]}</MAvatar>;
};

export default Author;
