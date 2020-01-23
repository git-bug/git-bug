import Table from '@material-ui/core/Table/Table';
import TableBody from '@material-ui/core/TableBody/TableBody';
import React from 'react';
import BugRow from './BugRow';

function List({ bugs }) {
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
