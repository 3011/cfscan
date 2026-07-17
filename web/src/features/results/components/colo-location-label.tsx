import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

interface ColoLocationLabelProps {
  code: string
  city?: string
  country?: string
  continent?: string
  className?: string
  cityOnly?: boolean
}

export function ColoLocationLabel({
  code,
  city = '',
  country = '',
  continent = '',
  className,
  cityOnly = false,
}: ColoLocationLabelProps) {
  if (!code) return <span className="text-muted-foreground">—</span>
  const known = Boolean(city && country)
  const location = known ? (cityOnly ? city : `${city}, ${country}`) : 'Location unknown'
  const accessibleLabel = known ? `${code} · ${city}, ${country}${continent ? ` · ${continent}` : ''}` : `${code} · Location unknown`

  return (
    <Tooltip>
      <TooltipTrigger render={<span tabIndex={0}
          className={cn('inline-block max-w-full cursor-help truncate rounded-sm text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring', className)}
          aria-label={accessibleLabel} />}>
          <span className="font-mono font-medium">{code}</span>
          <span className="text-muted-foreground"> · {location}</span>
        </TooltipTrigger>
      <TooltipContent>
        {known ? (
          <div className="space-y-0.5">
            <p className="font-medium">{code} · {city}, {country}</p>
            {continent ? <p>{continent}</p> : null}
          </div>
        ) : <p>Cloudflare location metadata is unavailable for this colo.</p>}
      </TooltipContent>
    </Tooltip>
  )
}
