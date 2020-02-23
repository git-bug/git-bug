import React from 'react';

import Table from '@material-ui/core/Table/Table';
import TableBody from '@material-ui/core/TableBody/TableBody';

import BugRow from './BugRow';
import { BugListFragment } from './ListQuery.generated';

type Props = { bugs: BugListFragment };
function List({ bugs }: Props) {
  return (
    <Table>
      <TableBody>
        {bugs.edges.map(({ cursor, node }) => (
          <BugRow bug={node} key={cursor} />
        ))}
      </TableBody>
    </Table>
  );
}

export default List;
