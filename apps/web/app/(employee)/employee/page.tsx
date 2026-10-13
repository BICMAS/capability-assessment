import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { CapabilityInstrument, RequiredOnly } from "@/components/capability-instrument";
import { PathwayView } from "@/components/pathway";
import { listPublishedFrameworks, type FrameworkVersion } from "@/lib/capability-framework";
import { readLevel } from "@/lib/generated/assessment-gap/levels";
import { readPathway } from "@/lib/generated/development/pathways";
import { readProfile } from "@/lib/org-directory";
import { SESSION_COOKIE } from "@/lib/session";

export default async function EmployeePage() {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const [profile, frameworks] = await Promise.all([readProfile(token), listPublishedFrameworks(token)]);
  const titles = new Set(profile.roles.map((role) => role.title));

  return (
    <section className="capability-board">
      <h2>Role</h2>
      <ul>
        {profile.roles.map((role) => (
          <li key={role.id}>{role.title}</li>
        ))}
      </ul>
      <h2>Published framework</h2>
      {frameworks.map((framework) => (
        <article key={framework.id}>
          <h3>
            {framework.frameworkName} v{framework.versionNumber}
          </h3>
          <FrameworkLevels token={token} personId={profile.id} framework={framework} roleTitles={titles} />
        </article>
      ))}
    </section>
  );
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
    <div className="capability-list">
      {ordered.map(({ requirement, capability, explanation, pathway }) => (
        <div key={requirement.id} className={explanation ? "primary-capability" : undefined}>
          {explanation ? (
            <CapabilityInstrument name={capability?.name ?? "Capability"} explanation={explanation} />
          ) : (
            <RequiredOnly name={capability?.name ?? "Capability"} requiredLevel={requirement.requiredLevel} />
          )}
          {pathway ? <PathwayView pathway={pathway} /> : null}
        </div>
      ))}
    </div>
  );
}
