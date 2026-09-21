import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useEvaluateQuizAnswer, useGenerateQuizQuestion } from "@/lib/query/quiz";
import { expectNoA11yViolations } from "@/test/a11y";
import { PracticeQuiz } from "./practice-quiz";

vi.mock("@/lib/query/quiz", () => ({
  useGenerateQuizQuestion: vi.fn(),
  useEvaluateQuizAnswer: vi.fn(),
}));

const idle = { mutate: vi.fn(), isPending: false, isError: false, error: null };

// fakeMutation mocks a react-query mutation whose mutate invokes onSuccess.
function fakeMutation<T>(response: unknown): T {
  return {
    ...idle,
    mutate: vi.fn((_input: unknown, options?: { onSuccess?: (value: unknown) => void }) => {
      options?.onSuccess?.(response);
    }),
  } as unknown as T;
}

type QuestionMutation = ReturnType<typeof useGenerateQuizQuestion>;
type AnswerMutation = ReturnType<typeof useEvaluateQuizAnswer>;

describe("PracticeQuiz", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("genera la pregunta y evalúa la respuesta", async () => {
    const generate = fakeMutation<QuestionMutation>({ kind: "fill", prompt: "I bet ___ the races." });
    const evaluate = fakeMutation<AnswerMutation>({ correct: true, feedback: "Bien hecho.", follow_up: "" });
    vi.mocked(useGenerateQuizQuestion).mockReturnValue(generate);
    vi.mocked(useEvaluateQuizAnswer).mockReturnValue(evaluate);

    const user = userEvent.setup();
    render(<PracticeQuiz practiceId="p1" fragmentIndex={0} />);

    await user.click(screen.getByRole("button", { name: "Ponlo en práctica" }));
    expect(screen.getByText("I bet ___ the races.")).toBeInTheDocument();

    await user.type(screen.getByLabelText("Tu respuesta"), "on");
    await user.click(screen.getByRole("button", { name: "Responder" }));

    expect(evaluate.mutate).toHaveBeenCalledWith(
      { fragmentIndex: 0, question: "I bet ___ the races.", answer: "on" },
      expect.anything(),
    );
    expect(screen.getByText("Bien hecho.")).toBeInTheDocument();
  });

  it("encadena la pregunta de seguimiento", async () => {
    vi.mocked(useGenerateQuizQuestion).mockReturnValue(
      fakeMutation<QuestionMutation>({ kind: "open", prompt: "¿Por qué?" }),
    );
    vi.mocked(useEvaluateQuizAnswer).mockReturnValue(
      fakeMutation<AnswerMutation>({ correct: false, feedback: "Casi.", follow_up: "¿Y por qué no 'in'?" }),
    );

    const user = userEvent.setup();
    render(<PracticeQuiz practiceId="p1" fragmentIndex={1} />);

    await user.click(screen.getByRole("button", { name: "Ponlo en práctica" }));
    await user.type(screen.getByLabelText("Tu respuesta"), "porque sí");
    await user.click(screen.getByRole("button", { name: "Responder" }));

    expect(screen.getByText("Casi.")).toBeInTheDocument();
    expect(screen.getByText("¿Y por qué no 'in'?")).toBeInTheDocument();
  });

  it("no tiene violaciones de accesibilidad", async () => {
    vi.mocked(useGenerateQuizQuestion).mockReturnValue(
      fakeMutation<QuestionMutation>({ kind: "fill", prompt: "I bet ___ the races." }),
    );
    vi.mocked(useEvaluateQuizAnswer).mockReturnValue(fakeMutation<AnswerMutation>({}));

    const user = userEvent.setup();
    const { container } = render(<PracticeQuiz practiceId="p1" fragmentIndex={0} />);
    await user.click(screen.getByRole("button", { name: "Ponlo en práctica" }));
    await expectNoA11yViolations(container);
  });
});
