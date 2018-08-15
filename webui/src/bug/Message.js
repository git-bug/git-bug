import { withStyles } from '@material-ui/core/styles'
import Tooltip from '@material-ui/core/Tooltip/Tooltip'
import Typography from '@material-ui/core/Typography'
import gql from 'graphql-tag'
import * as moment from 'moment'
import React from 'react'

const styles = theme => ({
  header: {
    ...theme.typography.body2,
    padding: '3px 3px 3px 6px',
    backgroundColor: '#f1f8ff',
    border: '1px solid #d1d5da',
    borderTopLeftRadius: 3,
    borderTopRightRadius: 3,
  },
  author: {
    ...theme.typography.body2,
    fontWeight: 'bold'
  },
  message: {
    borderLeft: '1px solid #d1d5da',
    borderRight: '1px solid #d1d5da',
    borderBottom: '1px solid #d1d5da',
    borderBottomLeftRadius: 3,
    borderBottomRightRadius: 3,
    backgroundColor: '#fff',
    minHeight: 50
  }
})

const Message = ({message, classes}) => (
  <div>
    <div className={classes.header}>
      <Tooltip title={message.author.email}>
        <span className={classes.author}>{message.author.name}</span>
      </Tooltip>
      <span> commented </span>
      <Tooltip title={moment(message.date).format('MMMM D, YYYY, h:mm a')}>
        <span> {moment(message.date).fromNow()} </span>
      </Tooltip>
    </div>
    <div className={classes.message}>
      <Typography>{message.message}</Typography>
    </div>
  </div>
)

Message.createFragment = gql`
  fragment Create on Operation {
    ... on CreateOperation {
      date
      author {
        name
        email
      }
      message
    }
  }
`

Message.commentFragment = gql`
  fragment Comment on Operation {
    ... on AddCommentOperation {
      date
      author {
        name
        email
      }
      message
    }
  }
`

export default withStyles(styles)(Message)
