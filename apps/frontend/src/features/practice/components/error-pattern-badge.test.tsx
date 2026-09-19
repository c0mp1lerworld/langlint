import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import type { components } from "@/lib/api/gen";
import { expectNoA11yViolations } from "@/test/a11y";
import { ErrorPatternBadge } from "./error-pattern-badge";

type ErrorPattern = components["schemas"]["ErrorPattern"];

const pattern: ErrorPattern = {
  code: "tense_agreement",
  severity: "critical",
  note: "Usa el pasado simple de 'go'.",
};

describe("ErrorPatternBadge", () => {
  it("muestra la etiqueta del código y la severidad", () => {
    render(<ErrorPatternBadge pattern={pattern} />);
    expect(screen.getByText("tiempo/concordancia")).toBeInTheDocument();
    expect(screen.getByText(/severidad crítico/)).toBeInTheDocument();
  });

  it("expone la nota en el tooltip", async () => {
    const user = userEvent.setup();
    render(<ErrorPatternBadge pattern={pattern} />);

    await user.hover(screen.getByText("tiempo/concordancia"));
    expect(screen.getByRole("tooltip")).toHaveTextContent("Usa el pasado simple de 'go'.");
  });

  it("no tiene violaciones de accesibilidad", async () => {
    const { container } = render(<ErrorPatternBadge pattern={pattern} />);
    await expectNoA11yViolations(container);
  });
});
