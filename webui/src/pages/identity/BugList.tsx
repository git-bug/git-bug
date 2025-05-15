import { Card, Divider, Link, Typography } from '@mui/material';
import CircularProgress from '@mui/material/CircularProgress';
import makeStyles from '@mui/styles/makeStyles';

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
  id: string;
};

function BugList({ id }: Props) {
  const classes = useStyles();
  const { loading, error, data } = useGetBugsByUserQuery({
    variables: {
      query: 'author:' + id + ' sort:creation',
    },
  });

  if (loading) return <CircularProgress />;
  if (error) return <p>Error: {error.message}</p>;
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
                underline="hover"
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
