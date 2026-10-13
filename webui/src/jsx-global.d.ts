// React 19 only declares JSX under React.JSX. Some libraries (e.g.
// hast-util-to-jsx-runtime) still type their output with the global JSX
// namespace, which would otherwise resolve to nothing and degrade to `any`.
import type { JSX as ReactJSX } from "react";

declare global {
  namespace JSX {
    type Element = ReactJSX.Element;
    type ElementClass = ReactJSX.ElementClass;
    type IntrinsicElements = ReactJSX.IntrinsicElements;
  }
}
