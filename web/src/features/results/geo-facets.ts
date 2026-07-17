import type { ResultColoFacet } from '@/features/results/types'

export interface GeoFacetOption {
  value: string
  label: string
  count: number
}

function aggregate(items: ResultColoFacet[], valueOf: (item: ResultColoFacet) => string): GeoFacetOption[] {
  const counts = new Map<string, number>()
  for (const item of items) {
    const value = valueOf(item)
    if (!value) continue
    counts.set(value, (counts.get(value) ?? 0) + item.count)
  }
  return [...counts.entries()]
    .map(([value, count]) => ({ value, label: value, count }))
    .sort((a, b) => a.label.localeCompare(b.label, 'en'))
}

export function buildGeoFacetOptions(
  facets: ResultColoFacet[],
  selected: { continent: string; country: string; city: string },
) {
  const countriesSource = selected.continent
    ? facets.filter((item) => item.continent === selected.continent)
    : facets
  const citiesSource = countriesSource.filter((item) => !selected.country || item.country === selected.country)
  const colosSource = citiesSource.filter((item) => !selected.city || item.city === selected.city)

  return {
    continents: aggregate(facets, (item) => item.continent),
    countries: aggregate(countriesSource, (item) => item.country),
    cities: aggregate(citiesSource, (item) => item.city),
    colos: [...colosSource]
      .sort((a, b) => a.code.localeCompare(b.code, 'en'))
      .map((item) => ({
        value: item.code,
        label: `${item.code} · ${item.city}, ${item.country}`,
        count: item.count,
      })),
  }
}

export function normalizeGeoSelection(
  facets: ResultColoFacet[],
  selected: { continent: string; country: string; city: string; colo: string },
) {
  const continentValid = !selected.continent || facets.some((item) => item.continent === selected.continent)
  if (!continentValid) return { continent: '', country: '', city: '', colo: '' }

  const continent = selected.continent
  const countryValid = !selected.country || facets.some((item) => (!continent || item.continent === continent) && item.country === selected.country)
  if (!countryValid) return { continent, country: '', city: '', colo: '' }

  const country = selected.country
  const cityValid = !selected.city || facets.some((item) => (!continent || item.continent === continent) && (!country || item.country === country) && item.city === selected.city)
  if (!cityValid) return { continent, country, city: '', colo: '' }

  const city = selected.city
  const colo = selected.colo && facets.some((item) => item.code === selected.colo && (!continent || item.continent === continent) && (!country || item.country === country) && (!city || item.city === city))
    ? selected.colo
    : ''
  return { continent, country, city, colo }
}
