import { useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { Activity, Eye, EyeOff, Loader2, Radar, ShieldCheck, Waypoints } from 'lucide-react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { ThemeToggle } from '@/components/layout/theme-toggle'
import { useAuth } from '@/features/auth/auth-context'
import { APIError } from '@/lib/http'

const loginSchema = z.object({
  username: z.string().trim().min(1, '请输入用户名'),
  password: z.string().min(1, '请输入密码'),
})
type LoginValues = z.infer<typeof loginSchema>

const highlights = [
  { icon: Activity, label: '多区域实时扫描与结果排行' },
  { icon: Waypoints, label: 'Cloudflare colo 地理分析' },
  { icon: ShieldCheck, label: '管理员与查看者权限隔离' },
]

export function LoginPage() {
  const auth = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const form = useForm<LoginValues>({ resolver: zodResolver(loginSchema), defaultValues: { username: '', password: '' } })

  async function submit(values: LoginValues) {
    setError('')
    try {
      await auth.login(values)
      const target = typeof location.state === 'object' && location.state && 'from' in location.state ? String(location.state.from) : '/'
      navigate(target, { replace: true })
    } catch (reason) {
      setError(reason instanceof APIError ? reason.message : '登录失败，请稍后重试')
    }
  }

  return (
    <main className="relative grid min-h-svh overflow-hidden bg-background lg:grid-cols-[minmax(0,1fr)_minmax(28rem,0.72fr)]">
      <div className="absolute right-4 top-4 z-20"><ThemeToggle /></div>
      <section className="relative hidden overflow-hidden bg-muted/45 p-10 lg:flex lg:flex-col lg:justify-between xl:p-14">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_15%,color-mix(in_oklch,var(--foreground)_7%,transparent),transparent_38%)]" />
        <div className="relative flex items-center gap-3">
          <span className="flex size-9 items-center justify-center rounded-2xl bg-primary text-primary-foreground shadow-sm"><Radar className="size-4" /></span>
          <div><p className="font-heading text-sm font-medium">CF Scanner</p><p className="text-xs text-muted-foreground">Network Quality Console</p></div>
        </div>
        <div className="relative max-w-xl space-y-8">
          <div className="space-y-3">
            <p className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">Cloudflare IP Operations</p>
            <h1 className="font-heading text-4xl font-medium leading-tight tracking-tight xl:text-5xl">从多地区实测数据中，持续找到更优的 Cloudflare IP。</h1>
            <p className="max-w-lg text-sm leading-7 text-muted-foreground">集中管理扫描任务、Agent、地址来源、自动化策略与账号权限。</p>
          </div>
          <div className="grid gap-3">
            {highlights.map((item) => <div key={item.label} className="flex items-center gap-3 text-sm"><span className="flex size-8 items-center justify-center rounded-2xl bg-background/70 shadow-sm ring-1 ring-foreground/5"><item.icon className="size-4 text-muted-foreground" /></span>{item.label}</div>)}
          </div>
        </div>
        <p className="relative text-xs text-muted-foreground">Private management console · Authorized access only</p>
      </section>
      <section className="flex min-h-svh items-center justify-center p-5 sm:p-8 lg:p-12">
        <div className="w-full max-w-sm space-y-6">
          <div className="flex items-center gap-3 lg:hidden">
            <span className="flex size-9 items-center justify-center rounded-2xl bg-primary text-primary-foreground"><Radar className="size-4" /></span>
            <div><p className="font-heading text-sm font-medium">CF Scanner</p><p className="text-xs text-muted-foreground">Network Quality Console</p></div>
          </div>
          <Card className="shadow-lg ring-foreground/7 dark:ring-foreground/12">
            <CardHeader>
              <CardTitle className="text-xl">登录管理平台</CardTitle>
              <CardDescription>使用管理员分配的账号继续访问。</CardDescription>
            </CardHeader>
            <CardContent>
              <Form {...form}>
                <form className="space-y-4" onSubmit={form.handleSubmit(submit)}>
                  {error ? <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert> : null}
                  <FormField control={form.control} name="username" render={({ field }) => (
                    <FormItem><FormLabel>用户名</FormLabel><FormControl><Input autoComplete="username" autoFocus placeholder="admin" {...field} /></FormControl><FormMessage /></FormItem>
                  )} />
                  <FormField control={form.control} name="password" render={({ field }) => (
                    <FormItem><FormLabel>密码</FormLabel><div className="relative"><FormControl><Input type={showPassword ? 'text' : 'password'} autoComplete="current-password" className="pr-9" {...field} /></FormControl><Button type="button" variant="ghost" size="icon-sm" className="absolute right-0.5 top-0.5" onClick={() => setShowPassword((value) => !value)} aria-label={showPassword ? '隐藏密码' : '显示密码'}>{showPassword ? <EyeOff /> : <Eye />}</Button></div><FormMessage /></FormItem>
                  )} />
                  <Button className="w-full" size="lg" type="submit" disabled={auth.loginPending}>{auth.loginPending ? <Loader2 className="animate-spin" /> : null}登录</Button>
                </form>
              </Form>
            </CardContent>
          </Card>
          <p className="text-center text-xs text-muted-foreground">账号由管理员统一创建和授权。</p>
        </div>
      </section>
    </main>
  )
}
