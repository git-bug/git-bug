import React from 'react';
import { Link as RouterLink } from 'react-router-dom';

import {
  Button,
  ClickAwayListener,
  Grow,
  Link,
  MenuItem,
  MenuList,
  Paper,
  Popper,
} from '@material-ui/core';
import Avatar from '@material-ui/core/Avatar';
import { makeStyles } from '@material-ui/core/styles';
import LockIcon from '@material-ui/icons/Lock';

import { useCurrentIdentityQuery } from './CurrentIdentity.generated';

const useStyles = makeStyles((theme) => ({
  displayName: {
    marginLeft: theme.spacing(2),
  },
  hidden: {
    display: 'none',
  },
  profileLink: {
    ...theme.typography.button,
  },
  popupButton: {
    textTransform: 'none',
    color: theme.palette.primary.contrastText,
  },
}));

const CurrentIdentity = () => {
  const classes = useStyles();
  const { loading, error, data } = useCurrentIdentityQuery();

  const [open, setOpen] = React.useState(false);
  const anchorRef = React.useRef<HTMLButtonElement>(null);

  if (error || loading || !data?.repository?.userIdentity) return null;

  const user = data.repository.userIdentity;

  const handleToggle = () => {
    setOpen((prevOpen) => !prevOpen);
  };

  const handleClose = (event: any) => {
    if (anchorRef.current && anchorRef.current.contains(event.target)) {
      return;
    }
    setOpen(false);
  };

  return (
    <>
      <Button
        ref={anchorRef}
        aria-controls={open ? 'menu-list-grow' : undefined}
        aria-haspopup="true"
        onClick={handleToggle}
        className={classes.popupButton}
      >
        <Avatar src={user.avatarUrl ? user.avatarUrl : undefined}>
          {user.displayName.charAt(0).toUpperCase()}
        </Avatar>
        <div className={classes.displayName}>{user.displayName}</div>
        <LockIcon
          color="secondary"
          className={user.isProtected ? '' : classes.hidden}
        />
      </Button>
      <Popper
        open={open}
        anchorEl={anchorRef.current}
        role={undefined}
        transition
        disablePortal
      >
        {({ TransitionProps, placement }) => (
          <Grow
            {...TransitionProps}
            style={{
              transformOrigin:
                placement === 'bottom' ? 'center top' : 'center bottom',
            }}
          >
            <Paper>
              <ClickAwayListener onClickAway={handleClose}>
                <MenuList autoFocusItem={open} id="menu-list-grow">
                  <MenuItem>
                    <Link
                      color="inherit"
                      className={classes.profileLink}
                      component={RouterLink}
                      to={`/user/${user.humanId}`}
                    >
                      Open profile
                    </Link>
                  </MenuItem>
                </MenuList>
              </ClickAwayListener>
            </Paper>
          </Grow>
        )}
      </Popper>
    </>
  );
};

export default CurrentIdentity;
