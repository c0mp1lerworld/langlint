import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useListPractices } from "@/lib/query/practices";
import { expectNoA11yViolations } from "@/test/a11y";
import { practice } from "@/test/fixtures";
import { PracticeList } from "./practice-list";

vi.mock("@/lib/query/practices", () => ({
  useListPractices: vi.fn(),
}));

function mockList(result: unknown): void {
  vi.mocked(useListPractices).mockReturnValue(result as ReturnType<typeof useListPractices>);
}

describe("PracticeList", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("muestra el estado de carga", () => {
    mockList({ isPending: true, isError: false, error: null, data: undefined });
    render(<PracticeList />);
    expect(screen.getByRole("status")).toHaveTextContent("Cargando prácticas…");
  });

  it("muestra el estado de error mapeado a UX", () => {
    mockList({ isPending: false, isError: true, error: new Error("boom"), data: undefined });
    render(<PracticeList />);
    expect(screen.getByRole("alert")).toHaveTextContent("Ocurrió un error inesperado.");
  });

  it("muestra el estado vacío", () => {
    mockList({ isPending: false, isError: false, error: null, data: { items: [], total: 0 } });
    render(<PracticeList />);
    expect(screen.getByText("Todavía no hay prácticas.")).toBeInTheDocument();
  });

  it("renderiza las prácticas como enlaces al detalle", () => {
    mockList({
      isPending: false,
      isError: false,
      error: null,
      data: { items: [practice()], total: 1 },
    });
    render(<PracticeList />);

    expect(screen.getByText("El perro escapó.")).toBeInTheDocument();
    expect(screen.getByText("The dog escaped.")).toBeInTheDocument();
    expect(screen.getByText("Completada")).toBeInTheDocument();
    expect(screen.getByRole("link")).toHaveAttribute(
      "href",
      "/practices/11111111-1111-7111-8111-111111111111",
    );
  });

  it("no tiene violaciones de accesibilidad", async () => {
    mockList({
      isPending: false,
      isError: false,
      error: null,
      data: { items: [practice()], total: 1 },
    });
    const { container } = render(<PracticeList />);
    await expectNoA11yViolations(container);
  });
});
