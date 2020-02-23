import React from 'react';

import MAvatar from '@material-ui/core/Avatar';
import Tooltip from '@material-ui/core/Tooltip/Tooltip';

import { AuthoredFragment } from './fragments.generated';

type Props = AuthoredFragment & {
  className?: string;
  bold?: boolean;
};

const Author = ({ author, ...props }: Props) => {
  if (!author.email) {
    return <span {...props}>{author.displayName}</span>;
  }

  return (
    <Tooltip title={author.email}>
      <span {...props}>{author.displayName}</span>
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
