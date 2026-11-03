export default function RoleProfilesLoading() {
  return (
    <div className="role-profile" aria-busy="true" aria-live="polite">
      <p className="empty-state">Loading role skills profiles…</p>
    </div>
  );
}
