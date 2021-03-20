import React from 'react';
import { Link } from 'react-router-dom';

import AppBar from '@material-ui/core/AppBar';
import Tab from '@material-ui/core/Tab';
import Tabs from '@material-ui/core/Tabs';
import Toolbar from '@material-ui/core/Toolbar';
import { makeStyles } from '@material-ui/core/styles';

import { LightSwitch } from '../../components/Themer';
import CurrentIdentity from '../CurrentIdentity/CurrentIdentity';

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
    padding: '0 20px',
  },
  logo: {
    height: '42px',
    marginRight: theme.spacing(2),
  },
}));

function a11yProps(index: any) {
  return {
    id: `nav-tab-${index}`,
    'aria-controls': `nav-tabpanel-${index}`,
  };
}

function NavTabs() {
  const [value, setValue] = React.useState(0);

  //TODO page refresh resets state. Must parse url to determine which tab is
  //highlighted
  const handleChange = (event: React.ChangeEvent<{}>, newValue: number) => {
    setValue(newValue);
  };

  return (
    <Tabs
      variant="fullWidth"
      value={value}
      onChange={handleChange}
      aria-label="nav tabs example"
    >
      <Tab label="Code" component="a" href="/code" {...a11yProps(0)} />
      <Tab label="Bugs" component="a" href="/" {...a11yProps(1)} />
      <Tab
        label="Pull Requests"
        component="a"
        href="/pulls"
        {...a11yProps(2)}
      />
      <Tab label="Projects" component="a" href="/projects" {...a11yProps(3)} />
      <Tab label="Wiki" component="a" href="/wiki" {...a11yProps(4)} />
      <Tab label="Settings" component="a" href="/settings" {...a11yProps(5)} />
    </Tabs>
  );
}

function Header() {
  const classes = useStyles();

  return (
    <>
      <AppBar position="fixed" className={classes.appBar}>
        <Toolbar>
          <Link to="/" className={classes.appTitle}>
            <img src="/logo.svg" className={classes.logo} alt="git-bug" />
            git-bug
          </Link>
          <div className={classes.filler} />
          <div className={classes.lightSwitch}>
            <LightSwitch />
          </div>
          <CurrentIdentity />
        </Toolbar>
      </AppBar>
      <div className={classes.offset} />
      <NavTabs />
    </>
  );
}

export default Header;
