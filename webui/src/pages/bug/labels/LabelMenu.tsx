import React, { useEffect, useRef, useState } from 'react';

import { IconButton } from '@material-ui/core';
import Menu from '@material-ui/core/Menu';
import MenuItem from '@material-ui/core/MenuItem';
import { SvgIconProps } from '@material-ui/core/SvgIcon';
import TextField from '@material-ui/core/TextField';
import { makeStyles, withStyles } from '@material-ui/core/styles';
import { darken } from '@material-ui/core/styles/colorManipulator';
import CheckIcon from '@material-ui/icons/Check';
import SettingsIcon from '@material-ui/icons/Settings';

import { Color } from '../../../gqlTypes';
import {
  ListLabelsDocument,
  useListLabelsQuery,
} from '../../list/ListLabels.generated';
import { BugFragment } from '../Bug.generated';
import { GetBugDocument } from '../BugQuery.generated';

import { useSetLabelMutation } from './SetLabel.generated';

type DropdownTuple = [string, string, Color];

type FilterDropdownProps = {
  children: React.ReactNode;
  dropdown: DropdownTuple[];
  icon?: React.ComponentType<SvgIconProps>;
  hasFilter?: boolean;
  itemActive: (key: string) => boolean;
  onClose: () => void;
  toggleLabel: (key: string, active: boolean) => void;
  onNewItem: (name: string) => void;
} & React.ButtonHTMLAttributes<HTMLButtonElement>;

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
  itemActive: {
    fontWeight: 600,
    color: theme.palette.text.primary,
  },
  icon: {
    paddingRight: theme.spacing(0.5),
  },
  labelcolor: {
    width: '15px',
    height: '15px',
    display: 'flex',
    backgroundColor: 'blue',
    borderRadius: '0.25rem',
    marginRight: '5px',
    marginLeft: '3px',
  },
  labelsheader: {
    display: 'flex',
    flexDirection: 'row',
  },
  menuRow: {
    display: 'flex',
    flexDirection: 'row',
    alignItems: 'center',
    flexWrap: 'wrap',
  },
}));

const _rgb = (color: Color) =>
  'rgb(' + color.R + ',' + color.G + ',' + color.B + ')';

// Create a style object from the label RGB colors
const createStyle = (color: Color) => ({
  backgroundColor: _rgb(color),
  borderBottomColor: darken(_rgb(color), 0.2),
});

function FilterDropdown({
  children,
  dropdown,
  icon: Icon,
  hasFilter,
  itemActive,
  onClose,
  toggleLabel,
  onNewItem,
}: FilterDropdownProps) {
  const [open, setOpen] = useState(false);
  const [filter, setFilter] = useState<string>('');
  const buttonRef = useRef<HTMLButtonElement>(null);
  const searchRef = useRef<HTMLButtonElement>(null);
  const classes = useStyles({ active: false });

  useEffect(() => {
    searchRef && searchRef.current && searchRef.current.focus();
  }, [filter]);

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
          onClose();
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
            }}
            onKeyDown={(e) => e.stopPropagation()}
            value={filter}
            label={`Filter ${children}`}
          />
        )}
        {filter !== '' &&
          dropdown.filter((d) => d[1].toLowerCase() === filter.toLowerCase())
            .length <= 0 && (
            <MenuItem
              style={{ whiteSpace: 'normal', wordBreak: 'break-all' }}
              onClick={() => {
                onNewItem(filter);
                setFilter('');
                setOpen(false);
              }}
            >
              Create new label '{filter}'
            </MenuItem>
          )}
        {dropdown
          .sort(function (x, y) {
            // true values first
            return itemActive(x[1]) === itemActive(y[1]) ? 0 : x ? -1 : 1;
          })
          .filter((d) => d[1].toLowerCase().includes(filter.toLowerCase()))
          .map(([key, value, color]) => (
            <MenuItem
              style={{ whiteSpace: 'normal', wordBreak: 'break-all' }}
              onClick={() => {
                toggleLabel(key, itemActive(key));
              }}
              key={key}
              className={itemActive(key) ? classes.itemActive : undefined}
            >
              <div className={classes.menuRow}>
                {itemActive(key) ? <CheckIcon fontSize={'small'} /> : null}
                <div
                  className={classes.labelcolor}
                  style={createStyle(color)}
                />
                {value}
              </div>
            </MenuItem>
          ))}
      </Menu>
    </>
  );
}

