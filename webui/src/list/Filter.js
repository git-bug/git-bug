import React from 'react';
import { Link } from 'react-router-dom';
import { makeStyles } from '@material-ui/styles';

function parse(query) {
  // TODO: extract the rest of the query?
  const params = {};

  // TODO: support escaping without quotes
  const re = /(\w+):(\w+|(["'])(([^\3]|\\.)*)\3)+/g;
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
    fontWeight: ({ active }) => (active ? 500 : 400),
    textDecoration: 'none',
    display: 'flex',
    alignSelf: ({ end }) => (end ? 'flex-end' : 'auto'),
    background: 'none',
    border: 'none',
  },
  icon: {
    paddingRight: theme.spacing(0.5),
  },
}));

function Filter({ active, to, children, icon: Icon, end, ...props }) {
  const classes = useStyles({ active, end });

  const content = (
    <>
      {Icon && <Icon fontSize="small" classes={{ root: classes.icon }} />}
      <div>{children}</div>
    </>
  );

  if (to) {
    return (
      <Link to={to} {...props} className={classes.element}>
        {content}
      </Link>
    );
  }

  return (
    <button {...props} className={classes.element}>
      {content}
    </button>
  );
}

export default Filter;
export { parse, stringify, quote };
