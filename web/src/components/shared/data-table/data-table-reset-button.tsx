import { RotateCcw } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

interface DataTableResetButtonProps {
  disabled: boolean
  onClick: () => void
  className?: string
}

export function DataTableResetButton({
  disabled,
  onClick,
  className,
}: DataTableResetButtonProps) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="default"
      className={cn("shrink-0", className)}
      disabled={disabled}
      onClick={onClick}
      aria-label="重置筛选"
    >
      <RotateCcw />
      重置
    </Button>
  )
}
