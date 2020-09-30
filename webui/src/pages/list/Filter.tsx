import clsx from 'clsx';
import { LocationDescriptor } from 'history';
import React, { useState, useRef } from 'react';
import { Link } from 'react-router-dom';

import Menu from '@material-ui/core/Menu';
import MenuItem from '@material-ui/core/MenuItem';
import { SvgIconProps } from '@material-ui/core/SvgIcon';
import { makeStyles } from '@material-ui/core/styles';
import ArrowDropDown from '@material-ui/icons/ArrowDropDown';

export type Query = { [key: string]: string[] };

function parse(query: string): Query {
  // TODO: extract the rest of the query?
  const params: Query = {};

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

function quote(value: string): string {
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

function stringify(params: Query): string {
  const parts: string[][] = Object.entries(params).map(([key, values]) => {
    return values.map((value) => `${key}:${quote(value)}`);
  });
  return new Array<string>().concat(...parts).join(' ');
}

const useStyles = makeStyles((theme) => ({
  element: {
    ...theme.typography.body2,
    color: '#444',
    padding: theme.spacing(0, 1),
    fontWeight: 400,
    textDecoration: 'none',
    display: 'flex',
    background: 'none',
    border: 'none',
  },
  itemActive: {
    fontWeight: 600,
    color: '#333',
  },
  icon: {
    paddingRight: theme.spacing(0.5),
  },
}));

type DropdownTuple = [string, string];

type FilterDropdownProps = {
  children: React.ReactNode;
  dropdown: DropdownTuple[];
  itemActive: (key: string) => boolean;
  icon?: React.ComponentType<SvgIconProps>;
  to: (key: string) => LocationDescriptor;
} & React.ButtonHTMLAttributes<HTMLButtonElement>;

function FilterDropdown({
  children,
  dropdown,
  itemActive,
  icon: Icon,
  to,
  ...props
}: FilterDropdownProps) {
  const [open, setOpen] = useState(false);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const classes = useStyles({ active: false });

  const content = (
    <>
      {Icon && <Icon fontSize="small" classes={{ root: classes.icon }} />}
      <div>{children}</div>
    </>
  );

  return (
    <>
      <button
        ref={buttonRef}
        onClick={() => setOpen(!open)}
        className={classes.element}
        {...props}
      >
        {content}
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
            className={itemActive(key) ? classes.itemActive : undefined}
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

export type FilterProps = {
  active: boolean;
  to: LocationDescriptor; // the target on click
  icon?: React.ComponentType<SvgIconProps>;
  children: React.ReactNode;
};
function Filter({ active, to, children, icon: Icon }: FilterProps) {
  const classes = useStyles();

  const content = (
    <>
      {Icon && <Icon fontSize="small" classes={{ root: classes.icon }} />}
      <div>{children}</div>
    </>
  );

  if (to) {
    return (
      <Link
        to={to}
        className={clsx(classes.element, active && classes.itemActive)}
      >
        {content}
      </Link>
    );
  }

  return (
    <div className={clsx(classes.element, active && classes.itemActive)}>
      {content}
    </div>
  );
}

export default Filter;
export { parse, stringify, quote, FilterDropdown, Filter };
