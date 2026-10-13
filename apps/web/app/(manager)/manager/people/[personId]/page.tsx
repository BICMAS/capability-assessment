import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { listPublishedFrameworks } from "@/lib/capability-framework";
import { listPeople, readPerson } from "@/lib/org-directory";
import { SESSION_COOKIE } from "@/lib/session";

import { PersonRecord, loadCapabilities } from "../../team";

export default async function ManagerPersonPage({ params }: { params: Promise<{ personId: string }> }) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const { personId } = await params;
  const people = await listPeople(token);
  const person = people.find((item) => item.id === personId);
  if (!person) {
    return (
      <section className="team-person">
        <p>You cannot open this person&apos;s record.</p>
      </section>
    );
  }
  const [profile, frameworks] = await Promise.all([readPerson(token, person.id), listPublishedFrameworks(token)]);
  const capabilities = await loadCapabilities(
    token,
    person.id,
    profile.roles.map((role) => role.title),
    frameworks,
  );

  return (
    <section className="team-person">
      <PersonRecord
        personId={person.id}
        personName={person.name}
        roles={profile.roles.map((role) => role.title)}
        capabilities={capabilities}
      />
    </section>
  );
}
