import { PracticeDiff } from "@/features/practice";

export default async function PracticePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <main className="mx-auto w-full max-w-6xl p-6">
      <PracticeDiff id={id} />
    </main>
  );
}
