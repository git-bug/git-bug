import React from 'react';

import {
  Checkbox,
  Divider,
  FormControl,
  FormControlLabel,
  FormGroup,
  FormLabel,
  Paper,
  TextField,
} from '@material-ui/core';
import Avatar from '@material-ui/core/Avatar';
import { makeStyles } from '@material-ui/core/styles';

import { useCurrentIdentityQuery } from '../../components/CurrentIdentity/CurrentIdentity.generated';
const useStyles = makeStyles((theme) => ({
  main: {
    maxWidth: 1200,
    margin: 'auto',
    marginTop: theme.spacing(4),
  },
  container: {
    display: 'flex',
    marginBottom: theme.spacing(1),
    marginRight: theme.spacing(2),
    marginLeft: theme.spacing(2),
  },
  leftSidebar: {
    marginTop: theme.spacing(2),
    marginRight: theme.spacing(2),
  },
  content: {
    marginTop: theme.spacing(5),
    marginRight: theme.spacing(4),
    padding: theme.spacing(3, 2),
    minWidth: 800,
    display: 'flex',
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
}));

const Identity = () => {
  // eslint-disable-next-line
  const classes = useStyles();
  // eslint-disable-next-line
  const { loading, error, data } = useCurrentIdentityQuery();
  const user = data?.repository?.userIdentity;
  console.log(user);
  return (
    <main className={classes.main}>
      <div className={classes.container}>
        <div className={classes.leftSidebar}>
          <h1>{user?.displayName ? user?.displayName : 'none'}</h1>
          <Avatar src={user?.avatarUrl ? user.avatarUrl : undefined}>
            {user?.displayName.charAt(0).toUpperCase()}
          </Avatar>
        </div>
        <Paper className={classes.content}>
          <Divider variant="fullWidth" />
          <FormControl component="fieldset">
            <FormGroup>
              <FormLabel className={classes.control} component="legend">
                Your account
              </FormLabel>
              <TextField
                className={classes.control}
                label="Name"
                variant="outlined"
                value={user?.name ? user?.name : '---'}
              />
              <TextField
                className={classes.control}
                label="Id (truncated)"
                variant="outlined"
                value={user?.humanId ? user?.humanId : '---'}
              />
              <TextField
                className={classes.control}
                label="Id (full)"
                variant="outlined"
                value={user?.id ? user?.id : '---'}
              />
              <TextField
                className={classes.control}
                label="E-Mail"
                variant="outlined"
                value={user?.email ? user?.email : '---'}
              />
              <TextField
                className={classes.control}
                label="Login"
                variant="outlined"
                value={user?.login ? user?.login : '---'}
              />
              <FormControlLabel
                className={classes.control}
                label="Protected"
                labelPlacement="end"
                value={user?.isProtected}
                control={<Checkbox color="secondary" indeterminate />}
              />
            </FormGroup>
          </FormControl>
        </Paper>
        <div className={classes.rightSidebar}>
          <Avatar
            src={user?.avatarUrl ? user.avatarUrl : undefined}
            className={classes.large}
          >
            {user?.displayName.charAt(0).toUpperCase()}
          </Avatar>
        </div>
      </div>
    </main>
  );
};

export default Identity;
