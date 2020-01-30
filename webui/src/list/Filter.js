import React, { useState, useRef } from 'react';
import { Link } from 'react-router-dom';
import { makeStyles } from '@material-ui/styles';
import Menu from '@material-ui/core/Menu';
import MenuItem from '@material-ui/core/MenuItem';
import ArrowDropDown from '@material-ui/icons/ArrowDropDown';

function parse(query) {
  // TODO: extract the rest of the query?
  const params = {};

  // TODO: support escaping without quotes
  const re = /(\w+):([A-Za-z0-9-]+|(["'])(([^\3]|\\.)*)\3)+/g;
  let matches;
  while ((matches = re.exec(query)) !== null) {
    if (!params[matches[1]]) {
      params[matches[1]] = [];
    }

    let value;
    if (matches[4]) {
      value = matches[4];
    } else {
      value = matches[2];
    }
    value = value.replace(/\\(.)/g, '$1');
    params[matches[1]].push(value);
  }
  return params;
}

function quote(value) {
  const hasSingle = value.includes("'");
  const hasDouble = value.includes('"');
  const hasSpaces = value.includes(' ');
  if (!hasSingle && !hasDouble && !hasSpaces) {
    return value;
  }

  if (!hasDouble) {
    return `"${value}"`;
  }

  if (!hasSingle) {
    return `'${value}'`;
  }

  value = value.replace(/"/g, '\\"');
  return `"${value}"`;
}

function stringify(params) {
  const parts = Object.entries(params).map(([key, values]) => {
    return values.map(value => `${key}:${quote(value)}`);
  });
  return [].concat(...parts).join(' ');
}

const useStyles = makeStyles(theme => ({
  element: {
    ...theme.typography.body2,
    color: ({ active }) => (active ? '#333' : '#444'),
    padding: theme.spacing(0, 1),
    fontWeight: ({ active }) => (active ? 600 : 400),
    textDecoration: 'none',
    display: 'flex',
    background: 'none',
    border: 'none',
  },
  itemActive: {
    fontWeight: 600,
  },
  icon: {
    paddingRight: theme.spacing(0.5),
  },
}));

function Dropdown({ children, dropdown, itemActive, to, ...props }) {
  const [open, setOpen] = useState(false);
  const buttonRef = useRef();
  const classes = useStyles();

  return (
    <>
      <button ref={buttonRef} onClick={() => setOpen(!open)} {...props}>
        {children}
        <ArrowDropDown fontSize="small" />
      </button>
      <Menu
        getContentAnchorEl={null}
        anchorOrigin={{
          vertical: 'bottom',
          horizontal: 'left',
        }}
        transformOrigin={{
          vertical: 'top',
          horizontal: 'left',
        }}
        open={open}
        onClose={() => setOpen(false)}
        anchorEl={buttonRef.current}
      >
        {dropdown.map(([key, value]) => (
          <MenuItem
            component={Link}
            to={to(key)}
            className={itemActive(key) ? classes.itemActive : null}
            onClick={() => setOpen(false)}
            key={key}
          >
            {value}
          </MenuItem>
        ))}
      </Menu>
    </>
  );
}

function Filter({ active, to, children, icon: Icon, dropdown, ...props }) {
  const classes = useStyles({ active });

  const content = (
    <>
      {Icon && <Icon fontSize="small" classes={{ root: classes.icon }} />}
      <div>{children}</div>
    </>
  );

  if (dropdown) {
    return (
      <Dropdown
        {...props}
        to={to}
        dropdown={dropdown}
        className={classes.element}
      >
        {content}
      </Dropdown>
    );
  }

  if (to) {
    return (
      <Link to={to} {...props} className={classes.element}>
        {content}
      </Link>
    );
  }

  return <div className={classes.element}>{content}</div>;
}

export default Filter;
export { parse, stringify, quote };
