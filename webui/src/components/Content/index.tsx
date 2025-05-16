import type { Root as HastRoot } from 'hast';
import type { Root as MdRoot } from 'mdast';
import { useEffect, useState } from 'react';
import * as React from 'react';
import * as production from 'react/jsx-runtime';
import rehypeHighlight from 'rehype-highlight';
import rehypeReact from 'rehype-react';
import rehypeSanitize from 'rehype-sanitize';
import remarkBreaks from 'remark-breaks';
import remarkGemoji from 'remark-gemoji';
import remarkGfm from 'remark-gfm';
import remarkParse from 'remark-parse';
import remarkRehype from 'remark-rehype';
import type { Options as RemarkRehypeOptions } from 'remark-rehype';
import { unified } from 'unified';
import type { Plugin, Processor } from 'unified';
import { Node as UnistNode } from 'unified/lib';

import AnchorTag from './AnchorTag';
import BlockQuoteTag from './BlockQuoteTag';
import ImageTag from './ImageTag';
import PreTag from './PreTag';

type Props = { markdown: string };

// @lygaret 2025/05/16
// type inference for some of this doesn't work, but the pipeline is fine
// this might get better when we upgrade typescript

type RemarkPlugin = Plugin<[], MdRoot, HastRoot>;
type RemarkRehypePlugin = Plugin<RemarkRehypeOptions[], MdRoot, HastRoot>;
type RehypePlugin<Options extends unknown[] = []> = Plugin<
  Options,
  HastRoot,
  HastRoot
>;

const markdownPipeline: Processor<
  UnistNode,
  undefined,
  undefined,
  HastRoot,
  React.JSX.Element
> = unified()
  .use(remarkParse)
  .use(remarkGemoji as unknown as RemarkPlugin)
  .use(remarkBreaks as unknown as RemarkPlugin)
  .use(remarkGfm)
  .use(remarkRehype as unknown as RemarkRehypePlugin, {
    allowDangerousHtml: true,
  })
  .use(rehypeSanitize as unknown as RehypePlugin)
  .use(rehypeHighlight as unknown as RehypePlugin)
  .use(rehypeReact, {
    ...production,
    components: {
      a: AnchorTag,
      blockquote: BlockQuoteTag,
      img: ImageTag,
      pre: PreTag,
    },
  });

const Content: React.FC<Props> = ({ markdown }: Props) => {
  const [content, setContent] = useState(<></>);

  useEffect(() => {
    markdownPipeline
      .process(markdown)
      .then((file) => setContent(file.result))
      .catch((err: any) => {
        setContent(
          <>
            <span className="error">{err}</span>
            <pre>{markdown}</pre>
          </>
        );
      });
  }, [markdown]);

  return content;
};

export default Content;
