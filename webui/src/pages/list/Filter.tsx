import clsx from 'clsx';
import { LocationDescriptor } from 'history';
import React, { useRef, useState, useEffect } from 'react';
import { Link } from 'react-router-dom';

import Menu from '@material-ui/core/Menu';
import MenuItem from '@material-ui/core/MenuItem';
import { SvgIconProps } from '@material-ui/core/SvgIcon';
import TextField from '@material-ui/core/TextField';
import { makeStyles, withStyles } from '@material-ui/core/styles';
import { darken } from '@material-ui/core/styles/colorManipulator';
import ArrowDropDown from '@material-ui/icons/ArrowDropDown';
import CheckIcon from '@material-ui/icons/Check';

import { Color } from '../../gqlTypes';

const CustomTextField = withStyles((theme) => ({
  root: {
    margin: '0 8px 12px 8px',
    '& label.Mui-focused': {
      margin: '0 2px',
      color: theme.palette.text.secondary,
    },
    '& .MuiInput-underline::before': {
      borderBottomColor: theme.palette.divider,
    },
    '& .MuiInput-underline::after': {
      borderBottomColor: theme.palette.divider,
    },
  },
}))(TextField);

const ITEM_HEIGHT = 48;

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
    color: theme.palette.text.secondary,
    padding: theme.spacing(0, 1),
    fontWeight: 400,
    textDecoration: 'none',
    display: 'flex',
    background: 'none',
    border: 'none',
  },
  itemActive: {
    fontWeight: 600,
    color: theme.palette.text.primary,
  },
  icon: {
    paddingRight: theme.spacing(0.5),
  },
  labelMenu: {
    '& .MuiMenu-paper': {
      //somehow using "width" won't override the default width...
      minWidth: '35ch',
    },
  },
  labelMenuItem: {
    whiteSpace: 'normal',
    wordBreak: 'break-word',
    display: 'flex',
    alignItems: 'initial',
  },
  labelcolor: {
    minWidth: '0.5rem',
    display: 'flex',
    borderRadius: '0.25rem',
    marginRight: '5px',
    marginLeft: '3px',
  },
}));
const _rgb = (color: Color) =>
  'rgb(' + color.R + ',' + color.G + ',' + color.B + ')';

// Create a style object from the label RGB colors
const createStyle = (color: Color) => ({
  backgroundColor: _rgb(color),
  borderBottomColor: darken(_rgb(color), 0.2),
});

type DropdownTuple = [string, string, Color?];

type FilterDropdownProps = {
  children: React.ReactNode;
  dropdown: DropdownTuple[];
  itemActive: (key: string) => boolean;
  icon?: React.ComponentType<SvgIconProps>;
  to: (key: string) => LocationDescriptor;
  hasFilter?: boolean;
} & React.ButtonHTMLAttributes<HTMLButtonElement>;

function FilterDropdown({
  children,
  dropdown,
  itemActive,
  icon: Icon,
  to,
  hasFilter,
  ...props
}: FilterDropdownProps) {
  const [open, setOpen] = useState(false);
  const [filter, setFilter] = useState<string>('');
  const buttonRef = useRef<HTMLButtonElement>(null);
  const searchRef = useRef<HTMLButtonElement>(null);
  const classes = useStyles({ active: false });

  useEffect(() => {
    searchRef && searchRef.current && searchRef.current.focus();
  }, [filter]);

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
        className={classes.labelMenu}
        getContentAnchorEl={null}
        ref={searchRef}
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
        PaperProps={{
          style: {
            maxHeight: ITEM_HEIGHT * 4.5,
            width: '25ch',
          },
        }}
      >
        {hasFilter && (
          <CustomTextField
            onChange={(e) => {
              const { value } = e.target;
              setFilter(value);
            }}
            onKeyDown={(e) => e.stopPropagation()}
            value={filter}
            label={`Filter ${children}`}
          />
        )}
        {dropdown
          .filter((d) => d[1].toLowerCase().includes(filter.toLowerCase()))
          .map(([key, value, color]) => (
            <MenuItem
              component={Link}
              to={to(key)}
              className={classes.labelMenuItem}
              selected={itemActive(key)}
              onClick={() => setOpen(false)}
              key={key}
            >
              {itemActive(key) && <CheckIcon />}
              {color && (
                <div
                  className={classes.labelcolor}
                  style={createStyle(color)}
                />
              )}
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
