import { createMuiTheme } from '@material-ui/core/styles';

const defaultDarkTheme = createMuiTheme({
  palette: {
    type: 'dark',
    primary: {
      main: '#263238',
    },
  },
});

export default defaultDarkTheme;
