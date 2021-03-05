import React from 'react';

import {
  Button,
  ClickAwayListener,
  Grow,
  MenuItem,
  MenuList,
  Paper,
  Popper,
} from '@material-ui/core';
import Avatar from '@material-ui/core/Avatar';
import { makeStyles } from '@material-ui/core/styles';

import { useCurrentIdentityQuery } from './CurrentIdentity.generated';

const useStyles = makeStyles((theme) => ({
  displayName: {
    marginLeft: theme.spacing(2),
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
      >
        <Avatar src={user.avatarUrl ? user.avatarUrl : undefined}>
          {user.displayName.charAt(0).toUpperCase()}
        </Avatar>
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
                  <MenuItem>Display Name: {user.displayName}</MenuItem>
                  <MenuItem>Human Id: {user.humanId}</MenuItem>
                  <MenuItem>Email: {user.email}</MenuItem>
                </MenuList>
              </ClickAwayListener>
            </Paper>
          </Grow>
        )}
      </Popper>
      <div className={classes.displayName}>{user.displayName}</div>
    </>
  );
};

export default CurrentIdentity;
