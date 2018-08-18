import React from 'react';
import { withStyles } from '@material-ui/core/styles';
import {
  getContrastRatio,
  darken,
} from '@material-ui/core/styles/colorManipulator';
import * as allColors from '@material-ui/core/colors';
import { common } from '@material-ui/core/colors';

// JS's modulo returns negative numbers sometimes.
// This ensures the result is positive.
const mod = (n, m) => ((n % m) + m) % m;

// Minimum contrast between the background and the text color
const contrastThreshold = 2.5;

// Filter out the "common" color
const labelColors = Object.entries(allColors)
  .filter(([key, value]) => value !== common)
  .map(([key, value]) => value);

// Generate a hash (number) from a string
const hash = string =>
  string.split('').reduce((a, b) => ((a << 5) - a + b.charCodeAt(0)) | 0, 0);

// Get the background color from the label
const getColor = label =>
  labelColors[mod(hash(label), labelColors.length)][500];

// Guess the text color based on the background color
const getTextColor = background =>
  getContrastRatio(background, common.white) >= contrastThreshold
    ? common.white // White on dark backgrounds
    : common.black; // And black on light ones

const _genStyle = background => ({
  backgroundColor: background,
  color: getTextColor(background),
  borderBottomColor: darken(background, 0.2),
});

// Generate a style object (text, background and border colors) from the label
const genStyle = label => _genStyle(getColor(label));

const styles = theme => ({
  label: {
    ...theme.typography.body2,
    padding: '0 6px',
    fontSize: '0.9em',
    margin: '0 1px',
    borderRadius: '3px',
    display: 'inline-block',
    borderBottom: 'solid 1.5px',
    verticalAlign: 'bottom',
  },
});

const Label = ({ label, classes }) => (
  <span className={classes.label} style={genStyle(label)}>
    {label}
  </span>
);

export default withStyles(styles)(Label);
