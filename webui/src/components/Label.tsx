import React from 'react';

import { Chip } from '@material-ui/core';
import { common } from '@material-ui/core/colors';
import {
  darken,
  getContrastRatio,
} from '@material-ui/core/styles/colorManipulator';

import { Color } from '../gqlTypes';

import { LabelFragment } from './fragments.generated';

const _rgb = (color: Color) =>
  'rgb(' + color.R + ',' + color.G + ',' + color.B + ')';

// Minimum contrast between the background and the text color
const contrastThreshold = 2.5;
// Guess the text color based on the background color
const getTextColor = (background: string) =>
  getContrastRatio(background, common.white) >= contrastThreshold
    ? common.white // White on dark backgrounds
    : common.black; // And black on light ones

// Create a style object from the label RGB colors
const createStyle = (color: Color) => ({
  backgroundColor: _rgb(color),
  color: getTextColor(_rgb(color)),
  borderBottomColor: darken(_rgb(color), 0.2),
  margin: '3px',
});

type Props = { label: LabelFragment };
function Label({ label }: Props) {
  return (
    <Chip
      size={'small'}
      label={label.name}
      style={createStyle(label.color)}
    ></Chip>
  );
}
export default Label;
