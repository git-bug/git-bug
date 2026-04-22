import { pipe } from '@arrows/composition';
import BugReportOutlinedIcon from '@mui/icons-material/BugReportOutlined';
import CallMergeIcon from '@mui/icons-material/CallMerge';
import CheckCircleOutline from '@mui/icons-material/CheckCircleOutline';
import ErrorOutline from '@mui/icons-material/ErrorOutline';
import Toolbar from '@mui/material/Toolbar';
import makeStyles from '@mui/styles/makeStyles';
import * as React from 'react';
import { Location, useParams } from 'react-router';

import {
  Filter,
  FilterDropdown,
  FilterProps,
  parse,
  Query,
  stringify,
} from './Filter';
import { useBugCountQuery } from './FilterToolbar.generated';
import { useListIdentitiesQuery } from './ListIdentities.generated';
import { useListLabelsQuery } from './ListLabels.generated';

const useStyles = makeStyles((theme) => ({
  toolbar: {
    backgroundColor: theme.palette.primary.light,
    borderColor: theme.palette.divider,
    borderWidth: '1px 0',
    borderStyle: 'solid',
    margin: theme.spacing(0, -1),
  },
  spacer: {
    flex: 1,
  },
  // Vertical rule separating the kind selector (Issues/PRs) from the
  // status/sort filters. Sits flush with the toolbar's text height.
  separator: {
    width: 1,
    alignSelf: 'stretch',
    backgroundColor: theme.palette.divider,
    margin: theme.spacing(1, 1.5),
  },
}));

function safeDecode(s: string): string {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
}

// This prepends the filter text with a count
type CountingFilterProps = {
  query: string; // the query used as a source to count the number of element
  children: React.ReactNode;
} & FilterProps;

function CountingFilter({ query, children, ...props }: CountingFilterProps) {
  const { repoName } = useParams<{ repoName: string }>();
  const repoRef = repoName ? safeDecode(repoName) : null;
  const { data, loading, error } = useBugCountQuery({
    variables: { query, repoRef },
  });

  let prefix;
  if (loading) prefix = '...';
  else if (error || !data?.repository) prefix = '???';
  // TODO: better prefixes & error handling
  else prefix = data.repository.bugs.totalCount;

  return (
    <Filter {...props}>
      {prefix} {children}
    </Filter>
  );
}

