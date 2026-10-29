export const maxOrganogramBytes = 5 * 1024 * 1024;

const extensions = [".png", ".jpg", ".jpeg", ".pdf", ".xls", ".xlsx"];

export function organogramUploadIssue(file: { name: string; size: number }): string | null {
  const name = file.name.trim().toLowerCase();
  if (file.size <= 0 || name === "") {
    return "Choose a PNG, JPG, PDF, XLS, or XLSX file.";
  }
  if (file.size > maxOrganogramBytes) {
    return "The file must be 5 MB or smaller.";
  }
  if (!extensions.some((extension) => name.endsWith(extension))) {
    return "Upload a PNG, JPG, PDF, XLS, or XLSX file.";
  }
  return null;
}
