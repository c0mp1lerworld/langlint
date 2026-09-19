import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { expectNoA11yViolations } from "@/test/a11y";
import { errorPatternAggregate } from "@/test/fixtures";
import { ErrorPatternFrequency } from "./error-pattern-frequency";

describe("ErrorPatternFrequency", () => {
  it("muestra el estado vacío", () => {
    render(<ErrorPatternFrequency patterns={[]} />);
    expect(
      screen.getByText("Todavía no hay errores registrados en esta ventana."),
    ).toBeInTheDocument();
  });

  it("ordena por frecuencia y muestra etiqueta y veces", () => {
    render(
      <ErrorPatternFrequency
        patterns={[
          errorPatternAggregate({ code: "word_order", count: 1 }),
          errorPatternAggregate({ code: "tense_agreement", count: 3 }),
        ]}
      />,
    );

    const items = screen.getAllByRole("listitem");
    expect(items[0]).toHaveTextContent("tiempo/concordancia");
    expect(items[0]).toHaveTextContent("3 veces");
    expect(items[1]).toHaveTextContent("orden de palabras");
  });

  it("usa el singular cuando count es 1", () => {
    render(
      <ErrorPatternFrequency
        patterns={[errorPatternAggregate({ code: "false_friend", count: 1 })]}
      />,
    );
    expect(screen.getByText(/1 vez$/)).toBeInTheDocument();
  });

  it("no tiene violaciones de accesibilidad", async () => {
    const { container } = render(
      <ErrorPatternFrequency patterns={[errorPatternAggregate()]} />,
    );
    await expectNoA11yViolations(container);
  });
});
