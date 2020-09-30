import React, { useState, useRef } from 'react';

import Button from '@material-ui/core/Button';
import Paper from '@material-ui/core/Paper';
import Tab from '@material-ui/core/Tab';
import Tabs from '@material-ui/core/Tabs';
import TextField from '@material-ui/core/TextField';
import { makeStyles, Theme } from '@material-ui/core/styles';

import Content from 'src/components/Content';

import { useAddCommentMutation } from './CommentForm.generated';
import { TimelineDocument } from './TimelineQuery.generated';

type StyleProps = { loading: boolean };
const useStyles = makeStyles<Theme, StyleProps>((theme) => ({
  container: {
    margin: theme.spacing(2, 0),
    padding: theme.spacing(0, 2, 2, 2),
  },
  textarea: {},
  tabContent: {
    margin: theme.spacing(2, 0),
  },
  preview: {
    borderBottom: `solid 3px ${theme.palette.grey['200']}`,
    minHeight: '5rem',
  },
  actions: {
    display: 'flex',
    justifyContent: 'flex-end',
  },
}));

type TabPanelProps = {
  children: React.ReactNode;
  value: number;
  index: number;
} & React.HTMLProps<HTMLDivElement>;
function TabPanel({ children, value, index, ...props }: TabPanelProps) {
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`editor-tabpanel-${index}`}
      aria-labelledby={`editor-tab-${index}`}
      {...props}
    >
      {value === index && children}
    </div>
  );
}

const a11yProps = (index: number) => ({
  id: `editor-tab-${index}`,
  'aria-controls': `editor-tabpanel-${index}`,
});

type Props = {
  bugId: string;
};

function CommentForm({ bugId }: Props) {
  const [addComment, { loading }] = useAddCommentMutation();
  const [input, setInput] = useState<string>('');
  const [tab, setTab] = useState(0);
  const classes = useStyles({ loading });
  const form = useRef<HTMLFormElement>(null);

  const submit = () => {
    addComment({
      variables: {
        input: {
          prefix: bugId,
          message: input,
        },
      },
      refetchQueries: [
        // TODO: update the cache instead of refetching
        {
          query: TimelineDocument,
          variables: {
            id: bugId,
            first: 100,
          },
        },
      ],
      awaitRefetchQueries: true,
    }).then(() => setInput(''));
  };

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    submit();
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLElement>) => {
    // Submit on cmd/ctrl+enter
    if ((e.metaKey || e.altKey) && e.keyCode === 13) {
      submit();
    }
  };

  return (
    <Paper className={classes.container}>
      <form onSubmit={handleSubmit} ref={form}>
        <Tabs value={tab} onChange={(_, t) => setTab(t)}>
          <Tab label="Write" {...a11yProps(0)} />
          <Tab label="Preview" {...a11yProps(1)} />
        </Tabs>
        <div className={classes.tabContent}>
          <TabPanel value={tab} index={0}>
            <TextField
              onKeyDown={handleKeyDown}
              fullWidth
              label="Comment"
              placeholder="Leave a comment"
              className={classes.textarea}
              multiline
              value={input}
              variant="filled"
              rows="4" // TODO: rowsMin support
              onChange={(e: any) => setInput(e.target.value)}
              disabled={loading}
            />
          </TabPanel>
          <TabPanel value={tab} index={1} className={classes.preview}>
            <Content markdown={input} />
          </TabPanel>
        </div>
        <div className={classes.actions}>
          <Button
            variant="contained"
            color="primary"
            type="submit"
            disabled={loading}
          >
            Comment
          </Button>
        </div>
      </form>
    </Paper>
  );
}

export default CommentForm;
