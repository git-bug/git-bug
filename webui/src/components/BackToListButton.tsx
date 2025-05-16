import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import Button from '@mui/material/Button';
import makeStyles from '@mui/styles/makeStyles';
import { Link } from 'react-router';

const useStyles = makeStyles((theme) => ({
  backButton: {
    position: 'sticky',
    top: '80px',
    backgroundColor: theme.palette.primary.dark,
    color: theme.palette.primary.contrastText,
    '&:hover': {
      backgroundColor: theme.palette.primary.main,
      color: theme.palette.primary.contrastText,
    },
  },
}));

function BackToListButton() {
  const classes = useStyles();

  return (
    <Button
      variant="contained"
      className={classes.backButton}
      aria-label="back to issue list"
      component={Link}
      to="/"
    >
      <ArrowBackIcon />
      Back to List
    </Button>
  );
}

export default BackToListButton;
