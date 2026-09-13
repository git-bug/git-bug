import {
  useFloating,
  useClick,
  useDismiss,
  useRole,
  useListNavigation,
  useInteractions,
  offset,
  flip,
  FloatingPortal,
  FloatingFocusManager,
} from "@floating-ui/react";
import { GitBranch, Tag } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { GitRefType } from "@/__generated__/enums";
import { useFragment, type FragmentType } from "@/__generated__/fragment-masking";
import { graphql } from "@/__generated__/gql";
import { Button } from "@/components/ui/button";
import * as Listbox from "@/components/ui/listbox";
import { cn } from "@/lib/utils";

export const REF_SELECTOR_REFS_FRAGMENT = graphql(`
  fragment RefSelectorRefs on GitRefConnection {
    nodes {
      name
      shortName
      type
    }
  }
`);

interface RefSelectorProps {
  refs: FragmentType<typeof REF_SELECTOR_REFS_FRAGMENT>;
  currentRef: string;
  onSelect: (shortName: string) => void;
}

// Branch / tag selector dropdown for the code browser. Shown in two groups
// (branches, tags) with an inline search filter.
export function RefSelector({ refs: refsProp, currentRef, onSelect }: RefSelectorProps) {
  const data = useFragment(REF_SELECTOR_REFS_FRAGMENT, refsProp);
  const gitRefs = data.nodes;

  const [open, setOpen] = useState(false);
  const [filter, setFilter] = useState("");
  const [activeIndex, setActiveIndex] = useState<number | null>(null);

  const elementsRef = useRef<(HTMLElement | null)[]>([]);
  const searchRef = useRef<HTMLInputElement>(null);

  const { refs, floatingStyles, context } = useFloating({
    open,
    onOpenChange(nextOpen) {
      setOpen(nextOpen);
      if (!nextOpen) setFilter("");
    },
    placement: "bottom-start",
    middleware: [offset(4), flip()],
  });

  const click = useClick(context);
  const dismiss = useDismiss(context);
  const role = useRole(context, { role: "listbox" });
  const listNav = useListNavigation(context, {
    listRef: elementsRef,
    activeIndex,
    onNavigate: setActiveIndex,
    loop: true,
    virtual: true,
    focusItemOnOpen: false,
  });

  const { getReferenceProps, getFloatingProps, getItemProps } = useInteractions([
    click,
    dismiss,
    role,
    listNav,
  ]);

  const filtered = gitRefs.filter((r) => r.shortName.toLowerCase().includes(filter.toLowerCase()));
  const branches = filtered.filter((r) => r.type === GitRefType.Branch);
  const tags = filtered.filter((r) => r.type === GitRefType.Tag);

  // Build a flat list for indexing (branches first, then tags)
  const flatItems = [...branches, ...tags];

  // Reset active index when filtered list changes
  useEffect(() => {
    setActiveIndex(flatItems.length > 0 ? 0 : null);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- reset on filter change
  }, [filter]);

  function handleSearchKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter" && activeIndex != null) {
      e.preventDefault();
      const ref = flatItems[activeIndex];
      if (ref) {
        onSelect(ref.shortName);
        setOpen(false);
        setFilter("");
      }
    }
  }

  let itemIndex = 0;

  return (
    <>
      <Button
        ref={refs.setReference}
        variant="outline"
        size="sm"
        className="gap-2 font-mono text-xs"
        {...getReferenceProps()}
      >
        <GitBranch className="size-3.5" />
        {currentRef}
      </Button>

      {open && (
        <FloatingPortal>
          <FloatingFocusManager context={context} modal={false} initialFocus={searchRef}>
            <Listbox.Content
              ref={refs.setFloating}
              style={floatingStyles}
              className="w-64"
              {...getFloatingProps()}
            >
              <div className="text-muted-foreground px-3 pt-2 pb-1 text-xs font-semibold">
                Switch branch / tag
              </div>
              <Listbox.Search
                ref={searchRef}
                placeholder="Filter…"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                onKeyDown={handleSearchKeyDown}
                className="text-xs"
                aria-activedescendant={
                  activeIndex != null ? `ref-option-${activeIndex}` : undefined
                }
              />
              <Listbox.ScrollArea>
                {branches.length > 0 && (
                  <Listbox.Group>
                    <Listbox.GroupLabel>Branches</Listbox.GroupLabel>
                    {branches.map((ref) => {
                      const i = itemIndex++;
                      return (
                        <Listbox.Item
                          key={ref.name}
                          id={`ref-option-${i}`}
                          ref={(el) => {
                            elementsRef.current[i] = el;
                          }}
                          active={activeIndex === i}
                          selected={ref.shortName === currentRef}
                          className={cn("text-xs", ref.shortName === currentRef && "font-medium")}
                          {...getItemProps({
                            onClick: () => {
                              onSelect(ref.shortName);
                              setOpen(false);
                              setFilter("");
                            },
                          })}
                        >
                          <GitBranch className="text-muted-foreground size-3 shrink-0" />
                          <span className="flex-1 truncate font-mono">{ref.shortName}</span>
                        </Listbox.Item>
                      );
                    })}
                  </Listbox.Group>
                )}
                {tags.length > 0 && (
                  <Listbox.Group>
                    <Listbox.GroupLabel>Tags</Listbox.GroupLabel>
                    {tags.map((ref) => {
                      const i = itemIndex++;
                      return (
                        <Listbox.Item
                          key={ref.name}
                          id={`ref-option-${i}`}
                          ref={(el) => {
                            elementsRef.current[i] = el;
                          }}
                          active={activeIndex === i}
                          selected={ref.shortName === currentRef}
                          className={cn("text-xs", ref.shortName === currentRef && "font-medium")}
                          {...getItemProps({
                            onClick: () => {
                              onSelect(ref.shortName);
                              setOpen(false);
                              setFilter("");
                            },
                          })}
                        >
                          <Tag className="text-muted-foreground size-3 shrink-0" />
                          <span className="flex-1 truncate font-mono">{ref.shortName}</span>
                        </Listbox.Item>
                      );
                    })}
                  </Listbox.Group>
                )}
                {filtered.length === 0 && <Listbox.Empty />}
              </Listbox.ScrollArea>
            </Listbox.Content>
          </FloatingFocusManager>
        </FloatingPortal>
      )}
    </>
  );
}
