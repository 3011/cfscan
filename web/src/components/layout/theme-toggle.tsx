import { Check, Laptop, Moon, Sun } from 'lucide-react'
import { useTheme } from 'next-themes'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

type ThemeName = 'light' | 'dark' | 'system'

const themes: Array<{ value: ThemeName; label: string; icon: typeof Sun }> = [
  { value: 'light', label: '浅色', icon: Sun },
  { value: 'dark', label: '深色', icon: Moon },
  { value: 'system', label: '跟随系统', icon: Laptop },
]

export function ThemeToggle() {
  const { theme = 'system', setTheme } = useTheme()
  const selected = themes.find((item) => item.value === theme) ?? themes[2]
  const CurrentIcon = selected.icon

  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="ghost" size="icon" aria-label={`当前主题：${selected.label}，点击切换`} />}>
          <CurrentIcon className="size-4" />
        </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        {themes.map((item) => {
          const Icon = item.icon
          const active = item.value === theme
          return (
            <DropdownMenuItem key={item.value} onClick={() => setTheme(item.value)}>
              <Icon />
              <span className="flex-1">{item.label}</span>
              {active ? <Check className="text-muted-foreground" /> : null}
            </DropdownMenuItem>
          )
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
