import { PracticeList } from "@/features/practice";

export default function HomePage() {
  return (
    <main className="mx-auto w-full max-w-4xl p-6">
      <h1 className="mb-1 text-2xl font-semibold text-gray-900">LangLint</h1>
      <p className="mb-6 text-gray-600">Prácticas de escritura productiva con corrección de IA.</p>
      <PracticeList />
    </main>
  );
}
