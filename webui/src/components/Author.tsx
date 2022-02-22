import MAvatar from '@mui/material/Avatar';
import Link from '@mui/material/Link';
import Tooltip from '@mui/material/Tooltip/Tooltip';
import { Link as RouterLink } from 'react-router-dom';

import { AuthoredFragment } from '../graphql/fragments.generated';

type Props = AuthoredFragment & {
  className?: string;
  bold?: boolean;
};

const Author = ({ author, ...props }: Props) => {
  return (
    <Tooltip title={`Goto the ${author.displayName}'s profile.`}>
      <Link
        {...props}
        component={RouterLink}
        to={`/user/${author.id}`}
        underline="hover"
      >
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
