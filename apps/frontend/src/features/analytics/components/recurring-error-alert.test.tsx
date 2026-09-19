import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { expectNoA11yViolations } from "@/test/a11y";
import { errorPatternAggregate } from "@/test/fixtures";
import { RecurringErrorAlert } from "./recurring-error-alert";

describe("RecurringErrorAlert", () => {
  it("sin patrón → no renderiza nada", () => {
    const { container } = render(<RecurringErrorAlert pattern={null} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("muestra el patrón más frecuente y su frecuencia", () => {
    render(
      <RecurringErrorAlert pattern={errorPatternAggregate({ code: "tense_agreement", count: 2 })} />,
    );
    expect(screen.getByRole("status")).toHaveTextContent(
      "Tu error más frecuente es tiempo/concordancia (2 veces).",
    );
  });

  it("no tiene violaciones de accesibilidad", async () => {
    const { container } = render(
      <RecurringErrorAlert pattern={errorPatternAggregate({ count: 1 })} />,
    );
    await expectNoA11yViolations(container);
  });
});
