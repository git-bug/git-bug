import React, { createContext, useContext, useState } from 'react';

import { ThemeProvider, useMediaQuery } from '@material-ui/core';
import IconButton from '@material-ui/core/IconButton/IconButton';
import Tooltip from '@material-ui/core/Tooltip/Tooltip';
import { Theme } from '@material-ui/core/styles';
import { NightsStayRounded, WbSunnyRounded } from '@material-ui/icons';

const ThemeContext = createContext({
  toggleMode: () => {},
  mode: '',
});

const LightSwitch = () => {
  const { mode, toggleMode } = useContext(ThemeContext);
  const nextMode = mode === 'light' ? 'dark' : 'light';
  const description = `Switch to ${nextMode} theme`;

  return (
    <Tooltip title={description}>
      <IconButton onClick={toggleMode} aria-label={description}>
        {mode === 'light' ? (
          <WbSunnyRounded color="secondary" />
        ) : (
          <NightsStayRounded color="secondary" />
        )}
      </IconButton>
    </Tooltip>
  );
};

type Props = {
  children: React.ReactNode;
  lightTheme: Theme;
  darkTheme: Theme;
};
const Themer = ({ children, lightTheme, darkTheme }: Props) => {
  const preferseDarkMode = useMediaQuery('(prefers-color-scheme: dark)');
  const browserMode = preferseDarkMode ? 'dark' : 'light';
  const savedMode = localStorage.getItem('themeMode');
  const preferedMode = savedMode != null ? savedMode : browserMode;
  const [curMode, setMode] = useState(preferedMode);

  const toggleMode = () => {
    const preferedMode = curMode === 'light' ? 'dark' : 'light';
    localStorage.setItem('themeMode', preferedMode);
    setMode(preferedMode);
  };

  const preferedTheme = preferedMode === 'dark' ? darkTheme : lightTheme;

  return (
    <ThemeContext.Provider value={{ toggleMode: toggleMode, mode: curMode }}>
      <ThemeProvider theme={preferedTheme}>{children}</ThemeProvider>
    </ThemeContext.Provider>
  );
};

export { Themer as default, LightSwitch };
