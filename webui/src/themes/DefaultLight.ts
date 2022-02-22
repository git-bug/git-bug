import { createTheme } from '@mui/material/styles';

const defaultLightTheme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      dark: '#263238',
      main: '#5a6b73',
      light: '#f5f5f5',
      contrastText: '#fff',
    },
    info: {
      main: '#e2f1ff',
      contrastText: '#555',
    },
    success: {
      main: '#2ea44fd9',
      contrastText: '#fff',
    },
    text: {
      secondary: '#555',
    },
  },
});

export default defaultLightTheme;
