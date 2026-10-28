import { zodResolver } from "@hookform/resolvers/zod"
import { useMemo } from "react"
import { Controller, type Resolver, useForm } from "react-hook-form"
import { useTranslation } from "react-i18next"
import { z } from "zod"

import { Button } from "@/components/ui/Button"
import { Card, CardContent } from "@/components/ui/Card"
import { Input } from "@/components/ui/Input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/Select"
import type {
  IManga,
  IMangaCreate,
  IMangaUpdate,
  IPublicationStatus,
  ISource,
} from "@/types/models"
import { PUBLICATION_STATUSES } from "@/types/models"

export type MangaFormMode = "create" | "edit"

// Что уходит наружу: create — IMangaCreate, edit — IMangaUpdate (частичный
// PATCH: пустые опциональные поля конвертируются в undefined и не уходят).
export type MangaFormPayload<M extends MangaFormMode> = M extends "create"
  ? IMangaCreate
  : IMangaUpdate

// Значения формы — строки (удобно для input/select), конвертация в payload
// на отправке. Опциональные поля: "" = не задано.
interface MangaFormValues {
  source_id: string
  url: string
  title: string
  alt_titles?: string
  latest_chapter?: string
  chapters_count?: string
  publication_status?: string
}

// Тип t из useTranslation: динамические ключи словаря строго типизированы.
type Translate = ReturnType<typeof useTranslation>["t"]

function buildSchema(t: Translate) {
  return z.object({
    source_id: z.string().min(1, t("manga.form.requiredField")),
    url: z.string().trim().min(1, t("manga.form.requiredField")),
    title: z.string().trim().min(1, t("manga.form.requiredField")),
    alt_titles: z.string().optional(),
    latest_chapter: z.string().optional(),
    chapters_count: z
      .string()
      .refine((v) => v === "" || /^\d+$/.test(v), t("manga.form.invalidNumber"))
      .optional(),
    publication_status: z.string().optional(),
  })
}

function optionalText(value: string | undefined): string | undefined {
  const trimmed = value?.trim()
  return trimmed ? trimmed : undefined
}

// Пустые строки опциональных полей -> undefined: в JSON они не попадают,
// поэтому edit-PATCH меняет только заполненные поля.
function toPayload<M extends MangaFormMode>(mode: M, values: MangaFormValues): MangaFormPayload<M> {
  const shared = {
    title: values.title.trim(),
    alt_titles: optionalText(values.alt_titles),
    latest_chapter: optionalText(values.latest_chapter),
    chapters_count: values.chapters_count ? Number(values.chapters_count) : undefined,
    publication_status: values.publication_status || undefined,
  }
  if (mode === "create") {
    return {
      source_id: Number(values.source_id),
      url: values.url.trim(),
      ...shared,
    } as MangaFormPayload<M>
  }
  return shared as MangaFormPayload<M>
}

export interface MangaFormProps<M extends MangaFormMode = MangaFormMode> {
  mode: M
  sources: ISource[]
  initial?: IManga
  // Async-колбэк тоже подходит: void-возврат означает «результат не ждём»,
  // индикатор занятости — проп pending.
  onSubmit: (payload: MangaFormPayload<M>) => void
  onCancel?: () => void
  pending?: boolean
}

