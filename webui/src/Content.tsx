import React from 'react';
import html from 'remark-html';
import parse from 'remark-parse';
import remark2react from 'remark-react';
import unified from 'unified';

import ImageTag from './tag/ImageTag';
import PreTag from './tag/PreTag';

type Props = { markdown: string };
const Content: React.FC<Props> = ({ markdown }: Props) => {
  const processor = unified()
    .use(parse)
    .use(html)
    .use(remark2react, {
      remarkReactComponents: {
        img: ImageTag,
        pre: PreTag,
      },
    });

  const contents: React.ReactNode = processor.processSync(markdown).contents;
  return <>{contents}</>;
};

export default Content;
