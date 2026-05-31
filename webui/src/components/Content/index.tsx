import clsx from 'clsx';
import type { Root as HastRoot } from 'hast';
import type { Root as MdRoot } from 'mdast';
import { fromMarkdown, Options } from 'mdast-util-from-markdown';
import { useEffect, useState } from 'react';
import * as React from 'react';
import * as production from 'react/jsx-runtime';
import rehypeHighlight, {
  Options as RehypeHighlightOpts,
} from 'rehype-highlight';
import rehypeReact from 'rehype-react';
import rehypeSanitize from 'rehype-sanitize';
import remarkBreaks from 'remark-breaks';
import remarkGemoji from 'remark-gemoji';
import remarkGfm from 'remark-gfm';
import remarkParse from 'remark-parse';
import remarkRehype from 'remark-rehype';
import type { Options as RemarkRehypeOptions } from 'remark-rehype';
import { retext } from 'retext';
import retextEmoji from 'retext-emoji';
import { unified } from 'unified';
import type { Plugin, Processor } from 'unified';

import { ThemeContext } from '../Themer';

import AnchorTag from './AnchorTag';
import BlockQuoteTag from './BlockQuoteTag';
import ImageTag from './ImageTag';
import PreTag from './PreTag';

// @lygaret 2025/05/16
// type inference for some of this doesn't work, but the pipeline is fine
// this might get better when we upgrade typescript

type RetextPlugin = Plugin<any[], string, string>;
type RemarkPlugin = Plugin<[], MdRoot, HastRoot>;
type RemarkRehypePlugin = Plugin<RemarkRehypeOptions[], MdRoot, HastRoot>;
type RehypePlugin<Options extends unknown[] = []> = Plugin<
  Options,
  HastRoot,
  HastRoot
>;

const pipelines: Record<'title' | 'comment', Processor<any, any, any, any, any>> = {
  title: retext()
    .use(retextEmoji as unknown as RetextPlugin, { convert: 'encode' }),

  comment: unified()
    .use(remarkParse)
    .use(remarkGemoji as unknown as RemarkPlugin)
    .use(remarkBreaks as unknown as RemarkPlugin)
    .use(remarkGfm)
    .use(remarkRehype as unknown as RemarkRehypePlugin, {
      allowDangerousHtml: true,
    })
    .use(rehypeSanitize as unknown as RehypePlugin)
    .use(rehypeHighlight as unknown as RehypePlugin<RehypeHighlightOpts[]>, {
      detect: true,
      subset: ['text'],
    })
    .use(rehypeReact, {
      ...production,
      components: {
        a: AnchorTag,
        blockquote: BlockQuoteTag,
        img: ImageTag,
        pre: PreTag,
      },
    })
};

type Props = { 
  className?: string,
  markdown: string,
  inline?: boolean,
  pipeline?: keyof typeof pipelines
};

const Content: React.FC<Props> = ({ markdown, pipeline, inline, className }: Props) => {
  const theme = React.useContext(ThemeContext);
  const [content, setContent] = useState(<></>);

  useEffect(() => {
    pipelines[pipeline ?? 'comment']
      .process(markdown)
      .then((file) => {
        setContent(file.result ?? <>{file.value}</>)
      })
      .catch((err: any) => {
        setContent(
          <>
            <span className="error">{err}</span>
            <pre>{markdown}</pre>
          </>
        );
      });
  }, [markdown]);

  return inline
    ? <span className={className}>{content}</span>
    : <div className={clsx('highlight-theme', className)} data-theme={theme.mode}>
        {content}
      </div>;
};

export default Content;
