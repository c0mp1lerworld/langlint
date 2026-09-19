"use client";

import Link from "next/link";
import { userMessage } from "@/lib/api/errors";
import { useListPractices } from "@/lib/query/practices";
import { PracticeStatusBadge } from "./status-badge";

export function PracticeList() {
  const { data, isPending, isError, error } = useListPractices();

  if (isPending) {
    return (
      <p role="status" className="text-gray-600">
        Cargando prácticas…
      </p>
    );
  }

  if (isError) {
    return (
      <p role="alert" className="rounded-md bg-red-50 p-3 text-red-800">
        {userMessage(error)}
      </p>
    );
  }

  if (data.items.length === 0) {
    return <p className="text-gray-600">Todavía no hay prácticas.</p>;
  }

  return (
    <ul className="divide-y divide-gray-200 rounded-lg border border-gray-200" aria-label="Prácticas">
      {data.items.map((practice) => (
        <li key={practice.id}>
          <Link
            href={`/practices/${practice.id}`}
            className="flex items-center justify-between gap-4 p-4 hover:bg-gray-50 focus-visible:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600"
          >
            <span className="min-w-0">
              <span className="block truncate text-gray-900">{practice.source_text}</span>
              <span className="block truncate text-sm text-gray-500">{practice.draft_text}</span>
            </span>
            <PracticeStatusBadge status={practice.status} />
          </Link>
        </li>
      ))}
    </ul>
  );
}
