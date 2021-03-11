import React from 'react';

import { makeStyles } from '@material-ui/core/styles';

import { useCurrentIdentityQuery } from '../../components/CurrentIdentity/CurrentIdentity.generated';

const useStyles = makeStyles((theme) => ({}));

const Identity = () => {
  const classes = useStyles();
  const { loading, error, data } = useCurrentIdentityQuery();
  const user = data?.repository?.userIdentity;
  console.log(user);
  return <main></main>;
};

export default Identity;
