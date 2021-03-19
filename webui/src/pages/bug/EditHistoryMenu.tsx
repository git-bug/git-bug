import React from 'react';

import CircularProgress from '@material-ui/core/CircularProgress';
import Menu from '@material-ui/core/Menu';
import MenuItem from '@material-ui/core/MenuItem';

import Date from 'src/components/Date';

import { AddCommentFragment } from './MessageCommentFragment.generated';
import { CreateFragment } from './MessageCreateFragment.generated';
import { useMessageEditHistoryQuery } from './MessageEditHistory.generated';

const ITEM_HEIGHT = 48;

type Props = {
  anchor: null | HTMLElement;
  bugId: string;
  commentId: string;
  onClose: () => void;
};
function EditHistoryMenu({ anchor, bugId, commentId, onClose }: Props) {
  const open = Boolean(anchor);

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

  return (
    <div>
      <Menu
        id="long-menu"
        anchorEl={anchor}
        keepMounted
        open={open}
        onClose={onClose}
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
          <MenuItem key={index} onClick={onClose}>
            <Date date={edit.date} />
          </MenuItem>
        ))}
      </Menu>
    </div>
  );
}

export default EditHistoryMenu;
