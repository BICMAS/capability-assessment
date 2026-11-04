import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { ProposalConfirm } from "@/components/proposal-confirm";
import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import { listPublishedFrameworks } from "@/lib/capability-framework";
import { listAssessmentSessions } from "@/lib/generated/assessment-plan/plans";
import { listPeople, readPerson } from "@/lib/org-directory";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

import { confirmProposal } from "../../actions";
import { PersonRecord, loadCapabilities } from "../../team";

export default async function ManagerPersonPage({ params }: { params: Promise<{ personId: string }> }) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const { personId } = await params;
  const people = await settle(listPeople(token), "People");
  if (!people.ok) {
    return (
      <section className="team-person">
        <StatusMessage tone="error">{people.error}</StatusMessage>
      </section>
    );
  }
  const person = people.value.find((item) => item.id === personId);
  if (!person) {
    return (
      <section className="team-person">
        <EmptyState>You cannot open this person&apos;s record.</EmptyState>
      </section>
    );
  }
  const [profile, frameworks, sessions] = await Promise.all([
    settle(readPerson(token, person.id), "Profile"),
    settle(listPublishedFrameworks(token), "Frameworks"),
    settle(listAssessmentSessions(token, person.id), "Assessments"),
  ]);
  if (!profile.ok) {
    return (
      <section className="team-person">
        <StatusMessage tone="error">{profile.error}</StatusMessage>
      </section>
    );
  }
  const capabilities = frameworks.ok
    ? await loadCapabilities(
        token,
        person.id,
        profile.value.roles.map((role) => role.title),
        frameworks.value,
      )
    : [];

  return (
    <section className="team-person">
      {!frameworks.ok ? <StatusMessage tone="error">{frameworks.error}</StatusMessage> : null}
      {!sessions.ok ? <StatusMessage tone="error">{sessions.error}</StatusMessage> : null}
      {sessions.ok ? <ProposalConfirm personId={person.id} sessions={sessions.value} action={confirmProposal} /> : null}
      <PersonRecord
        personId={person.id}
        personName={person.name}
        roles={profile.value.roles.map((role) => role.title)}
        capabilities={capabilities}
      />
    </section>
  );
}
