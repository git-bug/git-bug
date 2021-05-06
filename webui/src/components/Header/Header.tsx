import React from 'react';
import { Link, useLocation } from 'react-router-dom';

import AppBar from '@material-ui/core/AppBar';
import Tab, { TabProps } from '@material-ui/core/Tab';
import Tabs from '@material-ui/core/Tabs';
import Toolbar from '@material-ui/core/Toolbar';
import Tooltip from '@material-ui/core/Tooltip/Tooltip';
import { fade, makeStyles } from '@material-ui/core/styles';

import CurrentIdentity from '../Identity/CurrentIdentity';
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
    marginRight: '20px',
    color: fade(theme.palette.primary.contrastText, 0.5),
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

const DisabledTabWithTooltip = (props: TabProps) => {
  /*The span elements around disabled tabs are needed, as the tooltip
   * won't be triggered by disabled elements.
   * See: https://material-ui.com/components/tooltips/#disabled-elements
   * This must be done in a wrapper component, otherwise the TabS component
   * cannot pass it styles down to the Tab component. Resulting in (console)
   * warnings. This wrapper acceps the passed down TabProps and pass it around
   * the span element to the Tab component.
   */
  const msg = `This feature doesn't exist yet. Come help us build it.`;
  return (
    <Tooltip title={msg}>
      <span>
        <Tab disabled {...props} />
      </span>
    </Tooltip>
  );
};

function Header() {
  const classes = useStyles();
  const location = useLocation();

  // Prevents error of invalid tab selection in <Tabs>
  // Will return a valid tab path or false if path is unkown.
  function highlightTab() {
    const validTabs = ['/', '/code', '/pulls', '/settings'];
    const tab = validTabs.find((tabPath) => tabPath === location.pathname);
    return tab === undefined ? false : tab;
  }

  return (
    <>
      <AppBar position="fixed" className={classes.appBar}>
        <Toolbar>
          <Link to="/" className={classes.appTitle}>
            <img src="/logo.svg" className={classes.logo} alt="git-bug logo" />
            git-bug
          </Link>
          <div className={classes.filler} />
          <LightSwitch className={classes.lightSwitch} />
          <CurrentIdentity />
        </Toolbar>
      </AppBar>
      <div className={classes.offset} />
      <Tabs centered value={highlightTab()} aria-label="nav tabs">
        <DisabledTabWithTooltip label="Code" value="/code" {...a11yProps(1)} />
        <Tab label="Bugs" value="/" component={Link} to="/" {...a11yProps(2)} />
        <DisabledTabWithTooltip
          label="Pull Requests"
          value="/pulls"
          {...a11yProps(3)}
        />
        <DisabledTabWithTooltip
          label="Settings"
          value="/settings"
          {...a11yProps(4)}
        />
      </Tabs>
    </>
  );
}

export default Header;
