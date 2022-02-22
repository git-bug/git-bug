import { alpha, TextField } from '@mui/material';
import { Theme } from '@mui/material/styles';
import createStyles from '@mui/styles/createStyles';
import withStyles from '@mui/styles/withStyles';

const BugTitleInput = withStyles((theme: Theme) =>
  createStyles({
    root: {
      '& .MuiInputLabel-outlined': {
        color: theme.palette.text.primary,
      },
      '& input:valid + fieldset': {
        color: theme.palette.text.primary,
        borderColor: theme.palette.divider,
        borderWidth: 2,
      },
      '& input:valid:hover + fieldset': {
        color: theme.palette.text.primary,
        borderColor: alpha(theme.palette.divider, 0.3),
        borderWidth: 2,
      },
      '& input:valid:focus + fieldset': {
        color: theme.palette.text.primary,
        borderColor: theme.palette.divider,
      },
      '& input:invalid + fieldset': {
        borderColor: theme.palette.error.main,
        borderWidth: 2,
      },
      '& input:invalid:hover + fieldset': {
        borderColor: theme.palette.error.main,
        borderWidth: 2,
      },
      '& input:invalid:focus + fieldset': {
        borderColor: theme.palette.error.main,
        borderWidth: 2,
      },
    },
  })
)(TextField);

export default BugTitleInput;
