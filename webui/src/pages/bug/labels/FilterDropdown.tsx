import React, { useEffect, useRef, useState } from 'react';

import { IconButton, MenuItem } from '@material-ui/core';
import Menu from '@material-ui/core/Menu';
import TextField from '@material-ui/core/TextField';
import { makeStyles, withStyles } from '@material-ui/core/styles';
import SettingsIcon from '@material-ui/icons/Settings';

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
  labelsheader: {
    display: 'flex',
    flexDirection: 'row',
  },
}));

export type FilterMenuItem<T> = {
  render: (item: T) => React.ReactNode;
};

type FilterDropdownProps<T> = {
  items?: FilterMenuItem<T>[];
  hasFilter?: boolean;
  onFilterChange: (filter: string) => void;
} & React.ButtonHTMLAttributes<HTMLButtonElement>;

function FilterDropdown<T>({
  items,
  hasFilter,
  onFilterChange,
}: FilterDropdownProps<T>) {
  const buttonRef = useRef<HTMLButtonElement>(null);
  const searchRef = useRef<HTMLButtonElement>(null);
  const classes = useStyles({ active: false });

  const [open, setOpen] = useState(false);
  const [filter, setFilter] = useState<string>('');

  useEffect(() => {
    searchRef && searchRef.current && searchRef.current.focus();
  }, [filter]);

  /*function sortBySelection(x: FilterMenuItem, y: FilterMenuItem) {
    if (x.isSelected() === y.isSelected()) {
      return 0;
    } else if (x.isSelected()) {
      return -1;
    } else {
      return 1;
    }
  }

  const sortedItems = items.sort(sortBySelection);*/

  return (
    <>
      <div className={classes.labelsheader}>
        Labels
        <IconButton
          ref={buttonRef}
          onClick={() => setOpen(!open)}
          className={classes.element}
        >
          <SettingsIcon fontSize={'small'} />
        </IconButton>
      </div>

      <Menu
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
        onClose={() => {
          setOpen(false);
          //const selectedLabels = dropdown
          //  .map(([key]) => (itemActive(key) ? key : ''))
          //  .filter((entry) => entry !== '');
          //onClose(selectedLabels);
        }}
        onExited={() => setFilter('')}
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
              onFilterChange(value);
            }}
            onKeyDown={(e) => e.stopPropagation()}
            value={filter}
            label={`Filter Labels`}
          />
        )}
        {items &&
          items.map((item, index) => {
            return <MenuItem key={index}>{item}</MenuItem>;
          })}
      </Menu>
    </>
  );
}

export default FilterDropdown;
