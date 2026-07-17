import * as React from 'react'
import { Button } from '@/components/ui/button'
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxTrigger,
} from '@/components/ui/combobox'
import { cn } from '@/lib/utils'

export interface SearchableComboboxOption {
  value: string
  label: string
  searchText?: string
  countLabel?: string
  disabled?: boolean
}

export interface SearchableComboboxProps
  extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'value' | 'onChange'> {
  value: string
  options: SearchableComboboxOption[]
  onValueChange: (value: string) => void
  placeholder: string
  searchPlaceholder?: string
  emptyText?: string
  allOption?: SearchableComboboxOption
  contentClassName?: string
}

export const SearchableCombobox = React.forwardRef<HTMLButtonElement, SearchableComboboxProps>(
  ({
    value,
    options,
    onValueChange,
    placeholder,
    searchPlaceholder = '搜索…',
    emptyText = '没有匹配选项',
    allOption,
    contentClassName,
    className,
    disabled,
    ...triggerProps
  }, ref) => {
    const [portalContainer, setPortalContainer] = React.useState<HTMLElement | null>(null)
    const triggerRef = React.useCallback((node: HTMLButtonElement | null) => {
      setPortalContainer(node?.closest<HTMLElement>('[role="dialog"]') ?? null)
      if (typeof ref === 'function') ref(node)
      else if (ref) ref.current = node
    }, [ref])
    const items = React.useMemo(
      () => allOption ? [allOption, ...options] : options,
      [allOption, options],
    )
    const selected = items.find((item) => item.value === value) ?? null

    return (
      <Combobox
        items={items}
        value={selected}
        onValueChange={(item) => onValueChange(item?.value ?? '')}
        itemToStringValue={(item) => [item.label, item.searchText].filter(Boolean).join(' ')}
        autoHighlight
      >
        <ComboboxTrigger
          render={
            <Button
              ref={triggerRef}
              type="button"
              variant="outline"
              className={cn('w-full justify-between border-transparent bg-input/50 px-3 font-normal shadow-none hover:bg-input/50 aria-expanded:bg-input/50 dark:bg-input/50 dark:hover:bg-input/50', className)}
              disabled={disabled}
              {...triggerProps}
            />
          }
        >
          <span data-slot="combobox-value" className={cn('min-w-0 flex-1 truncate text-left', !selected && 'text-muted-foreground')}>
            {selected?.label ?? placeholder}
          </span>
        </ComboboxTrigger>
        <ComboboxContent className={contentClassName} container={portalContainer ?? undefined}>
          <ComboboxInput showTrigger={false} placeholder={searchPlaceholder} />
          <ComboboxEmpty>{emptyText}</ComboboxEmpty>
          <ComboboxList>
            {(item) => (
              <ComboboxItem key={item.value} value={item} disabled={item.disabled}>
                <span className="flex min-w-0 flex-1 items-center justify-between gap-4">
                  <span className="truncate">{item.label}</span>
                  {item.countLabel ? (
                    <span className="shrink-0 tabular-nums text-muted-foreground">{item.countLabel}</span>
                  ) : null}
                </span>
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
    )
  },
)
SearchableCombobox.displayName = 'SearchableCombobox'
