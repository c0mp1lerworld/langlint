import Link from "next/link";
import { PracticeList } from "@/features/practice";

export default function HomePage() {
  return (
    <main className="mx-auto w-full max-w-4xl p-6">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">LangLint</h1>
          <p className="text-gray-600">Prácticas de escritura productiva con corrección de IA.</p>
        </div>
        <Link
          href="/practices/new"
          className="rounded-md bg-blue-700 px-4 py-2 text-sm font-medium text-white hover:bg-blue-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600"
        >
          Nueva práctica
        </Link>
      </div>
      <PracticeList />
    </main>
  );
}
