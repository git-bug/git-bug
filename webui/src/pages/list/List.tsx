import Table from '@mui/material/Table/Table';
import TableBody from '@mui/material/TableBody/TableBody';

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
