"use client";

import Link from "next/link";
import { useFieldArray, useForm } from "react-hook-form";
import { userMessage } from "@/lib/api/errors";
import { useCreatePractice } from "@/lib/query/practices";
import { createPracticeSchema, type CreatePracticeInput } from "@/lib/schemas/practice";
import { zodResolver } from "@/lib/utils/zod-rhf";

const FIELD_CLASS =
  "mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-gray-900 shadow-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600";

const LABEL_CLASS = "block text-sm font-medium text-gray-900";
const ERROR_CLASS = "mt-1 text-sm text-red-700";

export function PracticeForm() {
  const create = useCreatePractice();
  const {
    register,
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<CreatePracticeInput>({
    resolver: zodResolver(createPracticeSchema),
    defaultValues: {
      source_text: "",
      draft_text: "",
      target_rules: [{ verb: "", tense: "", note: "" }],
    },
  });
  const { fields, append, remove } = useFieldArray({ control, name: "target_rules" });

  return (
    <form
      noValidate
      onSubmit={handleSubmit((values) => create.mutate(values))}
      className="space-y-6"
    >
      <h1 className="text-xl font-semibold text-gray-900">Nueva práctica</h1>

      <div>
        <label htmlFor="source_text" className={LABEL_CLASS}>
          Texto en español
        </label>
        <textarea
          id="source_text"
          rows={3}
          aria-invalid={errors.source_text !== undefined}
          aria-describedby={errors.source_text !== undefined ? "source_text-error" : undefined}
          className={FIELD_CLASS}
          {...register("source_text")}
        />
        {errors.source_text !== undefined ? (
          <p id="source_text-error" role="alert" className={ERROR_CLASS}>
            {errors.source_text.message}
          </p>
        ) : null}
      </div>

      <div>
        <label htmlFor="draft_text" className={LABEL_CLASS}>
          Tu borrador en inglés
        </label>
        <textarea
          id="draft_text"
          rows={3}
          aria-invalid={errors.draft_text !== undefined}
          aria-describedby={errors.draft_text !== undefined ? "draft_text-error" : undefined}
          className={FIELD_CLASS}
          {...register("draft_text")}
        />
        {errors.draft_text !== undefined ? (
          <p id="draft_text-error" role="alert" className={ERROR_CLASS}>
            {errors.draft_text.message}
          </p>
        ) : null}
      </div>

      <fieldset className="space-y-3">
        <legend className={LABEL_CLASS}>Reglas objetivo</legend>
        {errors.target_rules?.root !== undefined ? (
          <p role="alert" className={ERROR_CLASS}>
            {errors.target_rules.root.message}
          </p>
        ) : null}

        <ul className="space-y-3">
          {fields.map((field, index) => (
            <li
              key={field.id}
              className="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1fr_1fr_auto] sm:items-end"
            >
              <div>
                <label htmlFor={`target_rules.${index}.verb`} className="block text-xs text-gray-600">
                  Verbo
                </label>
                <input
                  id={`target_rules.${index}.verb`}
                  aria-invalid={errors.target_rules?.[index]?.verb !== undefined}
                  className={FIELD_CLASS}
                  {...register(`target_rules.${index}.verb`)}
                />
                {errors.target_rules?.[index]?.verb !== undefined ? (
                  <p role="alert" className={ERROR_CLASS}>
                    {errors.target_rules[index]?.verb?.message}
                  </p>
                ) : null}
              </div>

              <div>
                <label htmlFor={`target_rules.${index}.tense`} className="block text-xs text-gray-600">
                  Tiempo verbal
                </label>
                <input
                  id={`target_rules.${index}.tense`}
                  className={FIELD_CLASS}
                  {...register(`target_rules.${index}.tense`)}
                />
              </div>

              <div>
                <label htmlFor={`target_rules.${index}.note`} className="block text-xs text-gray-600">
                  Nota
                </label>
                <input
                  id={`target_rules.${index}.note`}
                  className={FIELD_CLASS}
                  {...register(`target_rules.${index}.note`)}
                />
              </div>

              <button
                type="button"
                onClick={() => remove(index)}
                disabled={fields.length === 1}
                className="justify-self-start rounded-md border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600 disabled:opacity-50"
              >
                Eliminar
              </button>
            </li>
          ))}
        </ul>

        <button
          type="button"
          onClick={() => append({ verb: "", tense: "", note: "" })}
          className="rounded-md border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600"
        >
          Añadir regla
        </button>
      </fieldset>

      {create.isError ? (
        <p role="alert" className="rounded-md bg-red-50 p-3 text-red-800">
          {userMessage(create.error)}
        </p>
      ) : null}

      <div className="flex items-center gap-3">
        <button
          type="submit"
          disabled={create.isPending}
          className="rounded-md bg-blue-700 px-4 py-2 text-sm font-medium text-white hover:bg-blue-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600 disabled:opacity-60"
        >
          {create.isPending ? "Creando…" : "Crear práctica"}
        </button>
        <Link
          href="/"
          className="rounded-md px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600"
        >
          Cancelar
        </Link>
      </div>
    </form>
  );
}