export function MangaForm<M extends MangaFormMode>({
  mode,
  sources,
  initial,
  onSubmit,
  onCancel,
  pending,
}: MangaFormProps<M>) {
  const { t } = useTranslation()
  const schema = useMemo(() => buildSchema(t), [t])

  const {
    register,
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<MangaFormValues>({
    resolver: zodResolver(schema) as Resolver<MangaFormValues>,
    defaultValues: {
      source_id: initial ? String(initial.source_id) : "",
      url: initial?.url ?? "",
      title: initial?.title ?? "",
      alt_titles: initial?.alt_titles ?? "",
      latest_chapter: initial?.latest_chapter ?? "",
      chapters_count: initial?.chapters_count == null ? "" : String(initial.chapters_count),
      publication_status: initial?.publication_status ?? "",
    },
  })

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">
        {mode === "create" ? t("manga.newTitle") : t("manga.editTitle")}
      </h1>
      <Card className="max-w-2xl">
        <CardContent>
          <form
            className="space-y-4"
            onSubmit={handleSubmit((values) => {
              void onSubmit(toPayload(mode, values) as MangaFormPayload<M>)
            })}
            noValidate
          >
            <div className="space-y-1.5">
              <label htmlFor="manga-form-source" className="block text-sm font-medium">
                {t("manga.form.sourceId")}
              </label>
              {/* Select на Base UI не фокусируется через register — значение
                  подключаем через Controller; null = опция-плейсхолдер. */}
              <Controller
                control={control}
                name="source_id"
                render={({ field }) => (
                  <Select
                    items={[
                      { value: null, label: t("manga.form.selectSource") },
                      ...sources.map((s) => ({ value: String(s.id), label: s.name })),
                    ]}
                    value={field.value || null}
                    onValueChange={(value) => field.onChange(value ?? "")}
                    disabled={mode === "edit" || pending}
                  >
                    <SelectTrigger id="manga-form-source" className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={null}>{t("manga.form.selectSource")}</SelectItem>
                      {sources.map((s) => (
                        <SelectItem key={s.id} value={String(s.id)}>
                          {s.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              />
              {errors.source_id && (
                <p className="text-sm text-destructive">{errors.source_id.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <label htmlFor="manga-form-url" className="block text-sm font-medium">
                {t("manga.form.url")}
              </label>
              <Input
                id="manga-form-url"
                placeholder="/manga/..."
                disabled={mode === "edit" || pending}
                {...register("url")}
              />
              {errors.url && <p className="text-sm text-destructive">{errors.url.message}</p>}
            </div>
            <div className="space-y-1.5">
              <label htmlFor="manga-form-title" className="block text-sm font-medium">
                {t("manga.form.title")}
              </label>
              <Input id="manga-form-title" disabled={pending} {...register("title")} />
              {errors.title && <p className="text-sm text-destructive">{errors.title.message}</p>}
            </div>
            <div className="space-y-1.5">
              <label htmlFor="manga-form-alt-titles" className="block text-sm font-medium">
                {t("manga.form.altTitles")}
              </label>
              <Input id="manga-form-alt-titles" disabled={pending} {...register("alt_titles")} />
            </div>
            <div className="space-y-1.5">
              <label htmlFor="manga-form-latest-chapter" className="block text-sm font-medium">
                {t("manga.form.latestChapter")}
              </label>
              <Input
                id="manga-form-latest-chapter"
                disabled={pending}
                {...register("latest_chapter")}
              />
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <label htmlFor="manga-form-chapters-count" className="block text-sm font-medium">
                  {t("manga.form.chaptersCount")}
                </label>
                <Input
                  id="manga-form-chapters-count"
                  type="number"
                  min={0}
                  step={1}
                  disabled={pending}
                  {...register("chapters_count")}
                />
                {errors.chapters_count && (
                  <p className="text-sm text-destructive">{errors.chapters_count.message}</p>
                )}
              </div>
              <div className="space-y-1.5">
                <label htmlFor="manga-form-status" className="block text-sm font-medium">
                  {t("manga.form.publicationStatus")}
                </label>
                <Controller
                  control={control}
                  name="publication_status"
                  render={({ field }) => (
                    <Select
                      items={[
                        { value: null, label: t("manga.form.statusNotSet") },
                        ...PUBLICATION_STATUSES.map((s) => ({
                          value: s,
                          label: t(`manga.status.${s}`),
                        })),
                      ]}
                      value={field.value || null}
                      onValueChange={(value) => field.onChange(value ?? "")}
                      disabled={pending}
                    >
                      <SelectTrigger id="manga-form-status" className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value={null}>{t("manga.form.statusNotSet")}</SelectItem>
                        {PUBLICATION_STATUSES.map((s: IPublicationStatus) => (
                          <SelectItem key={s} value={s}>
                            {t(`manga.status.${s}`)}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                />
              </div>
            </div>
            <div className="flex gap-2">
              <Button type="submit" disabled={pending}>
                {pending ? t("manga.form.saving") : t("manga.form.save")}
              </Button>
              {onCancel && (
                <Button type="button" variant="ghost" disabled={pending} onClick={onCancel}>
                  {t("manga.form.cancel")}
                </Button>
              )}
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
