import {
  useFloating,
  useClick,
  useDismiss,
  useRole,
  useListNavigation,
  useTypeahead,
  useInteractions,
  offset,
  flip,
  FloatingPortal,
  FloatingFocusManager,
} from "@floating-ui/react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { GitBranch, ChevronDown, Tag } from "lucide-react";
import { useRef, useState } from "react";

import { Button } from "./button";
import * as Listbox from "./listbox";

const meta = {
  component: Listbox.Content,
  parameters: { layout: "centered", a11y: { disable: true } },
} satisfies Meta<typeof Listbox.Content>;

export default meta;
type Story = StoryObj<typeof meta>;

// ── Simple single-select ─────────────────────────────────────────────────────

const fruits = ["Apple", "Banana", "Cherry", "Dragonfruit", "Elderberry", "Fig", "Grape"];

function SimpleSelect() {
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState("Cherry");
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const selectedIndex = fruits.indexOf(selected);

  const elementsRef = useRef<(HTMLElement | null)[]>([]);
  const labelsRef = useRef<(string | null)[]>([]);

  const { refs, floatingStyles, context } = useFloating({
    open,
    onOpenChange: setOpen,
    placement: "bottom-start",
    middleware: [offset(4), flip()],
  });

  const click = useClick(context);
  const dismiss = useDismiss(context);
  const role = useRole(context, { role: "listbox" });
  const listNav = useListNavigation(context, {
    listRef: elementsRef,
    activeIndex,
    selectedIndex,
    onNavigate: setActiveIndex,
    loop: true,
  });
  const typeahead = useTypeahead(context, {
    listRef: labelsRef,
    activeIndex,
    onMatch: setActiveIndex,
  });

  const { getReferenceProps, getFloatingProps, getItemProps } = useInteractions([
    click,
    dismiss,
    role,
    listNav,
    typeahead,
  ]);

  return (
    <>
      <Button
        ref={refs.setReference}
        variant="outline"
        className="min-w-[160px] justify-between"
        {...getReferenceProps()}
      >
        {selected}
        <ChevronDown className="text-muted-foreground size-3.5" />
      </Button>

      {open && (
        <FloatingPortal>
          <FloatingFocusManager context={context} modal={false}>
            <Listbox.Content ref={refs.setFloating} style={floatingStyles} {...getFloatingProps()}>
              <Listbox.ScrollArea>
                {fruits.map((fruit, i) => {
                  labelsRef.current[i] = fruit;
                  return (
                    <Listbox.Item
                      key={fruit}
                      ref={(el) => {
                        elementsRef.current[i] = el;
                      }}
                      active={activeIndex === i}
                      selected={selected === fruit}
                      tabIndex={activeIndex === i ? 0 : -1}
                      {...getItemProps({
                        onClick: () => {
                          setSelected(fruit);
                          setOpen(false);
                        },
                      })}
                    >
                      {fruit}
                    </Listbox.Item>
                  );
                })}
              </Listbox.ScrollArea>
            </Listbox.Content>
          </FloatingFocusManager>
        </FloatingPortal>
      )}
    </>
  );
}

export const SingleSelect: Story = {
  render: () => <SimpleSelect />,
};

// ── Multi-select with search ─────────────────────────────────────────────────

const allTags = [
  { name: "bug", color: "rgb(252, 41, 41)" },
  { name: "enhancement", color: "rgb(0, 150, 255)" },
  { name: "documentation", color: "rgb(0, 180, 80)" },
  { name: "help wanted", color: "rgb(255, 152, 0)" },
  { name: "good first issue", color: "rgb(124, 58, 237)" },
  { name: "duplicate", color: "rgb(120, 120, 120)" },
  { name: "wontfix", color: "rgb(180, 180, 180)" },
];

