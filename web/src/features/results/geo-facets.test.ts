import { describe, expect, it } from 'vitest'
import { buildGeoFacetOptions, normalizeGeoSelection } from '@/features/results/geo-facets'
import type { ResultColoFacet } from '@/features/results/types'

const facets: ResultColoFacet[] = [
  { code: 'HKG', city: 'Hong Kong', country: 'Hong Kong', continent: 'Asia', count: 8 },
  { code: 'SIN', city: 'Singapore', country: 'Singapore', continent: 'Asia', count: 5 },
  { code: 'LAX', city: 'Los Angeles', country: 'United States', continent: 'North America', count: 3 },
]

describe('buildGeoFacetOptions', () => {
  it('only exposes values backed by result rows', () => {
    const result = buildGeoFacetOptions(facets, { continent: '', country: '', city: '' })
    expect(result.continents).toEqual([
      { value: 'Asia', label: 'Asia', count: 13 },
      { value: 'North America', label: 'North America', count: 3 },
    ])
  })

  it('cascades country, city and colo choices after selecting a continent', () => {
    const result = buildGeoFacetOptions(facets, { continent: 'Asia', country: '', city: '' })
    expect(result.countries.map((item) => item.value)).toEqual(['Hong Kong', 'Singapore'])
    expect(result.countries.some((item) => item.value === 'United States')).toBe(false)
    expect(result.colos.map((item) => item.value)).toEqual(['HKG', 'SIN'])
  })

  it('drops selections that are no longer backed by the current result set', () => {
    expect(normalizeGeoSelection(facets, { continent: 'Asia', country: 'United States', city: 'Los Angeles', colo: 'LAX' })).toEqual({
      continent: 'Asia', country: '', city: '', colo: '',
    })
  })


  it('clears every descendant when the selected continent disappears', () => {
    expect(normalizeGeoSelection(facets, { continent: 'Europe', country: 'United States', city: 'Los Angeles', colo: 'LAX' })).toEqual({
      continent: '', country: '', city: '', colo: '',
    })
  })
})
