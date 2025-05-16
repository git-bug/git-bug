import AppBar from '@mui/material/AppBar';
import Toolbar from '@mui/material/Toolbar';
import { alpha } from '@mui/material/styles';
import makeStyles from '@mui/styles/makeStyles';
import { Link } from 'react-router';

import CurrentIdentity from '../Identity/CurrentIdentity';
import CurrentRepository from '../Identity/CurrentRepository';
import { LightSwitch } from '../Themer';

const useStyles = makeStyles((theme) => ({
  offset: {
    ...theme.mixins.toolbar,
  },
  filler: {
    flexGrow: 1,
  },
  appBar: {
    backgroundColor: theme.palette.primary.dark,
    color: theme.palette.primary.contrastText,
  },
  appTitle: {
    ...theme.typography.h6,
    color: theme.palette.primary.contrastText,
    textDecoration: 'none',
    display: 'flex',
    alignItems: 'center',
  },
  lightSwitch: {
    marginRight: theme.spacing(2),
    color: alpha(theme.palette.primary.contrastText, 0.5),
  },
  logo: {
    height: '42px',
    marginRight: theme.spacing(2),
  },
}));

function Header() {
  const classes = useStyles();

  return (
    <>
      <AppBar position="fixed" className={classes.appBar}>
        <Toolbar>
          <Link to="/" className={classes.appTitle}>
            <img src="/logo.svg" className={classes.logo} alt="git-bug logo" />
            <CurrentRepository default="git-bug" />
          </Link>
          <div className={classes.filler} />
          <LightSwitch className={classes.lightSwitch} />
          <CurrentIdentity />
        </Toolbar>
      </AppBar>
      <div className={classes.offset}></div>
    </>
  );
}

export default Header;
