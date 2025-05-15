import CloseIcon from '@mui/icons-material/Close';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import MuiAccordion from '@mui/material/Accordion';
import MuiAccordionDetails from '@mui/material/AccordionDetails';
import MuiAccordionSummary from '@mui/material/AccordionSummary';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import MuiDialogContent from '@mui/material/DialogContent';
import MuiDialogTitle from '@mui/material/DialogTitle';
import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip/Tooltip';
import Typography from '@mui/material/Typography';
import { Theme } from '@mui/material/styles';
import { WithStyles } from '@mui/styles';
import createStyles from '@mui/styles/createStyles';
import withStyles from '@mui/styles/withStyles';
import moment from 'moment';
import * as React from 'react';

import Content from '../../components/Content';
import Moment from '../../components/Moment';

import { AddCommentFragment } from './MessageCommentFragment.generated';
import { CreateFragment } from './MessageCreateFragment.generated';
import { useMessageHistoryQuery } from './MessageHistory.generated';

const styles = (theme: Theme) =>
  createStyles({
    root: {
      margin: 0,
      padding: theme.spacing(2),
    },
    closeButton: {
      position: 'absolute',
      right: theme.spacing(1),
      top: theme.spacing(1),
    },
  });

export interface DialogTitleProps extends WithStyles<typeof styles> {
  id: string;
  children: React.ReactNode;
  onClose: () => void;
}

const DialogTitle = withStyles(styles)((props: DialogTitleProps) => {
  const { children, classes, onClose, ...other } = props;
  return (
    <MuiDialogTitle className={classes.root} {...other}>
      <Typography variant="h6">{children}</Typography>
      {onClose ? (
        <IconButton
          aria-label="close"
          className={classes.closeButton}
          onClick={onClose}
          size="large"
        >
          <CloseIcon />
        </IconButton>
      ) : null}
    </MuiDialogTitle>
  );
});

const DialogContent = withStyles((theme: Theme) => ({
  root: {
    padding: theme.spacing(2),
  },
}))(MuiDialogContent);

const Accordion = withStyles({
  root: {
    border: '1px solid rgba(0, 0, 0, .125)',
    boxShadow: 'none',
    '&:not(:last-child)': {
      borderBottom: 0,
    },
    '&:before': {
      display: 'none',
    },
    '&$expanded': {
      margin: 'auto',
    },
  },
  expanded: {},
})(MuiAccordion);

const AccordionSummary = withStyles((theme) => ({
  root: {
    backgroundColor: theme.palette.primary.light,
    borderBottomWidth: '1px',
    borderBottomStyle: 'solid',
    borderBottomColor: theme.palette.divider,
    marginBottom: -1,
    minHeight: 56,
    '&$expanded': {
      minHeight: 56,
    },
  },
  content: {
    '&$expanded': {
      margin: '12px 0',
    },
  },
  expanded: {},
}))(MuiAccordionSummary);

const AccordionDetails = withStyles((theme) => ({
  root: {
    display: 'block',
    overflow: 'auto',
    padding: theme.spacing(2),
  },
}))(MuiAccordionDetails);

type Props = {
  bugId: string;
  commentId: string;
  open: boolean;
  onClose: () => void;
};
function MessageHistoryDialog({ bugId, commentId, open, onClose }: Props) {
  const [expanded, setExpanded] = React.useState<string | false>('panel0');

  const { loading, error, data } = useMessageHistoryQuery({
    variables: { bugIdPrefix: bugId },
  });
  if (loading) {
    return (
      <Dialog
        onClose={onClose}
        aria-labelledby="customized-dialog-title"
        open={open}
        fullWidth
        maxWidth="sm"
      >
        <DialogTitle id="customized-dialog-title" onClose={onClose}>
          Loading...
        </DialogTitle>
        <DialogContent dividers>
          <Grid container justifyContent="center">
            <CircularProgress />
          </Grid>
        </DialogContent>
      </Dialog>
    );
  }
  if (error) {
    return (
      <Dialog
        onClose={onClose}
        aria-labelledby="customized-dialog-title"
        open={open}
        fullWidth
        maxWidth="sm"
      >
        <DialogTitle id="customized-dialog-title" onClose={onClose}>
          Something went wrong...
        </DialogTitle>
        <DialogContent dividers>
          <p>Error: {error.message}</p>
        </DialogContent>
      </Dialog>
    );
  }

  const comments = data?.repository?.bug?.timeline.comments as (
    | AddCommentFragment
    | CreateFragment
  )[];
  // NOTE Searching for the changed comment could be dropped if GraphQL get
  // filter by id argument for timelineitems
  const comment = comments.find((elem) => elem.id === commentId);
  // Sort by most recent edit. Must create a copy of constant history as
  // reverse() modifies inplace.
  const history = comment?.history.slice().reverse();
  const editCount = history?.length === undefined ? 0 : history?.length - 1;

  const handleChange =
    (panel: string) => (event: React.ChangeEvent<{}>, newExpanded: boolean) => {
      setExpanded(newExpanded ? panel : false);
    };

  const getSummary = (index: number, date: Date) => {
    const desc =
      index === editCount ? 'Created ' : `#${editCount - index} • Edited `;
    const mostRecent = index === 0 ? ' (most recent)' : '';
    return (
      <>
        <Tooltip title={moment(date).format('LLLL')}>
          <span>
            {desc}
            <Moment date={date} format="on ll" />
            {mostRecent}
          </span>
        </Tooltip>
      </>
    );
  };

  return (
    <Dialog
      onClose={onClose}
      aria-labelledby="customized-dialog-title"
      open={open}
      fullWidth
      maxWidth="md"
    >
      <DialogTitle id="customized-dialog-title" onClose={onClose}>
        {`Edited ${editCount} ${editCount > 1 ? 'times' : 'time'}.`}
      </DialogTitle>
      <DialogContent dividers>
        {history?.map((edit, index) => (
          <Accordion
            square
            key={index}
            expanded={expanded === 'panel' + index}
            onChange={handleChange('panel' + index)}
          >
            <AccordionSummary
              expandIcon={<ExpandMoreIcon />}
              aria-controls="panel1d-content"
              id="panel1d-header"
            >
              <Typography>{getSummary(index, edit.date)}</Typography>
            </AccordionSummary>
            <AccordionDetails>
              {edit.message !== '' ? (
                <Content markdown={edit.message} />
              ) : (
                <Content markdown="*No description provided.*" />
              )}
            </AccordionDetails>
          </Accordion>
        ))}
      </DialogContent>
    </Dialog>
  );
}

export default MessageHistoryDialog;
