import type { SearchableComboboxOption } from '@/components/shared/searchable-combobox'

export const TIMEZONES = [
  'UTC',
  'Asia/Shanghai',
  'Asia/Hong_Kong',
  'Asia/Tokyo',
  'Asia/Singapore',
  'Europe/London',
  'Europe/Berlin',
  'America/Los_Angeles',
  'America/New_York',
] as const

export const TIMEZONE_OPTIONS: SearchableComboboxOption[] = TIMEZONES.map((timezone) => ({
  value: timezone,
  label: timezone,
  searchText: timezone.replaceAll('_', ' '),
}))
