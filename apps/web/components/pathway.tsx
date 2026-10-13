import type { Pathway } from "@/lib/generated/development/pathways";

export function PathwayView({ pathway }: { pathway: Pathway }) {
  return (
    <section className="pathway">
      <h4>Pathway</h4>
      <ol>
        {pathway.items.map((item, index) => {
          const learnLink = item.stage === "learn" && item.deepLink ? item.deepLink : null;
          const state = item.status === "complete" ? "complete" : item.status === "upcoming" ? "upcoming" : "active";
          return (
            <li key={item.id} className={`pathway-step pathway-${state}`}>
              <span className="pathway-index">{index + 1}</span>
              <div>
                <p className="pathway-title">{item.stage === "learn" ? `Learn: ${item.title}` : item.title}</p>
                {state === "complete" ? <p className="pathway-state">Complete</p> : null}
                {state === "upcoming" ? <p className="pathway-state">Upcoming</p> : null}
                {learnLink ? (
                  <a className="learn-link" href={learnLink}>
                    Learn
                  </a>
                ) : null}
              </div>
            </li>
          );
        })}
      </ol>
    </section>
  );
}
