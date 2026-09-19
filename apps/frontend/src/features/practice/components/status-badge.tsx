import type { components } from "@/lib/api/gen";

type PracticeStatus = components["schemas"]["PracticeStatus"];

const STATUS_LABELS: Record<PracticeStatus, string> = {
  draft: "Borrador",
  analyzing: "Analizando",
  completed: "Completada",
  failed: "Fallida",
};

const STATUS_CLASSES: Record<PracticeStatus, string> = {
  draft: "bg-gray-100 text-gray-800",
  analyzing: "bg-blue-100 text-blue-900",
  completed: "bg-green-100 text-green-900",
  failed: "bg-red-100 text-red-900",
};

export function PracticeStatusBadge({ status }: { status: PracticeStatus }) {
  return (
    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_CLASSES[status]}`}>
      {STATUS_LABELS[status]}
    </span>
  );
}
