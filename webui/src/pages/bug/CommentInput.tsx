import React, { useState, useEffect } from 'react';

import Tab from '@material-ui/core/Tab';
import Tabs from '@material-ui/core/Tabs';
import TextField from '@material-ui/core/TextField';
import { makeStyles } from '@material-ui/core/styles';

import Content from 'src/components/Content';

const useStyles = makeStyles((theme) => ({
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
  loading: boolean;
  onChange: (comment: string) => void;
};

function CommentInput({ loading, onChange }: Props) {
  const [input, setInput] = useState<string>('');
  const [tab, setTab] = useState(0);
  const classes = useStyles();

  useEffect(() => {
    onChange(input);
  }, [input, onChange]);

  return (
    <div>
      <Tabs value={tab} onChange={(_, t) => setTab(t)}>
        <Tab label="Write" {...a11yProps(0)} />
        <Tab label="Preview" {...a11yProps(1)} />
      </Tabs>
      <div className={classes.tabContent}>
        <TabPanel value={tab} index={0}>
          <TextField
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
    </div>
  );
}

export default CommentInput;
