import React, { FormEvent } from 'react';

import Paper from '@material-ui/core/Paper';
import TextField from '@material-ui/core/TextField/TextField';
import { fade, makeStyles, Theme } from '@material-ui/core/styles';

import { useNewBugMutation } from './NewBug.generated';

/**
 * Styles
 */
const useStyles = makeStyles((theme: Theme) => ({
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

/**
 * Form to create a new issue
 */
function NewBugPage() {
  const classes = useStyles();
  let inputField: any;
  const [newBug, { loading, error }] = useNewBugMutation();

  function submitNewIssue(e: FormEvent) {
    e.preventDefault();
    newBug({
      variables: {
        input: {
          title: String(inputField.value),
          message: 'Message', //TODO
        },
      },
    });
    inputField.value = '';
  }

  if (loading) return <div>Loading</div>;
  if (error) return <div>Error</div>;

  return (
    <Paper className={classes.main}>
      <form className={classes.form} onSubmit={submitNewIssue}>
        <TextField
          inputRef={(node) => {
            inputField = node;
          }}
          label="Title"
          className={classes.titleInput}
          variant="outlined"
          fullWidth
          margin="dense"
        />
        <button type="submit">Submit</button>
      </form>
    </Paper>
  );
}

export default NewBugPage;
