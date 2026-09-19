"use client";

import { useId, useState, type ReactNode } from "react";

export function Tooltip({ label, children }: { label: string; children: ReactNode }) {
  const id = useId();
  const [open, setOpen] = useState(false);

  return (
    <span className="relative inline-flex">
      <span
        tabIndex={0}
        aria-describedby={open ? id : undefined}
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        onKeyDown={(event) => {
          if (event.key === "Escape") setOpen(false);
        }}
        className="cursor-help rounded outline-none focus-visible:ring-2 focus-visible:ring-blue-600"
      >
        {children}
      </span>
      <span
        id={id}
        role="tooltip"
        hidden={!open}
        className="absolute bottom-full left-1/2 z-10 mb-2 w-max max-w-xs -translate-x-1/2 rounded bg-gray-900 px-2 py-1 text-xs text-white shadow"
      >
        {label}
      </span>
    </span>
  );
}
