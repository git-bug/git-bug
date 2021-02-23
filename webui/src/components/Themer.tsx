import React, { createContext, useCallback, useContext, useState } from 'react';

import { ThemeProvider } from '@material-ui/core';
import IconButton from '@material-ui/core/IconButton/IconButton';
import Tooltip from '@material-ui/core/Tooltip/Tooltip';
import { createMuiTheme, ThemeOptions } from '@material-ui/core/styles';
import { NightsStayRounded, WbSunnyRounded } from '@material-ui/icons';

const defaultTheme: ThemeOptions = {
  palette: {
    type: 'light',
    primary: {
      main: '#263238',
    },
  },
};

const ThemeContext = createContext({
  toggleMode: () => {},
  mode: '',
});

const LightSwitch = () => {
  const { mode, toggleMode } = useContext(ThemeContext);

  return (
    <Tooltip title="Toggle Dark-/Lightmode">
      <IconButton onClick={toggleMode} aria-label="Toggle Dark-/Lightmode">
        {mode === 'light' ? (
          <WbSunnyRounded color="secondary" />
        ) : (
          <NightsStayRounded color="secondary" />
        )}
      </IconButton>
    </Tooltip>
  );
};

type Props = { children: React.ReactNode };
const Themer = ({ children }: Props) => {
  const [theme, setTheme] = useState(defaultTheme);

  const toggleMode = useCallback(() => {
    const newMode = theme.palette?.type === 'dark' ? 'light' : 'dark';
    const adjustedTheme: ThemeOptions = {
      ...theme,
      palette: {
        ...theme.palette,
        type: newMode,
      },
    };
    setTheme(adjustedTheme);
  }, [theme, setTheme]);

  const newMode = theme.palette?.type === 'dark' ? 'light' : 'dark';

  return (
    <ThemeContext.Provider value={{ toggleMode: toggleMode, mode: newMode }}>
      <ThemeProvider theme={createMuiTheme(theme)}> {children} </ThemeProvider>
    </ThemeContext.Provider>
  );
};

export { Themer as default, LightSwitch };
