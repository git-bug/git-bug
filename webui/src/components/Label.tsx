import { Chip } from '@mui/material';
import { common } from '@mui/material/colors';
import { darken, getContrastRatio } from '@mui/material/styles';

import { Color } from '../gqlTypes';
import { LabelFragment } from '../graphql/fragments.generated';

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
const createStyle = (color: Color, maxWidth?: string) => ({
  backgroundColor: _rgb(color),
  color: getTextColor(_rgb(color)),
  borderBottomColor: darken(_rgb(color), 0.2),
  maxWidth: maxWidth,
});

type Props = {
  label: LabelFragment;
  inline?: boolean;
  maxWidth?: string;
  className?: string;
};
function Label({ label, inline, maxWidth, className }: Props) {
  return (
    <Chip
      size={'small'}
      label={label.name}
      component={inline ? 'span' : 'div'}
      className={className}
      style={createStyle(label.color, maxWidth)}
    />
  );
}
export default Label;
