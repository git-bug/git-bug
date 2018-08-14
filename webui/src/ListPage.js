import CircularProgress from '@material-ui/core/CircularProgress'
import { withStyles } from '@material-ui/core/styles'
import Table from '@material-ui/core/Table/Table'
import TableBody from '@material-ui/core/TableBody/TableBody'
import gql from 'graphql-tag'
import React from 'react'
import { Query } from 'react-apollo'

import BugSummary from './BugSummary'

const QUERY = gql`
  {
    defaultRepository {
      bugs: allBugs(first: 10) {
        edges {
          cursor
          node {
            ...BugSummary
          }
        }
      }
    }
  }

  ${BugSummary.fragment}
`

const styles = theme => ({
  main: {
    maxWidth: 600,
    margin: 'auto',
    marginTop: theme.spacing.unit * 4
  }
})

const List = withStyles(styles)(({bugs, classes}) => (
  <main className={classes.main}>
    <Table className={classes.table}>
      <TableBody>
        {bugs.edges.map(({ cursor, node }) => (
          <BugSummary bug={node} key={cursor} />
        ))}
      </TableBody>
    </Table>
  </main>
))

const ListPage = () => (
  <Query query={QUERY}>
    {({loading, error, data}) => {
      if (loading) return <CircularProgress/>
      if (error) return <p>Error.</p>
      return <List bugs={data.defaultRepository.bugs}/>
    }}
  </Query>
)

export default ListPage
