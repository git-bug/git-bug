import { gql, useQuery } from '@apollo/client';
import ArrowDropDownIcon from '@mui/icons-material/ArrowDropDown';
import { Button, Menu, MenuItem } from '@mui/material';
import makeStyles from '@mui/styles/makeStyles';
import { useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router';

const REPO_NAMES_QUERY = gql`
  query RepoNames {
    repositories {
      totalCount
      nodes {
        name
      }
    }
  }
`;

interface RepoNamesData {
  repositories: { totalCount: number; nodes: Array<{ name: string | null }> };
}

const useStyles = makeStyles((theme) => ({
  button: {
    color: theme.palette.primary.contrastText,
    textTransform: 'none',
    marginLeft: theme.spacing(1),
  },
  menuItem: {
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    fontSize: '0.9rem',
  },
  reposLink: {
    color: theme.palette.primary.contrastText,
    textDecoration: 'none',
    marginLeft: theme.spacing(2),
  },
}));

/** RepoPicker shows the currently-active repo name (from the URL) with a
 *  dropdown to switch to any of the other registered repos. In single-repo
 *  mode (only the __default repo registered), the picker stays hidden and
 *  the header shows a static title instead. */
export default function RepoPicker() {
  const classes = useStyles();
  const { data } = useQuery<RepoNamesData>(REPO_NAMES_QUERY);
  const { repoName } = useParams<{ repoName: string }>();
  const navigate = useNavigate();
  const [anchor, setAnchor] = useState<null | HTMLElement>(null);

  const named = useMemo(() => {
    const all = data?.repositories.nodes ?? [];
    return all
      .map((n) => n.name)
      .filter((n): n is string => !!n && n !== '__default')
      .sort((a, b) => a.localeCompare(b));
  }, [data]);

  if (named.length === 0) return null; // single-repo mode

  const label = repoName ?? 'pick a repo';

  return (
    <>
      <Link to="/" className={classes.reposLink}>
        Repositories
      </Link>
      <Button
        className={classes.button}
        onClick={(e) => setAnchor(e.currentTarget)}
        endIcon={<ArrowDropDownIcon />}
      >
        {label}
      </Button>
      <Menu
        anchorEl={anchor}
        open={Boolean(anchor)}
        onClose={() => setAnchor(null)}
      >
        {named.map((n) => (
          <MenuItem
            key={n}
            className={classes.menuItem}
            selected={n === repoName}
            onClick={() => {
              setAnchor(null);
              navigate(
                `/r/${encodeURIComponent(n)}/?q=kind:issue status:open status:draft`
              );
            }}
          >
            {n}
          </MenuItem>
        ))}
      </Menu>
    </>
  );
}
