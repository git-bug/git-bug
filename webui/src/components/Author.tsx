import MAvatar from '@mui/material/Avatar';
import Link from '@mui/material/Link';
import Tooltip from '@mui/material/Tooltip/Tooltip';
import { Link as RouterLink } from 'react-router';

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

// Backend util/text.ValidUrl already restricts avatarUrl to http(s), but
// re-check in the UI: stored data might predate that validation, and a
// defensive scheme filter here is cheap insurance against data:/javascript:
// URLs being passed to an <img src>.
const isSafeHttpUrl = (u?: string | null) => !!u && /^https?:\/\//i.test(u);

export const Avatar = ({ author, ...props }: Props) => {
  if (isSafeHttpUrl(author.avatarUrl)) {
    return <MAvatar src={author.avatarUrl!} {...props} />;
  }

  return <MAvatar {...props}>{author.displayName[0]}</MAvatar>;
};

export default Author;
