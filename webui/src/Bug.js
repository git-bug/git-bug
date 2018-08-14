import { withStyles } from '@material-ui/core/styles'
import gql from 'graphql-tag'
import React from 'react'
import BugSummary from './BugSummary'

import Comment from './Comment'

const styles = theme => ({
  main: {
    maxWidth: 600,
    margin: 'auto',
    marginTop: theme.spacing.unit * 4
  }
})

const Bug = ({bug, classes}) => (
  <main className={classes.main}>
    <BugSummary bug={bug}/>

    {bug.comments.edges.map(({cursor, node}) => (
      <Comment key={cursor} comment={node}/>
    ))}
  </main>
)

Bug.fragment = gql`
  fragment Bug on Bug {
    ...BugSummary
    comments(first: 10) {
      edges {
        cursor
        node {
          ...Comment
        }
      }
    }
  }

  ${BugSummary.fragment}
  ${Comment.fragment}
`

export default withStyles(styles)(Bug)
