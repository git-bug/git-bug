import React from 'react';

import CircularProgress from '@material-ui/core/CircularProgress';
import IconButton, { IconButtonProps } from '@material-ui/core/IconButton';
import Menu from '@material-ui/core/Menu';
import MenuItem from '@material-ui/core/MenuItem';
import HistoryIcon from '@material-ui/icons/History';

import Date from 'src/components/Date';

import { AddCommentFragment } from './MessageCommentFragment.generated';
import { CreateFragment } from './MessageCreateFragment.generated';
import { useMessageEditHistoryQuery } from './MessageEditHistory.generated';

const ITEM_HEIGHT = 48;

type Props = {
  bugId: string;
  commentId: string;
  iconBtnProps?: IconButtonProps;
};
function EditHistoryMenu({ iconBtnProps, bugId, commentId }: Props) {
  const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
  const open = Boolean(anchorEl);

  const { loading, error, data } = useMessageEditHistoryQuery({
    variables: { bugIdPrefix: bugId },
  });
  if (loading) return <CircularProgress />;
  if (error) return <p>Error: {error}</p>;

  const comments = data?.repository?.bug?.timeline.comments as (
    | AddCommentFragment
    | CreateFragment
  )[];
  // NOTE Searching for the changed comment could be dropped if GraphQL get
  // filter by id argument for timelineitems
  const comment = comments.find((elem) => elem.id === commentId);
  const history = comment?.history;

  const handleClick = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  return (
    <div>
      <IconButton
        aria-label="more"
        aria-controls="long-menu"
        aria-haspopup="true"
        onClick={handleClick}
        {...iconBtnProps}
      >
        <HistoryIcon />
      </IconButton>
      <Menu
        id="long-menu"
        anchorEl={anchorEl}
        keepMounted
        open={open}
        onClose={handleClose}
        PaperProps={{
          style: {
            maxHeight: ITEM_HEIGHT * 4.5,
            width: '20ch',
          },
        }}
      >
        <MenuItem key={0} disabled>
          Edited {history?.length} times.
        </MenuItem>
        {history?.map((edit, index) => (
          <MenuItem key={index} onClick={handleClose}>
            <Date date={edit.date} />
          </MenuItem>
        ))}
      </Menu>
    </div>
  );
}

export default EditHistoryMenu;
