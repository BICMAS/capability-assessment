"use client";

import { useActionState } from "react";

import { practiseScenario, type PracticeResult } from "@/features/role-profiles/insight-actions";
import { practiceScenarios } from "@/lib/capability-insights";

const empty: PracticeResult = { feedback: "", recorded: false, voice: "unconfigured", error: "" };

export function PracticeForm({ profileId, people }: { profileId: string; people: string[] }) {
  const [result, action, pending] = useActionState(practiseScenario, empty);
  return (
    <form className="framework-form" action={action}>
      <input type="hidden" name="profileId" value={profileId} />
      <h4>Text practice prototype</h4>
      <p className="portal-note">This is a text simulation. No voice or avatar service is configured. A transcript is stored only when you choose to record it.</p>
      {people.length === 0 ? <p className="empty-state">Add an assessed person before recording a practice session.</p> : null}
      <label>
        Employee
        <select name="personId" required defaultValue="">
          <option value="">Choose a person</option>
          {people.map((personId) => (
            <option key={personId} value={personId}>
              {personId}
            </option>
          ))}
        </select>
      </label>
      <label>
        Scenario
        <select name="scenarioKey" required defaultValue="sales_negotiation">
          {practiceScenarios.map(([key, title]) => (
            <option key={key} value={key}>
              {title}
            </option>
          ))}
        </select>
      </label>
      <label>
        What would you say?
        <textarea name="message" required maxLength={2000} />
      </label>
      <label>
        <input type="checkbox" name="record" value="yes" /> Record this transcript for the selected person
      </label>
      <button className="primary" type="submit" disabled={pending}>
        Run text practice
      </button>
      {result.error ? <p className="status-message is-error" role="alert">{result.error}</p> : null}
      {result.feedback ? (
        <p className="portal-note" role="status">
          {result.feedback} {result.recorded ? "The transcript was stored." : "The transcript was not stored."} Voice: {result.voice}.
        </p>
      ) : null}
    </form>
  );
}
