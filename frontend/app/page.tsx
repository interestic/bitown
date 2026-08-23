import Link from "next/link";

const API_BASE = process.env.NEXT_PUBLIC_BITOWN_API_URL || "http://localhost:8080";

type City = {
  slug: string;
  name: string;
  country_code: string;
  pop: number;
};

async function loadRankings(): Promise<City[]> {
  try {
    const res = await fetch(`${API_BASE}/api/rankings`, { next: { revalidate: 60 } });
    if (!res.ok) return [];
    return res.json();
  } catch {
    return [];
  }
}

export default async function HomePage() {
  const cities = await loadRankings();
  return (
    <main>
      <h1>bitown</h1>
      <p className="lede">Visit a city, click once a day, and watch it grow.</p>
      <section className="panel">
        <h2>Top cities</h2>
        {cities.length === 0 ? (
          <p className="lede">No cities yet. Create one via the API, then open <code>/cities/&lt;slug&gt;</code>.</p>
        ) : (
          <ul className="home-list">
            {cities.map((c) => (
              <li key={c.slug}>
                <Link href={`/cities/${c.slug}`}>
                  <strong>{c.name}</strong> · {c.country_code} · pop {c.pop}
                </Link>
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
}
