import { gql, useMutation } from '@apollo/client';
import React, { FormEvent } from 'react';

import Paper from '@material-ui/core/Paper/Paper';
import TextField from '@material-ui/core/TextField/TextField';
import { fade, makeStyles, Theme } from '@material-ui/core/styles';

import GBButton from '../../components/Button/GBButton';

/**
 * Styles
 */
const useStyles = makeStyles((theme) => ({
  main: {
    maxWidth: 800,
    margin: 'auto',
    marginTop: theme.spacing(4),
    marginBottom: theme.spacing(4),
    padding: theme.spacing(2),
    overflow: 'hidden',
  },
  titleInput: {
    borderRadius: theme.shape.borderRadius,
    borderColor: fade(theme.palette.primary.main, 0.2),
    borderStyle: 'solid',
    borderWidth: '1px',
    backgroundColor: fade(theme.palette.primary.main, 0.05),
    padding: theme.spacing(0, 0),
    transition: theme.transitions.create([
      'width',
      'borderColor',
      'backgroundColor',
    ]),
  },
  form: {
    display: 'flex',
    flexDirection: 'row',
    flexWrap: 'wrap',
    justifyContent: 'flex-end',
  },
}));

const NEW_BUG = gql`
  mutation NewBug($input: NewBugInput) {
    newBug(input: $input) {
      title
      message
    }
  }
`;

/**
 * Form to create a new issue
 */
function NewPage() {
  const classes = useStyles({ searching: false });
  const [newBugInput] = useMutation(NEW_BUG);

  function submitNewIssue(e: FormEvent) {
    e.preventDefault();
    // TODO Call API
  }

  return (
    <Paper className={classes.main}>
      <form className={classes.form} onSubmit={submitNewIssue}>
        <TextField
          label="Title"
          className={classes.titleInput}
          variant="outlined"
          fullWidth
          margin="dense"
        />
        <GBButton to="/" text="Submit new issue" />
      </form>
    </Paper>
  );
}

export default NewPage;
