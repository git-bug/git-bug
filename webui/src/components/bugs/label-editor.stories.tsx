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
import type { Meta, StoryObj } from "@storybook/react-vite";
import { Settings2 } from "lucide-react";
import { useState } from "react";
import { useRef } from "react";

import { withApollo, withCachedFragments } from "@/../.storybook/decorators";
import { makeFragmentData } from "@/__generated__/fragment-masking";
import { LabelBadge, LABEL_FIELDS_FRAGMENT } from "@/components/shared/label-badge";
import { SectionHeading } from "@/components/shared/section-heading";
import * as Listbox from "@/components/ui/listbox";

// The real LabelEditor depends on GraphQL mutations. For stories, we build a
// self-contained version with the same UI but local state instead of mutations.

const allLabelsData = [
  {
    __typename: "Label" as const,
    name: "bug",
    color: { __typename: "Color" as const, R: 252, G: 41, B: 41 },
  },
  {
    __typename: "Label" as const,
    name: "enhancement",
    color: { __typename: "Color" as const, R: 0, G: 150, B: 255 },
  },
  {
    __typename: "Label" as const,
    name: "documentation",
    color: { __typename: "Color" as const, R: 0, G: 180, B: 80 },
  },
  {
    __typename: "Label" as const,
    name: "help wanted",
    color: { __typename: "Color" as const, R: 255, G: 152, B: 0 },
  },
  {
    __typename: "Label" as const,
    name: "good first issue",
    color: { __typename: "Color" as const, R: 124, G: 58, B: 237 },
  },
];

const allLabels = allLabelsData.map((l) => ({
  ...l,
  ...makeFragmentData(l, LABEL_FIELDS_FRAGMENT),
}));

function LabelEditorDemo() {
  const [currentNames, setCurrentNames] = useState<Set<string>>(
    new Set([allLabelsData[0]!.name, allLabelsData[2]!.name]),
  );

  const currentLabels = allLabels.filter((l) => currentNames.has(l.name));

  function toggleLabel(label: { name: string }) {
    setCurrentNames((prev) => {
      const next = new Set(prev);
      if (next.has(label.name)) {
        next.delete(label.name);
      } else {
        next.add(label.name);
      }
      return next;
    });
  }

  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const elementsRef = useRef<(HTMLElement | null)[]>([]);

  const { refs, floatingStyles, context } = useFloating({
    open,
    onOpenChange: setOpen,
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
    <div className="w-64">
      <div className="mb-2 flex items-center justify-between">
        <SectionHeading className="mb-0">Labels</SectionHeading>
        <button
          ref={refs.setReference}
          className="text-muted-foreground hover:text-foreground"
          {...getReferenceProps()}
        >
          <Settings2 className="size-3.5" />
        </button>
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
                {allLabels.map((label, i) => {
                  const active = currentNames.has(label.name);
                  return (
                    <Listbox.Item
                      key={label.name}
                      ref={(el) => {
                        elementsRef.current[i] = el;
                      }}
                      active={activeIndex === i}
                      selected={active}
                      tabIndex={activeIndex === i ? 0 : -1}
                      {...getItemProps({ onClick: () => toggleLabel(label) })}
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

const meta = {
  component: LabelEditorDemo,
  decorators: [
    withApollo,
    withCachedFragments(
      ...allLabelsData.map((l) => [LABEL_FIELDS_FRAGMENT, "LabelFields", l] as const),
    ),
  ],
  parameters: { layout: "centered", a11y: { disable: true } },
} satisfies Meta<typeof LabelEditorDemo>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
