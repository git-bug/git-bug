import React from 'react';

import { Card, Divider, Link, Typography } from '@material-ui/core';
import CircularProgress from '@material-ui/core/CircularProgress';
import { makeStyles } from '@material-ui/core/styles';

import Date from '../../components/Date';

import { useGetBugsByUserQuery } from './GetBugsByUser.generated';

const useStyles = makeStyles((theme) => ({
  main: {
    ...theme.typography.body2,
  },
  bugLink: {
    ...theme.typography.button,
  },
  cards: {
    backgroundColor: theme.palette.background.default,
    color: theme.palette.info.contrastText,
    padding: theme.spacing(1),
    margin: theme.spacing(1),
  },
}));

type Props = {
  humanId: string;
};

function BugList({ humanId }: Props) {
  const classes = useStyles();
  const { loading, error, data } = useGetBugsByUserQuery({
    variables: {
      query: 'author:' + humanId + ' sort:creation',
    },
  });

  if (loading) return <CircularProgress />;
  if (error) return <p>Error: {error}</p>;
  const bugs = data?.repository?.allBugs.nodes;

  return (
    <div className={classes.main}>
      {bugs?.map((bug, index) => {
        return (
          <Card className={classes.cards} key={index}>
            <Typography variant="overline" component="h2">
              <Link
                className={classes.bugLink}
                href={'/bug/' + bug.id}
                color={'inherit'}
              >
                {bug.title}
              </Link>
            </Typography>
            <Divider />
            <Typography variant="subtitle2">
              Created&nbsp;
              <Date date={bug.createdAt} />
            </Typography>
            <Typography variant="subtitle2">
              Last edited&nbsp;
              <Date date={bug.createdAt} />
            </Typography>
          </Card>
        );
      })}
      {bugs?.length === 0 && <p>No authored bugs by this user found.</p>}
    </div>
  );
}

export default BugList;
