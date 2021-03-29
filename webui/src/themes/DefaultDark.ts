import { createMuiTheme } from '@material-ui/core/styles';

const defaultDarkTheme = createMuiTheme({
  palette: {
    type: 'dark',
    primary: {
      dark: '#263238',
      main: '#2a393e',
      light: '#525252',
    },
    error: {
      main: '#f44336',
      dark: '#ff4949',
    },
    info: {
      main: '#2a393e',
      contrastText: '#ffffffb3',
    },
    success: {
      main: '#2ea44fd9',
      contrastText: '#fff',
    },
  },
});

export default defaultDarkTheme;
