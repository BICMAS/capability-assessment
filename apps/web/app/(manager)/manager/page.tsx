import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { TeamDirectory } from "@/features/manager/team-directory";
import { loadTeamMemberSummaries } from "@/features/manager/team-data";
import { listPublishedFrameworks } from "@/lib/capability-framework";
import { listPeople } from "@/lib/org-directory";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export default async function ManagerPage() {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }

  const [people, frameworks] = await Promise.all([
    settle(listPeople(token), "People"),
    settle(listPublishedFrameworks(token), "Frameworks"),
  ]);

  const members =
    people.ok && frameworks.ok ? await loadTeamMemberSummaries(token, people.value, frameworks.value) : [];

  return <TeamDirectory people={people} frameworks={frameworks} members={members} />;
}
