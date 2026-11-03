export default function AdminLoading() {
  return (
    <div className="home-dashboard home-loading" aria-busy="true" aria-live="polite">
      <p className="home-welcome">Loading your organisation overview…</p>
      <div className="home-kpi-grid">
        <div className="home-kpi home-skeleton" />
        <div className="home-kpi home-skeleton" />
        <div className="home-kpi home-skeleton" />
        <div className="home-kpi home-skeleton" />
      </div>
      <div className="home-split">
        <div className="home-panel home-skeleton home-skeleton-tall" />
        <div className="home-side">
          <div className="home-panel home-skeleton" />
          <div className="home-panel home-skeleton" />
        </div>
      </div>
    </div>
  );
}
