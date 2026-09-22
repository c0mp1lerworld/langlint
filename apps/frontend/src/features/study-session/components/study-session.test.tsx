import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useCreateStudySession } from "@/lib/query/study-sessions";
import { expectNoA11yViolations } from "@/test/a11y";
import { StudySession } from "./study-session";

vi.mock("@/lib/query/study-sessions", () => ({
  useCreateStudySession: vi.fn(),
}));

const idle = { mutate: vi.fn(), isPending: false, isError: false, error: null };

function fakeMutation<T>(response: unknown): T {
  return {
    ...idle,
    mutate: vi.fn((_input: unknown, options?: { onSuccess?: (value: unknown) => void }) => {
      options?.onSuccess?.(response);
    }),
  } as unknown as T;
}

type CreateMutation = ReturnType<typeof useCreateStudySession>;

const SESSION = {
  id: "s1",
  user_id: "u1",
  theory: "El orden de palabras en inglés es SVO.",
  weaknesses: [{ code: "word_order", severity: "moderate", count: 6, last_seen_at: "2026-09-22T10:00:00Z" }],
  traps: [{ code: "word_order", description: "Colocar el verbo al final." }],
  exercises: [{ kind: "fill", prompt: "I ___ (never) have seen.", answer: "have never" }],
  status: "generated",
  created_at: "2026-09-22T10:00:00Z",
};

describe("StudySession", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("genera y muestra la sesión de estudio", async () => {
    vi.mocked(useCreateStudySession).mockReturnValue(fakeMutation<CreateMutation>(SESSION));

    const user = userEvent.setup();
    render(<StudySession />);

    await user.click(screen.getByRole("button", { name: "Generar sesión de estudio" }));

    expect(screen.getByText("El orden de palabras en inglés es SVO.")).toBeInTheDocument();
    expect(screen.getByText(/Colocar el verbo al final\./)).toBeInTheDocument();
    expect(screen.getByText(/I ___ \(never\) have seen\./)).toBeInTheDocument();
  });

  it("envía la ventana elegida al generar", async () => {
    const mutation = fakeMutation<CreateMutation>(SESSION);
    vi.mocked(useCreateStudySession).mockReturnValue(mutation);

    const user = userEvent.setup();
    render(<StudySession />);

    await user.click(screen.getByRole("button", { name: "Mes" }));
    await user.click(screen.getByRole("button", { name: "Generar sesión de estudio" }));

    expect(mutation.mutate).toHaveBeenCalledWith({ window: "month" }, expect.anything());
  });

  it("no tiene violaciones de accesibilidad", async () => {
    vi.mocked(useCreateStudySession).mockReturnValue(fakeMutation<CreateMutation>(SESSION));

    const user = userEvent.setup();
    const { container } = render(<StudySession />);
    await user.click(screen.getByRole("button", { name: "Generar sesión de estudio" }));
    await expectNoA11yViolations(container);
  });
});
