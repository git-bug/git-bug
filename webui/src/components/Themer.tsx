import React, { createContext, useContext, useState } from 'react';

import { PaletteType, ThemeProvider, useMediaQuery } from '@material-ui/core';
import IconButton from '@material-ui/core/IconButton/IconButton';
import Tooltip from '@material-ui/core/Tooltip/Tooltip';
import { createMuiTheme } from '@material-ui/core/styles';
import { NightsStayRounded, WbSunnyRounded } from '@material-ui/icons';

const defaultTheme = {
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
  const preferseDarkMode = useMediaQuery('(prefers-color-scheme: dark)');
  const browserMode = preferseDarkMode ? 'dark' : 'light';
  const preferedMode = localStorage.getItem('themeMode');
  const curMode = preferedMode != null ? preferedMode : browserMode;

  const adjustedTheme = {
    ...theme,
    palette: {
      ...theme.palette,
      type: (curMode === 'dark' ? 'dark' : 'light') as PaletteType,
    },
  };

  const toggleMode = () => {
    const preferedMode = curMode === 'dark' ? 'light' : 'dark';
    localStorage.setItem('themeMode', preferedMode);
    const adjustedTheme = {
      ...theme,
      palette: {
        ...theme.palette,
        type: preferedMode as PaletteType,
      },
    };
    setTheme(adjustedTheme);
  };

  return (
    <ThemeContext.Provider value={{ toggleMode: toggleMode, mode: curMode }}>
      <ThemeProvider theme={createMuiTheme(adjustedTheme)}>
        {children}
      </ThemeProvider>
    </ThemeContext.Provider>
  );
};

export { Themer as default, LightSwitch };
