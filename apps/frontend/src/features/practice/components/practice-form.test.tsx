import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useCreatePractice } from "@/lib/query/practices";
import { expectNoA11yViolations } from "@/test/a11y";
import { PracticeForm } from "./practice-form";

vi.mock("@/lib/query/practices", () => ({
  useCreatePractice: vi.fn(),
}));

const idle = { mutate: vi.fn(), isPending: false, isError: false, error: null };

function mockCreate(result: unknown): void {
  vi.mocked(useCreatePractice).mockReturnValue(result as ReturnType<typeof useCreatePractice>);
}

describe("PracticeForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCreate(idle);
  });

  it("valida los campos obligatorios antes de enviar", async () => {
    const user = userEvent.setup();
    render(<PracticeForm />);

    await user.click(screen.getByRole("button", { name: "Crear práctica" }));

    expect(await screen.findByText("El texto en español es obligatorio")).toBeInTheDocument();
    expect(screen.getByText("El borrador en inglés es obligatorio")).toBeInTheDocument();
    expect(screen.getByText("El verbo es obligatorio")).toBeInTheDocument();
  });

  it("envía los valores válidos al mutation", async () => {
    const mutate = vi.fn();
    mockCreate({ ...idle, mutate });

    const user = userEvent.setup();
    render(<PracticeForm />);

    await user.type(screen.getByLabelText("Texto en español"), "El perro escapó.");
    await user.type(screen.getByLabelText("Tu borrador en inglés"), "The dog escaped.");
    await user.type(screen.getByLabelText("Verbo"), "escape");
    await user.click(screen.getByRole("button", { name: "Crear práctica" }));

    expect(mutate).toHaveBeenCalledOnce();
    expect(mutate).toHaveBeenCalledWith({
      source_text: "El perro escapó.",
      draft_text: "The dog escaped.",
      target_rules: [{ verb: "escape", tense: "", note: "" }],
    });
  });

  it("no tiene violaciones de accesibilidad", async () => {
    const { container } = render(<PracticeForm />);
    await expectNoA11yViolations(container);
  });
});
