"use client";

import { useCallback, useEffect, useMemo, useState } from "react";

const API_BASE = process.env.NEXT_PUBLIC_BITOWN_API_URL || "http://localhost:8080";

const SECTORS = ["pop", "ind", "tra", "sec", "env", "com"] as const;
type Sector = (typeof SECTORS)[number];

const LOCK: Record<Sector, number> = {
  pop: 0,
  ind: 50,
  tra: 100,
  sec: 300,
  env: 500,
  com: 1000,
};

type Metrics = {
  income: number;
  unemployment: number;
  roads: number;
  pollution: number;
  crime: number;
};

type City = {
  slug: string;
  name: string;
  country_code: string;
  pop: number;
  ind: number;
  tra: number;
  sec: number;
  env: number;
  com: number;
  metrics?: Metrics;
};

type EventRow = {
  id: number;
  event_type: string;
  delta: { sector?: string; delta?: number };
  created_at: string;
};

export function CityView({ slug }: { slug: string }) {
  const [city, setCity] = useState<City | null>(null);
  const [events, setEvents] = useState<EventRow[]>([]);
  const [status, setStatus] = useState("");
  const [err, setErr] = useState(false);
  const [busy, setBusy] = useState(false);
  const [mapTick, setMapTick] = useState(0);

  const mapSrc = useMemo(
    () => `${API_BASE}/api/cities/${encodeURIComponent(slug)}/map.png?t=${mapTick}`,
    [slug, mapTick],
  );

  const load = useCallback(async () => {
    const [cityRes, eventsRes] = await Promise.all([
      fetch(`${API_BASE}/api/cities/${encodeURIComponent(slug)}`),
      fetch(`${API_BASE}/api/cities/${encodeURIComponent(slug)}/events`),
    ]);
    if (!cityRes.ok) {
      setStatus("City not found");
      setErr(true);
      return;
    }
    setCity(await cityRes.json());
    if (eventsRes.ok) {
      setEvents(await eventsRes.json());
      setErr(false);
    } else {
      setEvents([]);
      setStatus("City loaded; event feed unavailable");
      setErr(true);
    }
  }, [slug]);

  useEffect(() => {
    void load();
  }, [load]);

  async function support(sector: Sector) {
    if (busy) return;
    setBusy(true);
    setStatus("Supporting…");
    setErr(false);
    try {
      const res = await fetch(`${API_BASE}/api/cities/${encodeURIComponent(slug)}/support`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sector }),
      });
      const raw = await res.text();
      let body: {
        already_voted?: boolean;
        message?: string;
        city?: City;
      } = {};
      if (raw) {
        try {
          body = JSON.parse(raw) as typeof body;
        } catch {
          body = { message: raw || `Error ${res.status}` };
        }
      } else if (!res.ok) {
        body = { message: `Error ${res.status}` };
      }
      if (res.status === 409 || body.already_voted) {
        setStatus("Already supported today");
        return;
      }
      if (!res.ok) {
        setStatus(typeof body.message === "string" ? body.message : `Error ${res.status}`);
        setErr(true);
        return;
      }
      if (body.city) setCity(body.city);
      setStatus(body.message || "+1!");
      setMapTick(Date.now());
      const eventsRes = await fetch(`${API_BASE}/api/cities/${encodeURIComponent(slug)}/events`);
      if (eventsRes.ok) setEvents(await eventsRes.json());
    } catch {
      setStatus("Network error");
      setErr(true);
    } finally {
      setBusy(false);
    }
  }

  if (!city) {
    return (
      <main>
        <h1>{slug}</h1>
        <p className={err ? "status err" : "status"} role="status">
          {status || "Loading…"}
        </p>
      </main>
    );
  }

  const metrics = city.metrics;
  const sharePath = `/cities/${city.slug}`;
  const badgeHref = `${API_BASE}/badge/${encodeURIComponent(city.slug)}.svg`;
  const mapHref = `${API_BASE}/api/cities/${encodeURIComponent(city.slug)}/map.png`;

  return (
    <main>
      <h1>{city.name}</h1>
      <p className="lede">
        {city.country_code} · slug <code>{city.slug}</code>
      </p>

      <div className="layout">
        <section className="map-panel" aria-label="City map">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={mapSrc} alt={`Map of ${city.name}`} loading="lazy" />
        </section>

        <aside>
          <section className="panel">
            <h2>Sectors</h2>
            <dl className="sectors">
              {SECTORS.map((s) => (
                <div className="row" key={s}>
                  <dt>{s}</dt>
                  <dd>{city[s]}</dd>
                </div>
              ))}
            </dl>
            <div className="actions" role="group" aria-label="Support sectors">
              {SECTORS.map((s) => {
                const locked = city.pop < LOCK[s];
                return (
                  <button
                    key={s}
                    type="button"
                    disabled={busy || locked}
                    title={locked ? `Unlocks at pop ${LOCK[s]}` : `Support ${s}`}
                    aria-label={locked ? `${s} locked until pop ${LOCK[s]}` : `Support ${s}`}
                    onClick={() => void support(s)}
                  >
                    +{s}
                  </button>
                );
              })}
            </div>
            {status ? (
              <p className={err ? "status err" : "status"} role="status">
                {status}
              </p>
            ) : null}
          </section>

          {metrics ? (
            <section className="panel" style={{ marginTop: "1rem" }}>
              <h2>Indicators</h2>
              <dl className="metrics">
                <div className="row">
                  <dt>Income</dt>
                  <dd>${metrics.income.toLocaleString()}</dd>
                </div>
                <div className="row">
                  <dt>Unemployment</dt>
                  <dd>{metrics.unemployment}%</dd>
                </div>
                <div className="row">
                  <dt>Roads</dt>
                  <dd>{metrics.roads}%</dd>
                </div>
                <div className="row">
                  <dt>Pollution</dt>
                  <dd>{metrics.pollution}%</dd>
                </div>
                <div className="row">
                  <dt>Crime</dt>
                  <dd>{metrics.crime}%</dd>
                </div>
              </dl>
            </section>
          ) : null}

          <section className="panel" style={{ marginTop: "1rem" }}>
            <h2>Share</h2>
            <div className="links">
              <a href={sharePath}>This page</a>
              <a href={badgeHref}>README badge</a>
              <a href={mapHref}>map.png</a>
            </div>
          </section>

          {events.length > 0 ? (
            <section className="panel" style={{ marginTop: "1rem" }}>
              <h2>Recent events</h2>
              <ul className="home-list">
                {events.slice(0, 8).map((ev) => (
                  <li key={ev.id}>
                    {ev.event_type}
                    {ev.delta?.sector ? ` · +${ev.delta.sector}` : ""} ·{" "}
                    {new Date(ev.created_at).toLocaleString()}
                  </li>
                ))}
              </ul>
            </section>
          ) : null}
        </aside>
      </div>
    </main>
  );
}
