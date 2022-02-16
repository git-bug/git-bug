import * as React from 'react';
import { Link } from 'react-router-dom';

import { makeStyles } from '@material-ui/core/styles';

const useStyles = makeStyles((theme) => ({
  tag: {
    color: theme.palette.text.secondary,
  },
}));

const AnchorTag: React.FC = ({
  children,
  href,
}: React.HTMLProps<HTMLAnchorElement>) => {
  const classes = useStyles();
  const origin = window.location.origin;
  const destination = href === undefined ? '' : href;
  const isInternalLink =
    destination.startsWith('/') || destination.startsWith(origin);
  const internalDestination = destination.replace(origin, '');
  const internalLink = (
    <Link className={classes.tag} to={internalDestination}>
      {children}
    </Link>
  );
  const externalLink = (
    <a
      className={classes.tag}
      href={destination}
      target="_blank"
      rel="noopener noreferrer"
    >
      {children}
    </a>
  );

  return isInternalLink ? internalLink : externalLink;
};

export default AnchorTag;
