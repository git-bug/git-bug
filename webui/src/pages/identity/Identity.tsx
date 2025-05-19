import InfoIcon from '@mui/icons-material/Info';
import MailOutlineIcon from '@mui/icons-material/MailOutline';
import { Link, Paper, Typography } from '@mui/material';
import Avatar from '@mui/material/Avatar';
import CircularProgress from '@mui/material/CircularProgress';
import Grid from '@mui/material/Grid';
import makeStyles from '@mui/styles/makeStyles';
import { Link as RouterLink } from 'react-router';

import { IdentityFragment } from '../../components/Identity/IdentityFragment.generated';

import { useGetUserStatisticQuery } from './GetUserStatistic.generated';

const useStyles = makeStyles((theme) => ({
  main: {
    maxWidth: `min(${theme.breakpoints.values.md}px, calc(100% - ${theme.spacing(8)}))`,
    margin: theme.spacing(4, 'auto'),
    [theme.breakpoints.down('md')]: {
      maxWidth: '100%',
      margin: 0,
    },
  },
  content: {
    padding: theme.spacing(0.5, 2, 2, 2),
    wordWrap: 'break-word',
  },
  large: {
    minWidth: 200,
    minHeight: 200,
    margin: 'auto',
    maxWidth: '100%',
    maxHeight: '100%',
  },
  heading: {
    marginTop: theme.spacing(3),
  },
  header: {
    ...theme.typography.h4,
    wordBreak: 'break-word',
  },
  infoIcon: {
    verticalAlign: 'bottom',
  },
}));

type Props = {
  identity: IdentityFragment;
};
const Identity = ({ identity }: Props) => {
  const classes = useStyles();
  const user = identity;

  const { loading, error, data } = useGetUserStatisticQuery({
    variables: {
      authorQuery: 'author:' + user?.id,
      participantQuery: 'participant:' + user?.id,
      actionQuery: 'actor:' + user?.id,
    },
  });

  if (loading) return <CircularProgress />;
  if (error) return <p>Error: {error.message}</p>;
  const statistic = data?.repository;
  const authoredCount = statistic?.authored?.totalCount;
  const participatedCount = statistic?.participated?.totalCount;
  const actionCount = statistic?.actions?.totalCount;

  return (
    <main className={classes.main}>
      <Paper elevation={3} className={classes.content}>
        <Grid spacing={2} container direction="row">
          <Grid xs={12} md={4} className={classes.heading} item>
            <Avatar
              src={user?.avatarUrl ? user.avatarUrl : undefined}
              className={classes.large}
            >
              {user?.displayName.charAt(0).toUpperCase()}
            </Avatar>
          </Grid>
          <Grid xs={12} md={4} item>
            <section>
              <h1 className={classes.header}>{user?.name}</h1>
              <Typography variant="subtitle1">
                Name: {user?.displayName ? user?.displayName : '---'}
              </Typography>
              <Typography variant="subtitle1">
                Id (truncated): {user?.humanId ? user?.humanId : '---'}
                <InfoIcon
                  titleAccess={user?.id ? user?.id : '---'}
                  className={classes.infoIcon}
                />
              </Typography>
              {user?.email && (
                <Typography
                  variant="subtitle1"
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    flexWrap: 'wrap',
                  }}
                >
                  <MailOutlineIcon />
                  <Link
                    href={'mailto:' + user?.email}
                    color={'inherit'}
                    underline="hover"
                  >
                    {user?.email}
                  </Link>
                </Typography>
              )}
            </section>
          </Grid>
          <Grid xs={12} md={4} item>
            <section>
              <h1 className={classes.header}>Statistics</h1>
              <Link
                component={RouterLink}
                to={`/?q=author%3A${user?.id}+sort%3Acreation`}
                color={'inherit'}
                underline="hover"
              >
                <Typography variant="subtitle1">
                  Created {authoredCount} bugs.
                </Typography>
              </Link>
              <Link
                component={RouterLink}
                to={`/?q=participant%3A${user?.id}+sort%3Acreation`}
                color={'inherit'}
                underline="hover"
              >
                <Typography variant="subtitle1">
                  Participated to {participatedCount} bugs.
                </Typography>
              </Link>
              <Link
                component={RouterLink}
                to={`/?q=actor%3A${user?.id}+sort%3Acreation`}
                color={'inherit'}
                underline="hover"
              >
                <Typography variant="subtitle1">
                  Interacted with {actionCount} bugs.
                </Typography>
              </Link>
            </section>
          </Grid>
        </Grid>
      </Paper>
    </main>
  );
};

export default Identity;
