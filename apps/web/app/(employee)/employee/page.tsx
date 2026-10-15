import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { CapabilityInstrument, RequiredOnly } from "@/components/capability-instrument";
import { PathwayView } from "@/components/pathway";
import { listPublishedFrameworks, type FrameworkVersion } from "@/lib/capability-framework";
import { readLevel } from "@/lib/generated/assessment-gap/levels";
import { readPathway } from "@/lib/generated/development/pathways";
import { listAssessmentSessions, type AssessmentSession } from "@/lib/generated/assessment-plan/plans";
import { readProfile } from "@/lib/org-directory";
import { SESSION_COOKIE } from "@/lib/session";

import { submitSessionAnswers } from "./session-actions";

export default async function EmployeePage({
  searchParams,
}: {
  searchParams: Promise<{ sessionError?: string }>;
}) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const [profile, frameworks, sessions, params] = await Promise.all([
    readProfile(token),
    listPublishedFrameworks(token),
    loadSessions(token),
    searchParams,
  ]);
  const titles = new Set(profile.roles.map((role) => role.title));
  const ownSessions = sessions.filter((session) => session.personId === profile.id);

  const roleLine = profile.roles.map((role) => role.title).join(" · ");
  const frameworkLine = [...new Set(frameworks.map((framework) => `${framework.frameworkName} v${framework.versionNumber}`))].join(" · ");

  return (
    <>
      <h2 className="briefing-heading">
        {roleLine ? <span className="briefing-role">{roleLine}</span> : null}
        {roleLine && frameworkLine ? <span className="briefing-mark" aria-hidden="true" /> : null}
        {frameworkLine ? <span className="briefing-framework">{frameworkLine}</span> : null}
      </h2>
      {ownSessions.length > 0 ? (
        <section className="briefing-session">
          {params.sessionError ? <p className="login-error">{params.sessionError}</p> : null}
          {ownSessions.map((session) => (
            <article key={session.id} className="correspondence">
              <p className="correspondence-status">{session.status}</p>
              {session.status === "submitted" && session.confirmed
                ? (session.proposals ?? []).map((proposal) => (
                    <p key={proposal.capabilityId} className="provisional">
                      {proposalLabel(proposal.source)} {proposal.percentage} · Provisional
                    </p>
                  ))
                : null}
              {session.status === "submitted" ? (
                session.skills.map((skill, index) => (
                  <div key={`${session.id}-${index}`} className="letter">
                    <p className="letter-instructions">{skill.instructions}</p>
                    {skill.items.map((item) => (
                      <div key={item.id} className="letter-exchange">
                        <p className="letter-prompt">{item.prompt}</p>
                        <p className="letter-answer">{item.answer}</p>
                      </div>
                    ))}
                  </div>
                ))
              ) : (
                <form action={submitSessionAnswers} className="framework-form">
                  <input type="hidden" name="sessionId" value={session.id} />
                  {session.skills.map((skill, index) => (
                    <div key={`${session.id}-${index}`} className="letter">
                      <p className="letter-instructions">{skill.instructions}</p>
                      {skill.items.map((item) => (
                        <label key={item.id} className="letter-prompt">
                          {item.prompt}
                          <textarea name={`answer.${item.id}`} required maxLength={8000} defaultValue={item.answer} />
                        </label>
                      ))}
                    </div>
                  ))}
                  <button className="primary" type="submit">
                    Submit answers
                  </button>
                </form>
              )}
            </article>
          ))}
        </section>
      ) : null}
      <div className="capability-list">
        {frameworks.map((framework) => (
          <FrameworkLevels key={framework.id} token={token} personId={profile.id} framework={framework} roleTitles={titles} />
        ))}
      </div>
    </>
  );
}

function proposalLabel(source: string): string {
  switch (source) {
    case "knowledge":
      return "Knowledge";
    case "situational_judgement":
      return "Situational judgement";
    case "ai_interview":
      return "AI interview";
    default:
      return source;
  }
}

async function loadSessions(token: string): Promise<AssessmentSession[]> {
  try {
    return await listAssessmentSessions(token);
  } catch {
    return [];
  }
}

async function FrameworkLevels({
  token,
  personId,
  framework,
  roleTitles,
}: {
  token: string;
  personId: string;
  framework: FrameworkVersion;
  roleTitles: Set<string>;
}) {
  const requirements = framework.requirements.filter((requirement) => {
    const role = framework.roles.find((item) => item.id === requirement.roleId);
    return role !== undefined && roleTitles.has(role.name);
  });
  const rows = await Promise.all(
    requirements.map(async (requirement) => {
      const capability = framework.capabilities.find((item) => item.id === requirement.capabilityId);
      const [explanation, pathway] = await Promise.all([
        readLevel({
          token,
          personId,
          frameworkVersionId: framework.id,
          capabilityCode: requirement.capabilityId,
        }),
        readPathway({ token, personId, capabilityId: requirement.capabilityId }),
      ]);
      return { requirement, capability, explanation, pathway };
    }),
  );

  const ordered = [...rows].sort((left, right) => Number(right.explanation !== null) - Number(left.explanation !== null));

  return (
    <>
      {ordered.map(({ requirement, capability, explanation, pathway }) => (
        <div key={requirement.id} className="primary-capability">
          {explanation ? (
            <CapabilityInstrument name={capability?.name ?? "Capability"} explanation={explanation} />
          ) : (
            <RequiredOnly name={capability?.name ?? "Capability"} requiredLevel={requirement.requiredLevel} />
          )}
          {pathway ? <PathwayView pathway={pathway} /> : null}
        </div>
      ))}
    </>
  );
}
