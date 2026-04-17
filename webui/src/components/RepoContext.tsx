import { useEffect } from 'react';
import { useParams } from 'react-router';

import { setCurrentRepoName } from '../apollo';

/** RepoContextBinder reads :repoName from the URL and pushes it into the
 *  Apollo header BEFORE its children render — setting the value in a
 *  useEffect would race the first useQuery in children (Apollo fires the
 *  request during child render, before any effect has run). The cleanup on
 *  unmount still uses useEffect. */
export default function RepoContextBinder({
  children,
}: {
  children: React.ReactNode;
}) {
  const { repoName } = useParams<{ repoName: string }>();
  // react-router's decoding behaviour has shifted between major versions, so
  // belt-and-braces: try decodeURIComponent and fall back if it would throw
  // (e.g. the value was already decoded by the router).
  const decoded = repoName ? safeDecode(repoName) : null;
  // Synchronous set: render-time side effect so Apollo reads the right
  // repo name on the initial query. React may invoke render twice in
  // StrictMode, which is fine — the setter is idempotent.
  setCurrentRepoName(decoded);
  useEffect(() => {
    return () => setCurrentRepoName(null);
  }, [repoName]);
  return <>{children}</>;
}

function safeDecode(s: string): string {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
}
