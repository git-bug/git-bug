import CircularProgress from '@material-ui/core/CircularProgress'
import gql from 'graphql-tag'
import React from 'react'
import { Query } from 'react-apollo'
import Timeline from './Timeline'
import Message from './Message'

const QUERY = gql`
  query($id: String!, $first: Int = 10, $after: String) {
    defaultRepository {
      bug(prefix: $id) {
        operations(first: $first, after: $after) {
          nodes {
            ...Create
            ...Comment
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
`

const TimelineQuery = ({id}) => (
  <Query query={QUERY} variables={{id}}>
    {({loading, error, data, fetchMore}) => {
      if (loading) return <CircularProgress/>
      if (error) return <p>Error: {error}</p>
      return <Timeline ops={data.defaultRepository.bug.operations.nodes} fetchMore={fetchMore}/>
    }}
  </Query>
)

export default TimelineQuery
