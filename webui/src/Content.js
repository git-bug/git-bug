import unified from 'unified';
import parse from 'remark-parse';
import html from 'remark-html';
import remark2react from 'remark-react';

const Content = ({ markdown }) => {
  const processor = unified()
    .use(parse)
    .use(html)
    .use(remark2react);

  return processor.processSync(markdown).contents;
};

export default Content;