type Props = {
  bug: BugFragment;
};
function LabelMenu({ bug }: Props) {
  const { data: labelsData } = useListLabelsQuery();
  const [bugLabelNames, setBugLabelNames] = useState(
    bug.labels.map((l) => l.name)
  );
  const [selectedLabels, setSelectedLabels] = useState(
    bug.labels.map((l) => l.name)
  );

  const [setLabelMutation] = useSetLabelMutation();

  useEffect(() => {});
  function toggleLabel(key: string, active: boolean) {
    const labels: string[] = active
      ? selectedLabels.filter((label) => label !== key)
      : selectedLabels.concat([key]);
    setSelectedLabels(labels);
    console.log('toggle (selected)');
    console.log(labels);
  }

  function diff(oldState: string[], newState: string[]) {
    console.log('oldState / Buglabels');
    console.log(oldState);
    console.log('newState / Selected');
    console.log(newState);
    const added = newState.filter((x) => !oldState.includes(x));
    const removed = oldState.filter((x) => !newState.includes(x));
    return {
      added: added,
      removed: removed,
    };
  }

  const changeBugLabels = (
    bugLabels = bug.labels.map((l) => l.name),
    selectedLabel = selectedLabels
  ) => {
    const labels = diff(bugLabels, selectedLabel);
    console.log('changeBugLabels');
    console.log(labels);
    console.log('bugLabelNames');
    console.log(bugLabelNames);
    if (labels.added.length > 0 || labels.removed.length > 0) {
      setLabelMutation({
        variables: {
          input: {
            prefix: bug.id,
            added: labels.added,
            Removed: labels.removed,
          },
        },
        refetchQueries: [
          // TODO: update the cache instead of refetching
          {
            query: GetBugDocument,
            variables: { id: bug.id },
          },
          {
            query: ListLabelsDocument,
          },
        ],
        awaitRefetchQueries: true,
      })
        .then((res) => {
          console.log(res);
          setBugLabelNames(selectedLabels);
        })
        .catch((e) => console.log(e));
    }
  };

  function isActive(key: string) {
    return selectedLabels.includes(key);
  }

  function createNewLabel(name: string) {
    console.log('CREATE NEW LABEL');
    setLabelMutation({
      variables: {
        input: {
          prefix: bug.id,
          added: [name],
        },
      },
      refetchQueries: [
        // TODO: update the cache instead of refetching
        {
          query: GetBugDocument,
          variables: { id: bug.id },
        },
        {
          query: ListLabelsDocument,
        },
      ],
      awaitRefetchQueries: true,
    })
      .then((res) => {
        console.log(res);

        const tmp = selectedLabels.concat([name]);
        console.log(tmp);
        console.log('tmp');
        setSelectedLabels(tmp);
        setBugLabelNames(bugLabelNames.concat([name]));

        changeBugLabels(bugLabelNames.concat([name]), tmp);
      })
      .catch((e) => console.log('createnewLabelError' + e));
  }

  let labels: any = [];
  if (
    labelsData?.repository &&
    labelsData.repository.validLabels &&
    labelsData.repository.validLabels.nodes
  ) {
    labels = labelsData.repository.validLabels.nodes.map((node) => [
      node.name,
      node.name,
      node.color,
    ]);
  }

  return (
    <FilterDropdown
      onClose={changeBugLabels}
      itemActive={isActive}
      toggleLabel={toggleLabel}
      dropdown={labels}
      onNewItem={createNewLabel}
      hasFilter
    >
      Labels
    </FilterDropdown>
  );
}

export default LabelMenu;
