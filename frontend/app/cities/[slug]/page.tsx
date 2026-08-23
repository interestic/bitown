import { CityView } from "./CityView";

export default async function CityPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  return <CityView slug={slug} />;
}
