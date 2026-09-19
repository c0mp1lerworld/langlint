import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { expectNoA11yViolations } from "@/test/a11y";
import { PracticeStatusBadge } from "./status-badge";

describe("PracticeStatusBadge", () => {
  it.each([
    ["draft", "Borrador"],
    ["analyzing", "Analizando"],
    ["completed", "Completada"],
    ["failed", "Fallida"],
  ] as const)("estado %s → etiqueta %s", (status, label) => {
    render(<PracticeStatusBadge status={status} />);
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it("no tiene violaciones de accesibilidad", async () => {
    const { container } = render(<PracticeStatusBadge status="completed" />);
    await expectNoA11yViolations(container);
  });
});
