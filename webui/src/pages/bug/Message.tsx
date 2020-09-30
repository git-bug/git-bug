import React from 'react';

import Paper from '@material-ui/core/Paper';
import { makeStyles } from '@material-ui/core/styles';

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
    color: '#444',
    padding: '0.5rem 1rem',
    borderBottom: '1px solid #ddd',
    display: 'flex',
    backgroundColor: '#e2f1ff',
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
    padding: '0 1rem',
  },
}));

type Props = {
  op: AddCommentFragment | CreateFragment;
};

function Message({ op }: Props) {
  const classes = useStyles();
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
        </header>
        <section className={classes.body}>
          <Content markdown={op.message} />
        </section>
      </Paper>
    </article>
  );
}

export default Message;
