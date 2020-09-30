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

const useStyles = makeStyles((theme) => ({
  label: {
    ...theme.typography.body1,
    padding: '1px 6px 0.5px',
    fontSize: '0.9em',
    fontWeight: 500,
    margin: '0.05em 1px calc(-1.5px + 0.05em)',
    borderRadius: '3px',
    display: 'inline-block',
    borderBottom: 'solid 1.5px',
    verticalAlign: 'bottom',
  },
}));

type Props = { label: LabelFragment };
function Label({ label }: Props) {
  const classes = useStyles();
  return (
    <span className={classes.label} style={createStyle(label.color)}>
      {label.name}
    </span>
  );
}

export default Label;
