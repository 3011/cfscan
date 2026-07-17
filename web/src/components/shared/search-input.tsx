import type { ComponentProps } from 'react'
import { Search } from 'lucide-react'
import { InputGroup, InputGroupAddon, InputGroupInput } from '@/components/ui/input-group'
import { cn } from '@/lib/utils'

interface SearchInputProps extends Omit<ComponentProps<typeof InputGroupInput>, 'className'> {
  className?: string
  inputClassName?: string
}

export function SearchInput({ className, inputClassName, ...props }: SearchInputProps) {
  return (
    <InputGroup className={className}>
      <InputGroupAddon aria-hidden="true">
        <Search className="size-4" />
      </InputGroupAddon>
      <InputGroupInput className={cn('min-w-0', inputClassName)} {...props} />
    </InputGroup>
  )
}
