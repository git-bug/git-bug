import CircularProgress from '@material-ui/core/CircularProgress';
import gql from 'graphql-tag';
import React from 'react';
import { Query } from 'react-apollo';
import LabelChange from './LabelChange';
import SetStatus from './SetStatus';
import SetTitle from './SetTitle';
import Timeline from './Timeline';
import Message from './Message';

const QUERY = gql`
  query($id: String!, $first: Int = 10, $after: String) {
    defaultRepository {
      bug(prefix: $id) {
        timeline(first: $first, after: $after) {
          nodes {
            ...LabelChange
            ...SetStatus
            ...SetTitle
            ...AddComment
            ...Create
          }
          pageInfo {
            hasNextPage
            endCursor
          }
        }
      }
    }
  }
  ${Message.createFragment}
  ${Message.commentFragment}
  ${LabelChange.fragment}
  ${SetTitle.fragment}
  ${SetStatus.fragment}
`;

const TimelineQuery = ({ id }) => (
  <Query query={QUERY} variables={{ id, first: 100 }}>
    {({ loading, error, data, fetchMore }) => {
      if (loading) return <CircularProgress />;
      if (error) return <p>Error: {error}</p>;
      return (
        <Timeline
          ops={data.defaultRepository.bug.timeline.nodes}
          fetchMore={fetchMore}
        />
      );
    }}
  </Query>
);

export default TimelineQuery;
