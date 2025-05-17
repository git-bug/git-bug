import { NightsStayRounded, WbSunnyRounded } from '@mui/icons-material';
import { ThemeProvider, StyledEngineProvider } from '@mui/material';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import { Theme } from '@mui/material/styles';
import * as React from 'react';
import { createContext, useContext, useState } from 'react';

declare module '@mui/styles/defaultTheme' {
  // eslint-disable-next-line @typescript-eslint/no-empty-interface
  interface DefaultTheme extends Theme {}
}

export const ThemeContext = createContext({
  toggleMode: () => {},
  mode: '',
});

type LightSwitchProps = {
  className?: string;
};
const LightSwitch = ({ className }: LightSwitchProps) => {
  const { mode, toggleMode } = useContext(ThemeContext);
  const nextMode = mode === 'light' ? 'dark' : 'light';
  const description = `Switch to ${nextMode} theme`;

  return (
    <Tooltip title={description}>
      <IconButton
        onClick={toggleMode}
        aria-label={description}
        className={className}
        size="large"
      >
        {mode === 'light' ? <WbSunnyRounded /> : <NightsStayRounded />}
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
  const savedMode = localStorage.getItem('themeMode');
  const preferedMode = savedMode != null ? savedMode : 'light';
  const [mode, setMode] = useState(preferedMode);

  const toggleMode = () => {
    const preferedMode = mode === 'light' ? 'dark' : 'light';
    localStorage.setItem('themeMode', preferedMode);
    setMode(preferedMode);
  };

  const preferedTheme = mode === 'dark' ? darkTheme : lightTheme;

  return (
    <ThemeContext.Provider value={{ toggleMode: toggleMode, mode: mode }}>
      <StyledEngineProvider injectFirst>
        <ThemeProvider theme={preferedTheme}>{children}</ThemeProvider>
      </StyledEngineProvider>
    </ThemeContext.Provider>
  );
};

export { Themer as default, LightSwitch };
