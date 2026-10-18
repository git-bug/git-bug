// Drafts: text typed but not sent yet, kept in the browser's localStorage so
// that it survives a reload, a closed tab or a browser restart.  A draft is
// written on each change and removed once empty, so sending it is clearing it.
// Drafts left alone for MAX_AGE_MS are forgotten.  Without storage (some
// private windows), drafts are not kept, and nothing else changes.

import { useCallback, useState } from "react";

const PREFIX = "git-bug:draft:";
const MAX_AGE_MS = 30 * 24 * 60 * 60 * 1000;

interface Stored {
  value: string;
  savedAt: number;
}

function isStored(v: unknown): v is Stored {
  return (
    typeof v === "object" &&
    v !== null &&
    "value" in v &&
    typeof v.value === "string" &&
    "savedAt" in v &&
    typeof v.savedAt === "number"
  );
}

function load(storageKey: string): Stored | null {
  try {
    const raw = localStorage.getItem(storageKey);
    const stored: unknown = raw === null ? null : JSON.parse(raw);
    return isStored(stored) ? stored : null;
  } catch {
    return null;
  }
}

function read(key: string): string {
  const stored = load(PREFIX + key);
  return stored && Date.now() - stored.savedAt < MAX_AGE_MS ? stored.value : "";
}

// Emptying removes the draft only while it still holds previous, the text this
// page saved last: otherwise another tab saved a newer one since, kept as is.
function write(key: string, value: string, previous: string) {
  try {
    if (value === "") {
      if (load(PREFIX + key)?.value === previous) localStorage.removeItem(PREFIX + key);
    } else {
      localStorage.setItem(PREFIX + key, JSON.stringify({ value, savedAt: Date.now() }));
    }
  } catch {
    // storage full or unavailable: the draft is not kept
  }
}

let pruned = false;

// Removes the expired drafts, once per page load.
function prune() {
  if (pruned) return;
  pruned = true;
  try {
    const expired: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const storageKey = localStorage.key(i);
      if (!storageKey?.startsWith(PREFIX)) continue;
      const stored = load(storageKey);
      if (!stored || Date.now() - stored.savedAt >= MAX_AGE_MS) expired.push(storageKey);
    }
    for (const storageKey of expired) localStorage.removeItem(storageKey);
  } catch {
    // storage unavailable: nothing to prune
  }
}

// Like useState for a string, kept as the draft of the key.  The key names
// what the text is for, like a comment on a bug of a repository; when it
// changes, the value becomes the draft of the new key.
export function useDraft(key: string): [string, (value: string) => void] {
  const [value, setValue] = useState(() => {
    prune();
    return read(key);
  });

  // Adjusting state during render is React's recommended alternative to
  // syncing it in an effect.
  const [loadedKey, setLoadedKey] = useState(key);
  if (key !== loadedKey) {
    setLoadedKey(key);
    setValue(read(key));
  }

  const update = useCallback(
    (next: string) => {
      setValue(next);
      write(key, next, value);
    },
    [key, value],
  );
  return [value, update];
}
