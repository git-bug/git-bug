import React from 'react';

import {
  Checkbox,
  FormControlLabel,
  Link,
  Paper,
  Typography,
} from '@material-ui/core';
import Avatar from '@material-ui/core/Avatar';
import { makeStyles } from '@material-ui/core/styles';
import InfoIcon from '@material-ui/icons/Info';
import MailOutlineIcon from '@material-ui/icons/MailOutline';

import { useCurrentIdentityQuery } from '../../components/CurrentIdentity/CurrentIdentity.generated';

import BugList from './BugList';

const useStyles = makeStyles((theme) => ({
  main: {
    maxWidth: 1000,
    margin: 'auto',
    marginTop: theme.spacing(4),
    padding: theme.spacing(3, 2),
    display: 'flex',
  },
  container: {
    display: 'flex',
    marginBottom: theme.spacing(1),
  },
  leftSidebar: {
    marginTop: theme.spacing(2),
    flex: '0 0 200px',
  },
  content: {
    marginTop: theme.spacing(5),
    padding: theme.spacing(3, 2),
    minWidth: 800,
    backgroundColor: theme.palette.background.paper,
  },
  rightSidebar: {
    marginTop: theme.spacing(5),
    flex: '0 0 200px',
  },
  large: {
    width: theme.spacing(20),
    height: theme.spacing(20),
  },
  control: {
    paddingBottom: theme.spacing(3),
  },
  header: {
    ...theme.typography.h4,
  },
}));

const Identity = () => {
  const classes = useStyles();
  const { data } = useCurrentIdentityQuery();
  const user = data?.repository?.userIdentity;
  console.log(user);

  return (
    <main className={classes.main}>
      <div className={classes.container}>
        <div className={classes.leftSidebar}>
          <h1 className={classes.header}>
            {user?.displayName ? user?.displayName : 'none'}
          </h1>
          <Avatar
            src={user?.avatarUrl ? user.avatarUrl : undefined}
            className={classes.large}
          >
            {user?.displayName.charAt(0).toUpperCase()}
          </Avatar>
          <Typography variant="h5" component="h2">
            Your account
          </Typography>
          <Typography variant="subtitle2" component="h2">
            Name: {user?.name ? user?.name : '---'}
          </Typography>
          <Typography variant="subtitle2" component="h3">
            Id (truncated): {user?.humanId ? user?.humanId : '---'}
            <InfoIcon
              fontSize={'small'}
              titleAccess={user?.id ? user?.id : '---'}
            />
          </Typography>
          <Typography variant="subtitle2" component="h3">
            Login: {user?.login ? user?.login : '---'}
          </Typography>
          <Typography
            variant="subtitle2"
            component="h3"
            style={{
              display: 'flex',
              alignItems: 'center',
              flexWrap: 'wrap',
            }}
          >
            <MailOutlineIcon />
            <Link href={'mailto:' + user?.email} color={'inherit'}>
              {user?.email ? user?.email : '---'}
            </Link>
          </Typography>
          <FormControlLabel
            className={classes.control}
            label="Protected"
            labelPlacement="end"
            value={user?.isProtected}
            control={<Checkbox color="secondary" indeterminate />}
          />
        </div>
        <Paper className={classes.content}>
          <Typography variant="h5" component="h2">
            Bugs authored by {user?.displayName}
          </Typography>
          <BugList humanId={user?.humanId ? user?.humanId : ''} />
        </Paper>
        <div className={classes.rightSidebar}></div>
      </div>
    </main>
  );
};

export default Identity;
