import React from 'react';
import gemoji from 'remark-gemoji';
import html from 'remark-html';
import parse from 'remark-parse';
import remark2react from 'remark-react';
import unified from 'unified';

import BlockQuoteTag from './BlockQuoteTag';
import ImageTag from './ImageTag';
import PreTag from './PreTag';

type Props = { markdown: string };
const Content: React.FC<Props> = ({ markdown }: Props) => {
  const content = unified()
    .use(parse)
    .use(gemoji)
    .use(html)
    .use(remark2react, {
      remarkReactComponents: {
        img: ImageTag,
        pre: PreTag,
        blockquote: BlockQuoteTag,
      },
    })
    .processSync(markdown).result;

  return <>{content}</>;
};

export default Content;
