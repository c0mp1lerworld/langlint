import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { expectNoA11yViolations } from "@/test/a11y";
import { Tooltip } from "./tooltip";

describe("Tooltip", () => {
  it("muestra la etiqueta al pasar el ratón y la oculta al salir", async () => {
    const user = userEvent.setup();
    render(
      <Tooltip label="texto de ayuda">
        <span>info</span>
      </Tooltip>,
    );

    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();

    await user.hover(screen.getByText("info"));
    expect(screen.getByRole("tooltip")).toHaveTextContent("texto de ayuda");

    await user.unhover(screen.getByText("info"));
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });

  it("se oculta con Escape desde el teclado", async () => {
    const user = userEvent.setup();
    render(
      <Tooltip label="ayuda">
        <button type="button">botón</button>
      </Tooltip>,
    );

    await user.tab();
    expect(screen.getByRole("tooltip")).toBeInTheDocument();

    await user.keyboard("{Escape}");
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });

  it("no tiene violaciones de accesibilidad", async () => {
    const { container } = render(
      <Tooltip label="ayuda">
        <span>info</span>
      </Tooltip>,
    );
    await expectNoA11yViolations(container);
  });
});
