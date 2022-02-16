import React, { createElement, Fragment, useEffect, useState } from 'react';
import rehypeReact from 'rehype-react';
import gemoji from 'remark-gemoji';
import html from 'remark-html';
import parse from 'remark-parse';
import remarkRehype from 'remark-rehype';
import { unified } from 'unified';

import AnchorTag from './AnchorTag';
import BlockQuoteTag from './BlockQuoteTag';
import ImageTag from './ImageTag';
import PreTag from './PreTag';

type Props = { markdown: string };
const Content: React.FC<Props> = ({ markdown }: Props) => {
  const [Content, setContent] = useState(<></>);

  useEffect(() => {
    unified()
      .use(parse)
      .use(gemoji)
      .use(html)
      .use(remarkRehype)
      .use(rehypeReact, {
        createElement,
        Fragment,
        components: {
          img: ImageTag,
          pre: PreTag,
          a: AnchorTag,
          blockquote: BlockQuoteTag,
        },
      })
      .process(markdown)
      .then((file) => {
        setContent(file.result);
      });
  }, [markdown]);

  return Content;
};

export default Content;
