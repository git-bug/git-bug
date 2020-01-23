import unified from 'unified';
import parse from 'remark-parse';
import html from 'remark-html';
import remark2react from 'remark-react';
import ImageTag from './tag/ImageTag';

const Content = ({ markdown }) => {
  const processor = unified()
    .use(parse)
    .use(html)
    .use(remark2react, {
      remarkReactComponents: {
        img: ImageTag,
      },
    });

  return processor.processSync(markdown).contents;
};

export default Content;
