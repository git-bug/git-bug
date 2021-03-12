import React from 'react';

import { MenuItem, MenuList } from '@material-ui/core';
import Avatar from '@material-ui/core/Avatar';
import { makeStyles } from '@material-ui/core/styles';

import { useCurrentIdentityQuery } from '../../components/CurrentIdentity/CurrentIdentity.generated';

const useStyles = makeStyles((theme) => ({}));

const Identity = () => {
  const classes = useStyles();
  const { loading, error, data } = useCurrentIdentityQuery();
  const user = data?.repository?.userIdentity;
  console.log(user);
  return (
    <main>
      <h1>Profile</h1>
      <Avatar src={user?.avatarUrl ? user.avatarUrl : undefined}>
        {user?.displayName.charAt(0).toUpperCase()}
      </Avatar>
      <ul>
        <li>Name: {user?.name ? user?.name : 'none'}</li>
        <li title={user?.id}>Id: {user?.humanId ? user?.humanId : 'none'}</li>
        <li>Email: {user?.email ? user?.email : 'none'}</li>
        <li>Login: {user?.login ? user?.login : 'none'}</li>
        <li>Protected: {user?.isProtected}</li>
      </ul>
    </main>
  );
};

export default Identity;
