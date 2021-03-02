import { createMuiTheme } from '@material-ui/core/styles';

const defaultDarkTheme = createMuiTheme({
  palette: {
    type: 'dark',
    primary: {
      main: '#263238',
      light: '#525252',
    },
    error: {
      main: '#f44336',
      dark: '#ff4949',
    },
    info: {
      main: '#2a393e',
    },
  },
});

export default defaultDarkTheme;
