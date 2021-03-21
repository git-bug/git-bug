import React from 'react';
import { Link } from 'react-router-dom';

import AppBar from '@material-ui/core/AppBar';
import Tab from '@material-ui/core/Tab';
import Tabs from '@material-ui/core/Tabs';
import Toolbar from '@material-ui/core/Toolbar';
import Tooltip from '@material-ui/core/Tooltip/Tooltip';
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

  const tooltipMsg = `This feature doesn't exist yet. Come help us build it.`;

  /*The span elements around disabled tabs are needed, as the tooltip
   * won't be triggered by disabled elements.
   * See: https://material-ui.com/components/tooltips/#disabled-elements
   */
  return (
    <Tabs
      centered
      value={value}
      onChange={handleChange}
      aria-label="nav tabs example"
    >
      <Tooltip title={tooltipMsg}>
        <span>
          <Tab
            disabled
            label="Code"
            component="a"
            href="/code"
            {...a11yProps(0)}
          />
        </span>
      </Tooltip>
      <Tab label="Bugs" component="a" href="/" {...a11yProps(1)} />
      <Tooltip title={tooltipMsg}>
        <span>
          <Tab
            disabled
            label="Pull Requests"
            component="a"
            href="/pulls"
            {...a11yProps(2)}
          />
        </span>
      </Tooltip>
      <Tooltip title={tooltipMsg}>
        <span>
          <Tab
            disabled
            label="Settings"
            component="a"
            href="/settings"
            {...a11yProps(3)}
          />
        </span>
      </Tooltip>
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