function quoteLabel(value: string) {
  const hasUnquotedColon = RegExp(/^[^'"].*:.*[^'"]$/);
  if (hasUnquotedColon.test(value)) {
    //quote values which contain a colon but are not quoted.
    //E.g. abc:abc becomes "abc:abc"
    return `"${value}"`;
  }
  return value;
}

type Props = {
  query: string;
  queryLocation: (query: string) => Location;
};

function FilterToolbar({ query, queryLocation }: Props) {
  const classes = useStyles();
  const params: Query = parse(query);
  const { repoName } = useParams<{ repoName: string }>();
  const repoRef = repoName ? safeDecode(repoName) : null;
  const { data: identitiesData } = useListIdentitiesQuery({
    variables: { repoRef },
  });
  const { data: labelsData } = useListLabelsQuery({
    variables: { repoRef },
  });

  let identities: any = [];
  let labels: any = [];

  if (
    identitiesData?.repository &&
    identitiesData.repository.allIdentities &&
    identitiesData.repository.allIdentities.nodes
  ) {
    identities = identitiesData.repository.allIdentities.nodes.map((node) => [
      node.name,
      node.name,
    ]);
  }

  if (
    labelsData?.repository &&
    labelsData.repository.validLabels &&
    labelsData.repository.validLabels.nodes
  ) {
    labels = labelsData.repository.validLabels.nodes.map((node) => [
      quoteLabel(node.name),
      node.name,
      node.color,
    ]);
  }

  const hasKey = (key: string): boolean =>
    params[key] && params[key].length > 0;
  const hasValue = (key: string, value: string): boolean =>
    hasKey(key) && params[key].includes(value);
  const containsValue = (key: string, value: string): boolean =>
    hasKey(key) && params[key].indexOf(value) !== -1;
  const loc = pipe(stringify, queryLocation);
  const replaceParam =
    (key: string, value: string) =>
    (params: Query): Query => ({
      ...params,
      [key]: [value],
    });
  // Set a key to a specific OR-list of values, e.g. status=[open,draft].
  // Used for the "open" / "closed" chips which cover multiple underlying
  // statuses (GitHub's Open tab includes drafts; Closed includes merged).
  const replaceParamMulti =
    (key: string, values: string[]) =>
    (params: Query): Query => ({
      ...params,
      [key]: values,
    });
  // Treat the chip as active if the filter contains ANY of the given
  // values for the key. "open" lights up whether the user has status:open,
  // status:draft, or both.
  const hasAnyValue = (key: string, values: string[]): boolean =>
    hasKey(key) && values.some((v) => params[key].includes(v));
  const toggleParamMulti =
    (key: string, values: string[]) =>
    (params: Query): Query => ({
      ...params,
      [key]: hasAnyValue(key, values) ? [] : values,
    });
  const toggleParam =
    (key: string, value: string) =>
    (params: Query): Query => ({
      ...params,
      [key]: params[key] && params[key].includes(value) ? [] : [value],
    });
  const toggleOrAddParam =
    (key: string, value: string) =>
    (params: Query): Query => {
      const values = params[key];
      return {
        ...params,
        [key]:
          params[key] && params[key].includes(value)
            ? values.filter((v) => v !== value)
            : values
              ? [...values, value]
              : [value],
      };
    };
  const clearParam =
    (key: string) =>
    (params: Query): Query => ({
      ...params,
      [key]: [],
    });

  // The kind selector is a radio group — one and only one of Issues/PRs is
  // "current" at any time. Default to Issues when kind isn't pinned.
  const currentKind: 'issue' | 'pr' = hasValue('kind', 'pr') ? 'pr' : 'issue';

  return (
    <Toolbar className={classes.toolbar}>
      <CountingFilter
        active={currentKind === 'issue'}
        query={pipe(replaceParam('kind', 'issue'), stringify)(params)}
        to={pipe(replaceParam('kind', 'issue'), loc)(params)}
        icon={BugReportOutlinedIcon}
      >
        issues
      </CountingFilter>
      <CountingFilter
        active={currentKind === 'pr'}
        query={pipe(replaceParam('kind', 'pr'), stringify)(params)}
        to={pipe(replaceParam('kind', 'pr'), loc)(params)}
        icon={CallMergeIcon}
      >
        PRs
      </CountingFilter>
      <div className={classes.separator} />
      <CountingFilter
        active={hasAnyValue('status', ['open', 'draft'])}
        query={pipe(
          replaceParam('kind', currentKind),
          replaceParamMulti('status', ['open', 'draft']),
          clearParam('sort'),
          stringify
        )(params)}
        to={pipe(replaceParamMulti('status', ['open', 'draft']), loc)(params)}
        icon={ErrorOutline}
      >
        open
      </CountingFilter>
      <CountingFilter
        active={hasAnyValue('status', ['closed', 'merged'])}
        query={pipe(
          replaceParam('kind', currentKind),
          replaceParamMulti('status', ['closed', 'merged']),
          clearParam('sort'),
          stringify
        )(params)}
        to={pipe(replaceParamMulti('status', ['closed', 'merged']), loc)(params)}
        icon={CheckCircleOutline}
      >
        closed
      </CountingFilter>
      <div className={classes.spacer} />
      {/*
      <Filter active={hasKey('author')}>Author</Filter>
      <Filter active={hasKey('label')}>Label</Filter>
      */}
      <FilterDropdown
        dropdown={identities}
        itemActive={(key) => hasValue('author', key)}
        to={(key) => pipe(toggleOrAddParam('author', key), loc)(params)}
        hasFilter
      >
        Author
      </FilterDropdown>
      <FilterDropdown
        dropdown={labels}
        itemActive={(key) => containsValue('label', key)}
        to={(key) => pipe(toggleOrAddParam('label', key), loc)(params)}
        hasFilter
      >
        Labels
      </FilterDropdown>
      <FilterDropdown
        dropdown={[
          ['id', 'ID'],
          ['creation', 'Newest'],
          ['creation-asc', 'Oldest'],
          ['edit', 'Recently updated'],
          ['edit-asc', 'Least recently updated'],
        ]}
        itemActive={(key) => hasValue('sort', key)}
        to={(key) => pipe(toggleParam('sort', key), loc)(params)}
      >
        Sort
      </FilterDropdown>
    </Toolbar>
  );
}

export default FilterToolbar;
