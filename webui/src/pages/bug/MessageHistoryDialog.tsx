import moment from 'moment';
import React from 'react';
import Moment from 'react-moment';

import MuiAccordion from '@material-ui/core/Accordion';
import MuiAccordionDetails from '@material-ui/core/AccordionDetails';
import MuiAccordionSummary from '@material-ui/core/AccordionSummary';
import CircularProgress from '@material-ui/core/CircularProgress';
import Dialog from '@material-ui/core/Dialog';
import MuiDialogContent from '@material-ui/core/DialogContent';
import MuiDialogTitle from '@material-ui/core/DialogTitle';
import Grid from '@material-ui/core/Grid';
import IconButton from '@material-ui/core/IconButton';
import Tooltip from '@material-ui/core/Tooltip/Tooltip';
import Typography from '@material-ui/core/Typography';
import {
  createStyles,
  Theme,
  withStyles,
  WithStyles,
} from '@material-ui/core/styles';
import CloseIcon from '@material-ui/icons/Close';
import ExpandMoreIcon from '@material-ui/icons/ExpandMore';

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
    <MuiDialogTitle disableTypography className={classes.root} {...other}>
      <Typography variant="h6">{children}</Typography>
      {onClose ? (
        <IconButton
          aria-label="close"
          className={classes.closeButton}
          onClick={onClose}
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
          <Grid container justify="center">
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
          <p>Error: {error}</p>
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

  const handleChange = (panel: string) => (
    event: React.ChangeEvent<{}>,
    newExpanded: boolean
  ) => {
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
            <AccordionDetails>{edit.message}</AccordionDetails>
          </Accordion>
        ))}
      </DialogContent>
    </Dialog>
  );
}

export default MessageHistoryDialog;
