import { useMutation } from "@apollo/client/react";
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
import { Settings2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import type { FragmentType } from "@/__generated__/fragment-masking";
import { graphql } from "@/__generated__/gql";
import { BugDetailDocument } from "@/__generated__/graphql";
import { LabelBadge, LABEL_FIELDS_FRAGMENT } from "@/components/shared/label-badge";
import { SectionHeading } from "@/components/shared/section-heading";
import * as Listbox from "@/components/ui/listbox";
import { toastManager } from "@/components/ui/toast";
import { useAuth } from "@/lib/auth";

const BUG_CHANGE_LABELS_MUTATION = graphql(`
  mutation BugChangeLabels($input: BugChangeLabelInput) {
    bugChangeLabels(input: $input) {
      bug {
        id
        labels {
          name
          ...LabelFields
        }
      }
    }
  }
`);

interface LabelEditorProps {
  bugPrefix: string;
  currentLabels: Array<{ name: string } & FragmentType<typeof LABEL_FIELDS_FRAGMENT>>;
  /** Current repo slug, passed as `repoRef` to the mutations. */
  repo?: string | null;
  /** Pre-fetched valid labels for the repository. */
  validLabels: Array<
    { name: string; color: { R: number; G: number; B: number } } & FragmentType<
      typeof LABEL_FIELDS_FRAGMENT
    >
  >;
}

// Gear-icon popover in the BugDetailPage sidebar for adding/removing labels.
// Loads all valid labels from the repo; the labels toggled while it is open
// are applied together by a single bugChangeLabels when it closes.  Hidden in
// read-only mode.
export function LabelEditor({ bugPrefix, currentLabels, repo, validLabels }: LabelEditorProps) {
  const { user } = useAuth();
  const [changeLabels, { loading: saving }] = useMutation(BUG_CHANGE_LABELS_MUTATION, {
    refetchQueries: [BugDetailDocument],
    errorPolicy: "all",
  });

  const currentNames = new Set(currentLabels.map((l) => l.name));

  // whether each label toggled is wanted, so that applying them leaves the
  // labels others changed meanwhile as they are
  const [toggled, setToggled] = useState<ReadonlyMap<string, boolean>>(new Map());
  const isSelected = (name: string) => toggled.get(name) ?? currentNames.has(name);

  function toggleLabel(name: string) {
    setToggled((prev) => new Map(prev).set(name, !(prev.get(name) ?? currentNames.has(name))));
  }

  function applyLabels() {
    const wanted = [...toggled];
    const added = wanted.filter(([name, on]) => on && !currentNames.has(name)).map(([n]) => n);
    const removed = wanted.filter(([name, on]) => !on && currentNames.has(name)).map(([n]) => n);
    // cleared now rather than on success, to keep the toggles made meanwhile
    // out of this change
    setToggled(new Map());
    if (added.length === 0 && removed.length === 0) return;
    void changeLabels({
      variables: { input: { prefix: bugPrefix, repoRef: repo, added, Removed: removed } },
    }).then((result) => {
      if (!result.error) return;
      // back for the next opening, unless toggled again meanwhile
      setToggled((prev) => new Map([...wanted, ...prev]));
      toastManager.add({ title: "Failed to save the labels", description: result.error.message });
    });
  }

  // Leaving the bug with the picker open applies its choices, like closing it
  // does.  Through a ref to the latest applyLabels: the one of the first render
  // knows no choices.
  const applyOnLeave = useRef(applyLabels);
  useEffect(() => {
    applyOnLeave.current = applyLabels;
  });
  useEffect(
    () => () => {
      applyOnLeave.current();
    },
    [],
  );

  // floating-ui state
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const elementsRef = useRef<(HTMLElement | null)[]>([]);

  const { refs, floatingStyles, context } = useFloating({
    open,
    onOpenChange: (next) => {
      setOpen(next);
      if (!next) applyLabels();
    },
    placement: "bottom-end",
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
  });

  const { getReferenceProps, getFloatingProps, getItemProps } = useInteractions([
    click,
    dismiss,
    role,
    listNav,
  ]);

  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        <SectionHeading className="mb-0">Labels</SectionHeading>
        {user && validLabels.length > 0 && (
          <button
            ref={refs.setReference}
            // one change at a time, so that a failed one restores no stale choice
            disabled={saving}
            className="text-muted-foreground hover:text-foreground disabled:opacity-50"
            {...getReferenceProps()}
          >
            <Settings2 className="size-3.5" />
          </button>
        )}
      </div>

      {open && (
        <FloatingPortal>
          <FloatingFocusManager context={context} modal={false}>
            <Listbox.Content
              // oxlint-disable-next-line react/refs -- floating-ui's ref setter, not a .current read
              ref={refs.setFloating}
              style={floatingStyles}
              className="w-56"
              {...getFloatingProps()}
            >
              <div className="text-muted-foreground mb-1 px-3 pt-2 text-xs font-medium">
                Apply labels
              </div>
              <Listbox.ScrollArea>
                {validLabels.map((label, i) => {
                  const active = isSelected(label.name);
                  return (
                    <Listbox.Item
                      key={label.name}
                      ref={(el) => {
                        elementsRef.current[i] = el;
                      }}
                      active={activeIndex === i}
                      selected={active}
                      tabIndex={activeIndex === i ? 0 : -1}
                      {...getItemProps({
                        onClick: () => toggleLabel(label.name),
                      })}
                    >
                      <span
                        className={`size-2 rounded-full border-2 transition-colors ${
                          active
                            ? "border-transparent"
                            : "border-muted-foreground/40 bg-transparent"
                        }`}
                        style={
                          active
                            ? {
                                backgroundColor: `rgb(${label.color.R},${label.color.G},${label.color.B})`,
                              }
                            : {}
                        }
                      />
                      <LabelBadge label={label} />
                    </Listbox.Item>
                  );
                })}
              </Listbox.ScrollArea>
            </Listbox.Content>
          </FloatingFocusManager>
        </FloatingPortal>
      )}

      {currentLabels.length === 0 ? (
        <p className="text-muted-foreground text-sm">None yet</p>
      ) : (
        <div className="flex flex-wrap gap-1">
          {currentLabels.map((label) => (
            <LabelBadge key={label.name} label={label} />
          ))}
        </div>
      )}
    </div>
  );
}
