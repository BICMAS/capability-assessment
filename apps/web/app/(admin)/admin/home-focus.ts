export type HomeFocus = "overview" | "frameworks" | "assignments" | "plans" | "organisation";

export function homeFocus(params: {
  focus?: string;
  assignment?: string;
  plan?: string;
  planError?: string;
  assignmentError?: string;
  results?: string;
  review?: string;
  importError?: string;
  save?: string;
  versionId?: string;
  action?: string;
}): HomeFocus {
  if (
    params.focus === "frameworks" ||
    params.focus === "assignments" ||
    params.focus === "plans" ||
    params.focus === "organisation"
  ) {
    return params.focus;
  }
  if (params.plan || params.planError) {
    return "plans";
  }
  if (params.assignment && !params.results) {
    return "plans";
  }
  if (params.assignment || params.assignmentError || params.results) {
    return "assignments";
  }
  if (params.review || params.importError || params.save || params.versionId || params.action) {
    return "frameworks";
  }
  return "overview";
}
