import { withStyles } from '@material-ui/core/styles'
import TableCell from '@material-ui/core/TableCell/TableCell'
import TableRow from '@material-ui/core/TableRow/TableRow'
import Tooltip from '@material-ui/core/Tooltip/Tooltip'
import Typography from '@material-ui/core/Typography'
import ErrorOutline from '@material-ui/icons/ErrorOutline'
import gql from 'graphql-tag'
import React from 'react'
import { Link } from 'react-router-dom'
import * as moment from 'moment'

const Open = ({className}) => <Tooltip title="Open">
  <ErrorOutline nativeColor='#28a745' className={className}/>
</Tooltip>

const Closed = ({className}) => <Tooltip title="Closed">
  <ErrorOutline nativeColor='#cb2431' className={className}/>
</Tooltip>

const Status = ({status, className}) => {
    switch(status) {
      case 'OPEN': return <Open className={className}/>
      case 'CLOSED': return <Closed className={className}/>
      default: return 'unknown status ' + status
    }
}

const styles = theme => ({
  cell: {
    display: 'flex',
    alignItems: 'center'
  },
  status: {
    margin: 10
  },
  title: {
    display: 'inline-block',
    textDecoration: 'none'
  },
  labels: {
    display: 'inline-block',
    paddingLeft: theme.spacing.unit,
    '&>span': {
      padding: '0 4px',
      margin: '0 1px',
      backgroundColor: '#da9898',
      borderRadius: '3px',
    }
  },
})

const BugSummary = ({bug, classes}) => (
  <TableRow hover>
    <TableCell className={classes.cell}>
      <Status status={bug.status} className={classes.status}/>
      <div>
        <Link to={'bug/'+bug.humanId}>
          <Typography variant={'title'} className={classes.title}>
            {bug.title}
          </Typography>
        </Link>
        <span className={classes.labels}>
          {bug.labels.map(l => (
            <span key={l}>{l}</span>)
          )}
        </span>
        <Typography color={'textSecondary'}>
          {bug.humanId} opened
          <Tooltip title={moment(bug.createdAt).format('MMMM D, YYYY, h:mm a')}>
            <span> {moment(bug.createdAt).fromNow()} </span>
          </Tooltip>
          by {bug.author.name}
        </Typography>
      </div>
    </TableCell>
  </TableRow>
)

BugSummary.fragment = gql`
  fragment BugSummary on Bug {
    id
    humanId
    title
    status
    createdAt
    labels
    author {
      name
    }
  }
`

export default withStyles(styles)(BugSummary)
