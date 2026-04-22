import { Tab, Tabs } from '@mui/material';
import makeStyles from '@mui/styles/makeStyles';
import { useState } from 'react';
import { useLocation } from 'react-router';

import BugTitleForm from 'src/components/BugTitleForm/BugTitleForm';
import IfLoggedIn from 'src/components/IfLoggedIn/IfLoggedIn';
import Label from 'src/components/Label';

import { BugFragment } from './Bug.generated';
import Checks from './Checks';
import CommentForm from './CommentForm';
import PrCommits from './PrCommits';
import PrDiff from './PrDiff';
import PrInfo from './PrInfo';
import Reviews from './Reviews';
import TimelineQuery from './TimelineQuery';
import LabelMenu from './labels/LabelMenu';

// Extract owner/repo from /r/<urlencoded-name>/bug/... — the Bug component
// renders inside the route subtree so useParams doesn't expose the outer
// :repoName directly. Matching the pathname keeps us decoupled from
// nested-routing details.
function repoNameFromPath(pathname: string): string | null {
  const m = pathname.match(/^\/r\/([^/]+)/);
  if (!m) return null;
  try {
    return decodeURIComponent(m[1]);
  } catch {
    return null;
  }
}

/**
 * Css in JS Styles
 */
const useStyles = makeStyles((theme) => ({
  main: {
    maxWidth: 1000,
    margin: 'auto',
    marginTop: theme.spacing(4),
  },
  // Wide layout for the Files changed tab: code beats prose for real
  // estate — we drop the narrow reading width and hide the label
  // sidebar so the diff can use the full viewport.
  mainWide: {
    maxWidth: 'none',
    margin: 0,
    marginTop: theme.spacing(4),
    paddingLeft: theme.spacing(2),
    paddingRight: theme.spacing(2),
  },
  header: {
    marginRight: theme.spacing(2),
    marginLeft: theme.spacing(3) + 40,
  },
  title: {
    ...theme.typography.h5,
  },
  id: {
    ...theme.typography.subtitle1,
    marginLeft: theme.spacing(1),
  },
  container: {
    display: 'flex',
    marginBottom: theme.spacing(1),
    marginRight: theme.spacing(2),
    marginLeft: theme.spacing(2),
  },
  timeline: {
    flex: 1,
    marginTop: theme.spacing(2),
    marginRight: theme.spacing(2),
    minWidth: 400,
  },
  rightSidebar: {
    marginTop: theme.spacing(2),
    flex: '0 0 200px',
  },
  rightSidebarTitle: {
    fontWeight: 'bold',
  },
  labelList: {
    listStyle: 'none',
    padding: 0,
    margin: 0,
    display: 'flex',
    flexDirection: 'row',
    flexWrap: 'wrap',
  },
  label: {
    marginTop: theme.spacing(0.1),
    marginBottom: theme.spacing(0.1),
    marginLeft: theme.spacing(0.25),
    marginRight: theme.spacing(0.25),
  },
  noLabel: {
    ...theme.typography.body2,
  },
  commentForm: {
    marginTop: theme.spacing(2),
    marginLeft: 48,
  },
  tabs: {
    borderBottom: `1px solid ${theme.palette.divider}`,
    marginBottom: theme.spacing(2),
    minHeight: 36,
  },
  tab: {
    minHeight: 36,
    textTransform: 'none',
    fontSize: '0.9rem',
  },
}));

type TabKey = 'conversation' | 'commits' | 'files';

type Props = {
  bug: BugFragment;
};

function Bug({ bug }: Props) {
  const classes = useStyles();
  const location = useLocation();
  const repoName = repoNameFromPath(location.pathname);
  const [tab, setTab] = useState<TabKey>('conversation');

  const isPr = bug.kind === 'PR';
  const hasRefs = !!(bug.baseRef && bug.headRef);

  const isFilesTab = tab === 'files' && isPr && hasRefs && !!repoName;
  // On the Files tab we swap to a full-width layout and drop the
  // label sidebar — unified diffs are useless when clipped to 800px.
  return (
    <main className={isFilesTab ? classes.mainWide : classes.main}>
      <div className={classes.header}>
        <BugTitleForm bug={bug} />
      </div>
      <div className={classes.container}>
        <div className={classes.timeline}>
          <PrInfo bug={bug} />
          <Checks bug={bug} repoName={repoName} />
          {isPr && hasRefs && repoName && (
            <Tabs
              value={tab}
              onChange={(_, v) => setTab(v)}
              className={classes.tabs}
              variant="standard"
            >
              <Tab
                value="conversation"
                label="Conversation"
                className={classes.tab}
              />
              <Tab value="commits" label="Commits" className={classes.tab} />
              <Tab value="files" label="Files changed" className={classes.tab} />
            </Tabs>
          )}
          {tab === 'conversation' && (
            <>
              <TimelineQuery bug={bug} />
              <Reviews bug={bug} />
              <IfLoggedIn>
                {() => (
                  <div className={classes.commentForm}>
                    <CommentForm bug={bug} />
                  </div>
                )}
              </IfLoggedIn>
            </>
          )}
          {tab === 'commits' && isPr && hasRefs && repoName && (
            <PrCommits
              repoName={repoName}
              baseRef={bug.baseRef!}
              headRef={bug.headCommit || bug.headRef!}
              originUrl={bug.originUrl}
            />
          )}
          {tab === 'files' && isPr && hasRefs && repoName && (
            <PrDiff
              repoName={repoName}
              baseRef={bug.baseRef!}
              headRef={bug.headCommit || bug.headRef!}
              originUrl={bug.originUrl}
            />
          )}
        </div>
        {!isFilesTab && (
        <div className={classes.rightSidebar}>
          <span className={classes.rightSidebarTitle}>
            <LabelMenu bug={bug} />
          </span>
          <ul className={classes.labelList}>
            {bug.labels.length === 0 && (
              <span className={classes.noLabel}>None yet</span>
            )}
            {bug.labels.map((l) => (
              <li className={classes.label} key={l.name}>
                <Label label={l} key={l.name} maxWidth="25ch" />
              </li>
            ))}
          </ul>
        </div>
        )}
      </div>
    </main>
  );
}

export default Bug;
