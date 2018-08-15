import React from 'react'
import { withStyles } from '@material-ui/core/styles'

const styles = theme => ({
  label: {
    padding: '0 4px',
    margin: '0 1px',
    backgroundColor: '#da9898',
    borderRadius: '3px'
  },
})

const Label = ({label, classes}) => <span className={classes.label}>{label}</span>

export default withStyles(styles)(Label)
