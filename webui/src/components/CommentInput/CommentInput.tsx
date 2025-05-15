import { Typography } from '@mui/material';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import TextField from '@mui/material/TextField';
import makeStyles from '@mui/styles/makeStyles';
import * as React from 'react';
import { useState, useEffect } from 'react';

import Content from '../Content';

/**
 * Styles
 */
const useStyles = makeStyles((theme) => ({
  container: {
    margin: theme.spacing(2, 0),
    padding: theme.spacing(0, 2, 2, 2),
  },
  textarea: {
    '& textarea.MuiInputBase-input': {
      resize: 'vertical',
    },
  },
  tabContent: {
    margin: theme.spacing(2, 0),
  },
  preview: {
    overflow: 'auto',
    borderBottom: `solid 3px ${theme.palette.grey['200']}`,
    minHeight: '5rem',
  },
  previewPlaceholder: {
    color: theme.palette.text.secondary,
    fontStyle: 'italic',
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
  inputProps?: any;
  inputText?: string;
  loading: boolean;
  onChange: (comment: string) => void;
};

/**
 * Component for issue comment input
 *
 * @param inputProps Reset input value
 * @param loading Disable input when component not ready yet
 * @param onChange Callback to return input value changes
 */
function CommentInput({ inputProps, inputText, loading, onChange }: Props) {
  const [input, setInput] = useState<string>(inputText ? inputText : '');
  const [tab, setTab] = useState(0);
  const classes = useStyles();

  useEffect(() => {
    if (inputProps) setInput(inputProps.value);
  }, [inputProps]);

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
            minRows={4}
            onChange={(e: any) => setInput(e.target.value)}
            disabled={loading}
          />
        </TabPanel>
        <TabPanel value={tab} index={1} className={classes.preview}>
          {input !== '' ? (
            <Content markdown={input} />
          ) : (
            <Typography className={classes.previewPlaceholder}>
              Nothing to preview.
            </Typography>
          )}
        </TabPanel>
      </div>
    </div>
  );
}

export default CommentInput;
