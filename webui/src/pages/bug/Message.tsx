import React, { useState } from 'react';

import IconButton from '@material-ui/core/IconButton';
import Paper from '@material-ui/core/Paper';
import Tooltip from '@material-ui/core/Tooltip/Tooltip';
import { makeStyles } from '@material-ui/core/styles';
import EditIcon from '@material-ui/icons/Edit';
import HistoryIcon from '@material-ui/icons/History';

import Author, { Avatar } from 'src/components/Author';
import Content from 'src/components/Content';
import Date from 'src/components/Date';
import IfLoggedIn from 'src/components/IfLoggedIn/IfLoggedIn';

import { BugFragment } from './Bug.generated';
import EditCommentForm from './EditCommentForm';
import { AddCommentFragment } from './MessageCommentFragment.generated';
import { CreateFragment } from './MessageCreateFragment.generated';
import MessageHistoryDialog from './MessageHistoryDialog';

const useStyles = makeStyles((theme) => ({
  author: {
    fontWeight: 'bold',
  },
  container: {
    display: 'flex',
  },
  avatar: {
    marginTop: 2,
  },
  bubble: {
    flex: 1,
    marginLeft: theme.spacing(1),
    minWidth: 0,
  },
  header: {
    ...theme.typography.body1,
    padding: '0.5rem 1rem',
    borderBottom: `1px solid ${theme.palette.divider}`,
    display: 'flex',
    borderTopRightRadius: theme.shape.borderRadius,
    borderTopLeftRadius: theme.shape.borderRadius,
    backgroundColor: theme.palette.info.main,
    color: theme.palette.info.contrastText,
  },
  title: {
    flex: 1,
  },
  tag: {
    ...theme.typography.button,
    color: '#888',
    border: '#ddd solid 1px',
    padding: '0 0.5rem',
    fontSize: '0.75rem',
    borderRadius: 2,
    marginLeft: '0.5rem',
  },
  body: {
    ...theme.typography.body2,
    padding: '0.5rem',
  },
  headerActions: {
    color: theme.palette.info.contrastText,
    padding: '0rem',
    marginLeft: theme.spacing(1),
    fontSize: '0.75rem',
    '&:hover': {
      backgroundColor: 'inherit',
    },
  },
}));

//TODO Move this button and menu in separate component directory
//TODO fix failing pipeline due to eslint error
type HistBtnProps = {
  bugId: string;
  commentId: string;
};
function HistoryMenuToggleButton({ bugId, commentId }: HistBtnProps) {
  const classes = useStyles();
  const [open, setOpen] = React.useState(false);

  const handleClickOpen = () => {
    setOpen(true);
  };

  const handleClose = () => {
    setOpen(false);
  };

  return (
    <div>
      <IconButton
        aria-label="more"
        aria-controls="long-menu"
        aria-haspopup="true"
        onClick={handleClickOpen}
        className={classes.headerActions}
      >
        <HistoryIcon />
      </IconButton>
      {
        // Render CustomizedDialogs on open to prevent fetching the history
        // before opening the history menu.
        open && (
          <MessageHistoryDialog
            bugId={bugId}
            commentId={commentId}
            open={open}
            onClose={handleClose}
          />
        )
      }
    </div>
  );
}

type Props = {
  bug: BugFragment;
  op: AddCommentFragment | CreateFragment;
};
function Message({ bug, op }: Props) {
  const classes = useStyles();
  const [editMode, switchToEditMode] = useState(false);
  const [comment, setComment] = useState(op);

  const editComment = (id: String) => {
    switchToEditMode(true);
  };

  function readMessageView() {
    return (
      <Paper elevation={1} className={classes.bubble}>
        <header className={classes.header}>
          <div className={classes.title}>
            <Author className={classes.author} author={comment.author} />
            <span> commented </span>
            <Date date={comment.createdAt} />
          </div>
          {comment.edited && (
            <HistoryMenuToggleButton bugId={bug.id} commentId={comment.id} />
          )}
          <IfLoggedIn>
            {() => (
              <Tooltip title="Edit Message" placement="top" arrow={true}>
                <IconButton
                  disableRipple
                  className={classes.headerActions}
                  aria-label="edit message"
                  onClick={() => editComment(comment.id)}
                >
                  <EditIcon />
                </IconButton>
              </Tooltip>
            )}
          </IfLoggedIn>
        </header>
        <section className={classes.body}>
          <Content markdown={comment.message} />
        </section>
      </Paper>
    );
  }

  function editMessageView() {
    const cancelEdition = () => {
      switchToEditMode(false);
    };

    const onPostSubmit = (comment: AddCommentFragment | CreateFragment) => {
      setComment(comment);
      switchToEditMode(false);
    };

    return (
      <div className={classes.bubble}>
        <EditCommentForm
          bug={bug}
          onCancel={cancelEdition}
          onPostSubmit={onPostSubmit}
          comment={comment}
        />
      </div>
    );
  }

  return (
    <article className={classes.container}>
      <Avatar author={comment.author} className={classes.avatar} />
      {editMode ? editMessageView() : readMessageView()}
    </article>
  );
}

export default Message;
