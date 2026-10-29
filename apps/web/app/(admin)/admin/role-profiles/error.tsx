"use client";

export default function RoleProfilesError({ error, reset }: { error: Error; reset: () => void }) {
  void error;
  return (
    <div className="role-profile">
      <p role="alert">This page could not be loaded.</p>
      <button type="button" onClick={reset}>
        Try again
      </button>
    </div>
  );
}
