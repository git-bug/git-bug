import React from 'react';
import { makeStyles } from '@material-ui/styles';
import {
  getContrastRatio,
  darken,
} from '@material-ui/core/styles/colorManipulator';
import { common } from '@material-ui/core/colors';

// Minimum contrast between the background and the text color
const contrastThreshold = 2.5;

// Guess the text color based on the background color
const getTextColor = background =>
  getContrastRatio(background, common.white) >= contrastThreshold
    ? common.white // White on dark backgrounds
    : common.black; // And black on light ones

const _rgb = color => 'rgb(' + color.R + ',' + color.G + ',' + color.B + ')';

// Create a style object from the label RGB colors
const createStyle = color => ({
  backgroundColor: _rgb(color),
  color: getTextColor(_rgb(color)),
  borderBottomColor: darken(_rgb(color), 0.2),
});

const useStyles = makeStyles(theme => ({
  label: {
    ...theme.typography.body1,
    padding: '1px 6px 0.5px',
    fontSize: '0.9em',
    fontWeight: '500',
    margin: '0.05em 1px calc(-1.5px + 0.05em)',
    borderRadius: '3px',
    display: 'inline-block',
    borderBottom: 'solid 1.5px',
    verticalAlign: 'bottom',
  },
}));

function Label({ label }) {
  const classes = useStyles();
  return (
    <span className={classes.label} style={createStyle(label.color)}>
      {label.name}
    </span>
  );
}

export default Label;
