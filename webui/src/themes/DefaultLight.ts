import { createMuiTheme } from '@material-ui/core/styles';

const defaultLightTheme = createMuiTheme({
  palette: {
    type: 'light',
    primary: {
      main: '#263238',
    },
  },
});

export default defaultLightTheme;
