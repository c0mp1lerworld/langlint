import axe from "axe-core";
import { expect } from "vitest";

// The `region` rule requires page-level landmarks and `color-contrast` needs a
// real rendering engine; components are tested in isolation under jsdom, so both
// are disabled here (the app layout provides landmarks and the styles use AA
// contrast classes, verified manually).
export async function expectNoA11yViolations(container: Element): Promise<void> {
  const results = await axe.run(container, {
    rules: { region: { enabled: false }, "color-contrast": { enabled: false } },
  });
  expect(results.violations).toEqual([]);
}
