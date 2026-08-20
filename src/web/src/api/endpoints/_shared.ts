// Helpers shared by more than one endpoint module. Keep this small — anything
// used by a single domain belongs in that domain's file.

/** Appends a form field only when it has non-whitespace content. */
export function appendOptionalFormValue(formData: FormData, key: string, value?: string | null) {
  const trimmed = value?.trim()
  if (trimmed) formData.append(key, trimmed)
}
