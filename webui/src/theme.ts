import { createMuiTheme } from '@material-ui/core/styles';

const defaultLightTheme = createMuiTheme({
  palette: {
    type: 'light',
    primary: {
      main: '#263238',
    },
  },
});

const defaultDarkTheme = createMuiTheme({
  palette: {
    type: 'dark',
    primary: {
      main: '#263238',
    },
  },
});

export { defaultLightTheme, defaultDarkTheme };
