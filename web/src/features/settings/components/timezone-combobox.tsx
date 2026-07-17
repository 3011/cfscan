import * as React from 'react'
import { SearchableCombobox } from '@/components/shared/searchable-combobox'
import { TIMEZONE_OPTIONS } from '@/lib/timezones'

interface TimezoneComboboxProps
  extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'value' | 'onChange'> {
  value: string
  onValueChange: (value: string) => void
}

export const TimezoneCombobox = React.forwardRef<HTMLButtonElement, TimezoneComboboxProps>(
  ({ value, onValueChange, ...props }, ref) => {
    const options = React.useMemo(
      () => TIMEZONE_OPTIONS.some((option) => option.value === value)
        ? TIMEZONE_OPTIONS
        : [{ value, label: value, searchText: value.replaceAll('_', ' ') }, ...TIMEZONE_OPTIONS],
      [value],
    )

    return (
      <SearchableCombobox
        ref={ref}
        value={value}
        options={options}
        onValueChange={onValueChange}
        placeholder="选择时区"
        searchPlaceholder="搜索时区，例如 Shanghai、UTC…"
        emptyText="没有匹配的时区"
        aria-label="时区"
        {...props}
      />
    )
  },
)
TimezoneCombobox.displayName = 'TimezoneCombobox'
