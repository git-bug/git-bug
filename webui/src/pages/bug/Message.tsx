import React from 'react';

import IconButton from '@material-ui/core/IconButton';
import Paper from '@material-ui/core/Paper';
import Tooltip from '@material-ui/core/Tooltip/Tooltip';
import { makeStyles } from '@material-ui/core/styles';
import EditIcon from '@material-ui/icons/Edit';

import Author, { Avatar } from 'src/components/Author';
import Content from 'src/components/Content';
import Date from 'src/components/Date';

import { AddCommentFragment } from './MessageCommentFragment.generated';
import { CreateFragment } from './MessageCreateFragment.generated';

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
  editButton: {
    color: theme.palette.info.contrastText,
    padding: '0rem',
    fontSize: '0.75rem',
    '&:hover': {
      backgroundColor: 'inherit',
    },
  },
}));

type Props = {
  op: AddCommentFragment | CreateFragment;
};

function Message({ op }: Props) {
  const classes = useStyles();

  const editComment = (id: String) => {
    console.log(id);
  };

  return (
    <article className={classes.container}>
      <Avatar author={op.author} className={classes.avatar} />
      <Paper elevation={1} className={classes.bubble}>
        <header className={classes.header}>
          <div className={classes.title}>
            <Author className={classes.author} author={op.author} />
            <span> commented </span>
            <Date date={op.createdAt} />
          </div>
          {op.edited && <div className={classes.tag}>Edited</div>}
          <Tooltip title="Edit Message" placement="top" arrow={true}>
            <IconButton
              disableRipple
              className={classes.editButton}
              aria-label="edit message"
              onClick={() => editComment(op.id)}
            >
              <EditIcon />
            </IconButton>
          </Tooltip>
        </header>
        <section className={classes.body}>
          <Content markdown={op.message} />
        </section>
      </Paper>
    </article>
  );
}

export default Message;
