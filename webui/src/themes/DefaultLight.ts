import { createMuiTheme } from '@material-ui/core/styles';

const defaultLightTheme = createMuiTheme({
  palette: {
    type: 'light',
    primary: {
      main: '#263238',
      light: '#f5f5f5',
    },
    info: {
      main: '#e2f1ff',
    },
    text: {
      secondary: '#555',
    },
  },
});

export default defaultLightTheme;
