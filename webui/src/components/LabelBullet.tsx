import React from 'react';

import { common } from '@material-ui/core/colors';
import { makeStyles } from '@material-ui/core/styles';
import {
  getContrastRatio,
  darken,
} from '@material-ui/core/styles/colorManipulator';

import { Color } from 'src/gqlTypes';

import { LabelFragment } from './fragments.generated';

// Minimum contrast between the background and the text color
const contrastThreshold = 2.5;

// Guess the text color based on the background color
const getTextColor = (background: string) =>
  getContrastRatio(background, common.white) >= contrastThreshold
    ? common.white // White on dark backgrounds
    : common.black; // And black on light ones

const _rgb = (color: Color) =>
  'rgb(' + color.R + ',' + color.G + ',' + color.B + ')';

// Create a style object from the label RGB colors
const createStyle = (color: Color) => ({
  backgroundColor: _rgb(color),
  color: getTextColor(_rgb(color)),
  borderBottomColor: darken(_rgb(color), 0.2),
});

const useStyles = makeStyles(theme => ({
  label: {
    ...theme.typography.body1,
    padding: '8px',
    margin: '0 8px',
    fontSize: '0.9em',
    borderRadius: '3px',
    display: 'inline-block',
    borderBottom: 'solid 1.5px',
    verticalAlign: 'bottom',
  },
}));

type Props = { label: LabelFragment };
function LabelBullet({ label }: Props) {
  const classes = useStyles();
  return (
    <span className={classes.label} style={createStyle(label.color)}></span>
  );
}

export default LabelBullet;
