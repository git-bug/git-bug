import React from 'react';
import { Link } from 'react-router-dom';

import './GBButton.css';

interface GBButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  to: string;
  text: string;
}

/**
 * Standard button for issue actions
 */
const GBButton: React.FC<GBButtonProps> = (props) => {
  return (
    <Link to={props.to} className="bt-issue">
      {props.text}
    </Link>
  );
};

export default GBButton;