function MultiSelectWithSearch() {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<string[]>(["bug", "enhancement"]);
  const [activeIndex, setActiveIndex] = useState<number | null>(null);

  const elementsRef = useRef<(HTMLElement | null)[]>([]);
  const searchRef = useRef<HTMLInputElement>(null);

  const { refs, floatingStyles, context } = useFloating({
    open,
    onOpenChange(nextOpen) {
      setOpen(nextOpen);
      if (!nextOpen) setSearch("");
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

  const filtered = search.trim()
    ? allTags.filter((t) => t.name.toLowerCase().includes(search.toLowerCase()))
    : allTags;

  // Reset the active item when the filtered list size changes
  const [lastCount, setLastCount] = useState(filtered.length);
  if (filtered.length !== lastCount) {
    setLastCount(filtered.length);
    setActiveIndex(filtered.length > 0 ? 0 : null);
  }

  function toggle(name: string) {
    setSelected((prev) => (prev.includes(name) ? prev.filter((n) => n !== name) : [...prev, name]));
  }

  return (
    <>
      <Button
        ref={refs.setReference}
        variant="outline"
        className="min-w-[160px] justify-between"
        {...getReferenceProps()}
      >
        <Tag className="size-3.5" />
        Labels ({selected.length})
        <ChevronDown className="text-muted-foreground size-3.5" />
      </Button>

      {open && (
        <FloatingPortal>
          <FloatingFocusManager context={context} modal={false} initialFocus={searchRef}>
            <Listbox.Content ref={refs.setFloating} style={floatingStyles} {...getFloatingProps()}>
              <Listbox.Search
                ref={searchRef}
                placeholder="Search tags…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && activeIndex != null) {
                    e.preventDefault();
                    const tag = filtered[activeIndex];
                    if (tag) toggle(tag.name);
                  }
                }}
              />
              <Listbox.ScrollArea>
                {filtered.length === 0 && <Listbox.Empty>No tags found</Listbox.Empty>}
                {filtered.map((tag, i) => (
                  <Listbox.Item
                    key={tag.name}
                    ref={(el) => {
                      elementsRef.current[i] = el;
                    }}
                    active={activeIndex === i}
                    selected={selected.includes(tag.name)}
                    {...getItemProps({ onClick: () => toggle(tag.name) })}
                  >
                    <span
                      className="size-2.5 shrink-0 rounded-full"
                      style={{ backgroundColor: tag.color }}
                    />
                    {tag.name}
                  </Listbox.Item>
                ))}
              </Listbox.ScrollArea>
              {selected.length > 0 && (
                <Listbox.Footer>
                  <Listbox.FooterButton onClick={() => setSelected([])}>
                    Clear all
                  </Listbox.FooterButton>
                </Listbox.Footer>
              )}
            </Listbox.Content>
          </FloatingFocusManager>
        </FloatingPortal>
      )}
    </>
  );
}

export const MultiSelectSearch: Story = {
  render: () => <MultiSelectWithSearch />,
};

// ── Grouped select ───────────────────────────────────────────────────────────

const branches = ["main", "develop", "feature/auth"];
const tags = ["v1.0.0", "v1.1.0", "v2.0.0-rc1"];

function GroupedSelect() {
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState("main");
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

  const q = filter.toLowerCase();
  const filteredBranches = branches.filter((b) => b.toLowerCase().includes(q));
  const filteredTags = tags.filter((t) => t.toLowerCase().includes(q));
  const allFiltered = [...filteredBranches, ...filteredTags];

  // Reset the active item when the filtered list size changes
  const [lastCount, setLastCount] = useState(allFiltered.length);
  if (allFiltered.length !== lastCount) {
    setLastCount(allFiltered.length);
    setActiveIndex(allFiltered.length > 0 ? 0 : null);
  }

  let idx = 0;

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
        {selected}
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
              <Listbox.Search
                ref={searchRef}
                placeholder="Filter…"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && activeIndex != null) {
                    e.preventDefault();
                    const item = allFiltered[activeIndex];
                    if (item) {
                      setSelected(item);
                      setOpen(false);
                    }
                  }
                }}
                className="text-xs"
              />
              <Listbox.ScrollArea>
                {filteredBranches.length > 0 && (
                  <Listbox.Group>
                    <Listbox.GroupLabel>Branches</Listbox.GroupLabel>
                    {filteredBranches.map((b) => {
                      const i = idx++;
                      return (
                        <Listbox.Item
                          key={b}
                          ref={(el) => {
                            elementsRef.current[i] = el;
                          }}
                          active={activeIndex === i}
                          selected={selected === b}
                          className="font-mono text-xs"
                          {...getItemProps({
                            onClick: () => {
                              setSelected(b);
                              setOpen(false);
                            },
                          })}
                        >
                          <GitBranch className="text-muted-foreground size-3 shrink-0" />
                          {b}
                        </Listbox.Item>
                      );
                    })}
                  </Listbox.Group>
                )}
                {filteredTags.length > 0 && (
                  <Listbox.Group>
                    <Listbox.GroupLabel>Tags</Listbox.GroupLabel>
                    {filteredTags.map((t) => {
                      const i = idx++;
                      return (
                        <Listbox.Item
                          key={t}
                          ref={(el) => {
                            elementsRef.current[i] = el;
                          }}
                          active={activeIndex === i}
                          selected={selected === t}
                          className="font-mono text-xs"
                          {...getItemProps({
                            onClick: () => {
                              setSelected(t);
                              setOpen(false);
                            },
                          })}
                        >
                          <Tag className="text-muted-foreground size-3 shrink-0" />
                          {t}
                        </Listbox.Item>
                      );
                    })}
                  </Listbox.Group>
                )}
                {allFiltered.length === 0 && <Listbox.Empty />}
              </Listbox.ScrollArea>
            </Listbox.Content>
          </FloatingFocusManager>
        </FloatingPortal>
      )}
    </>
  );
}

export const Grouped: Story = {
  render: () => <GroupedSelect />,
};
