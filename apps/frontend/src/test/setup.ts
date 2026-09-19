import "@testing-library/jest-dom/vitest";
import { createElement, type AnchorHTMLAttributes, type ReactNode } from "react";
import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";

const routerMock = vi.hoisted(() => ({
  push: vi.fn(),
  replace: vi.fn(),
  back: vi.fn(),
  forward: vi.fn(),
  refresh: vi.fn(),
  prefetch: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => routerMock,
  usePathname: () => "/",
  useSearchParams: () => new URLSearchParams(),
}));

vi.mock("next/link", () => ({
  default: ({
    href,
    children,
    ...rest
  }: AnchorHTMLAttributes<HTMLAnchorElement> & { href: unknown; children?: ReactNode }) =>
    createElement("a", { ...rest, href: typeof href === "string" ? href : "#" }, children),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});
