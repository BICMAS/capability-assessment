import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { listPublishedFrameworks } from "@/lib/capability-framework";
import { listPeople, readPerson } from "@/lib/org-directory";
import { SESSION_COOKIE } from "@/lib/session";

import { TeamRow, loadCapabilities } from "./team";

export default async function ManagerPage() {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const [people, frameworks] = await Promise.all([listPeople(token), listPublishedFrameworks(token)]);
  const members = await Promise.all(
    people.map(async (person) => {
      const profile = await readPerson(token, person.id);
      const capabilities = await loadCapabilities(
        token,
        person.id,
        profile.roles.map((role) => role.title),
        frameworks,
      );
      return { person, roles: profile.roles.map((role) => role.title), capabilities };
    }),
  );

  return (
    <section className="team-home">
      <ul>
        {members.map(({ person, roles, capabilities }) => (
          <li key={person.id}>
            <TeamRow personId={person.id} personName={person.name} roles={roles} capabilities={capabilities} />
          </li>
        ))}
      </ul>
    </section>
  );
}
