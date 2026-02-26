import CheckIcon from '@mui/icons-material/Check';
import SettingsIcon from '@mui/icons-material/Settings';
import { IconButton } from '@mui/material';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import makeStyles from '@mui/styles/makeStyles';
import * as React from 'react';
import { useRef, useState } from 'react';

import { useListIdentitiesQuery } from '../list/ListIdentities.generated';

import { BugFragment } from './Bug.generated';
import { GetBugDocument } from './BugQuery.generated';
import { useSetAssigneeMutationMutation } from './SetAssigneeMutation.generated';

const useStyles = makeStyles((theme) => ({
  gearBtn: {
    ...theme.typography.body2,
    color: theme.palette.text.secondary,
    padding: theme.spacing(0, 1),
    fontWeight: 400,
    textDecoration: 'none',
    display: 'flex',
    background: 'none',
    border: 'none',
    '&:hover': {
      backgroundColor: 'transparent',
      color: theme.palette.text.primary,
    },
  },
  header: {
    display: 'flex',
    flexDirection: 'row',
  },
  menuRow: {
    display: 'flex',
    alignItems: 'center',
  },
}));

type Props = {
  bug: BugFragment;
};

function AssigneeMenu({ bug }: Props) {
  const classes = useStyles();
  const [open, setOpen] = useState(false);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const { data: identitiesData } = useListIdentitiesQuery();
  const [setAssigneeMutation] = useSetAssigneeMutationMutation();

  const currentAssigneeId = bug.assignee?.humanId || '';

  const setAssignee = (identityPrefix: string) => {
    setAssigneeMutation({
      variables: {
        input: {
          prefix: bug.id,
          assignee: identityPrefix,
        },
      },
      refetchQueries: [
        {
          query: GetBugDocument,
          variables: { id: bug.id },
        },
      ],
      awaitRefetchQueries: true,
    })
      .then(() => setOpen(false))
      .catch((e) => console.log(e));
  };

  const identities =
    identitiesData?.repository?.allIdentities?.nodes || [];

  return (
    <>
      <div className={classes.header}>
        Assignee
        <IconButton
          ref={buttonRef}
          onClick={() => setOpen(!open)}
          className={classes.gearBtn}
          disableRipple
          size="small"
        >
          <SettingsIcon fontSize="small" />
        </IconButton>
      </div>
      <Menu
        anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
        transformOrigin={{ vertical: 'top', horizontal: 'left' }}
        open={open}
        onClose={() => setOpen(false)}
        anchorEl={buttonRef.current}
        PaperProps={{ style: { maxHeight: 200, width: '20ch' } }}
      >
        <MenuItem onClick={() => setAssignee('')}>
          <em>Unassigned</em>
        </MenuItem>
        {identities.map((identity) => (
          <MenuItem
            key={identity.id}
            onClick={() => setAssignee(identity.humanId)}
            selected={identity.humanId === currentAssigneeId}
          >
            <div className={classes.menuRow}>
              {identity.humanId === currentAssigneeId && <CheckIcon fontSize="small" />}
              {identity.displayName}
            </div>
          </MenuItem>
        ))}
      </Menu>
    </>
  );
}

export default AssigneeMenu;
