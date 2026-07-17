import { chromium } from 'playwright-core'

const baseURL = process.env.BASE_URL ?? 'http://127.0.0.1:4173'
const username = process.env.CFSCAN_UI_USERNAME
const password = process.env.CFSCAN_UI_PASSWORD

if (!username || !password) {
  throw new Error('Set CFSCAN_UI_USERNAME and CFSCAN_UI_PASSWORD for an administrator smoke-test account.')
}

const routes = [
  ['/', '运行总览'],
  ['/jobs', '扫描任务'],
  ['/results', '结果排行'],
  ['/sources', 'IP 数据源'],
  ['/blacklist', '黑名单'],
  ['/agents', 'Agent 节点'],
  ['/settings', '设置'],
  ['/users', '账号与权限'],
]

const browser = await chromium.launch({
  executablePath: process.env.CHROMIUM_PATH ?? '/usr/bin/chromium',
  headless: true,
  args: ['--no-sandbox'],
})

const report = { desktop: [], interactions: {}, mobile: [], errors: [] }

function captureErrors(page, prefix = '') {
  page.on('pageerror', (error) => report.errors.push(`${prefix}page:${error.message}`))
  page.on('console', (message) => {
    if (message.type() === 'error' && !message.text().includes('401')) {
      report.errors.push(`${prefix}console:${message.text()}`)
    }
  })
}

async function login(page) {
  await page.goto(`${baseURL}/login`, { waitUntil: 'domcontentloaded' })
  await page.getByText('登录管理平台', { exact: true }).waitFor({ timeout: 20_000 })
  await page.getByLabel('用户名').fill(username)
  await page.locator('input[name=password]').fill(password)
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await page.getByRole('heading', { level: 1, name: '运行总览' }).waitFor({ timeout: 20_000 })
}

async function verifyPersistentReset(page, { route, heading, placeholder, setup }) {
  await page.goto(`${baseURL}${route}`, { waitUntil: 'domcontentloaded' })
  await page.getByRole('heading', { level: 1, name: heading }).waitFor({ timeout: 20_000 })
  if (setup) await setup()
  const reset = page.getByRole('button', { name: '重置筛选' })
  await reset.waitFor()
  const input = page.getByPlaceholder(placeholder).first()
  const initiallyDisabled = await reset.isDisabled()
  await input.fill('cfscan-no-match')
  await page.waitForFunction(() => {
    const button = document.querySelector('button[aria-label="重置筛选"]')
    return button && !button.hasAttribute('disabled')
  })
  const enabledAfterFilter = !(await reset.isDisabled())
  await reset.click()
  await page.waitForFunction((expectedPlaceholder) => {
    const input = [...document.querySelectorAll('input')].find((element) => element.getAttribute('placeholder') === expectedPlaceholder)
    const button = document.querySelector('button[aria-label="重置筛选"]')
    return input?.value === '' && button?.hasAttribute('disabled')
  }, placeholder)
  return initiallyDisabled && enabledAfterFilter && (await input.inputValue()) === '' && (await reset.isDisabled())
}

try {
  const desktopContext = await browser.newContext({ viewport: { width: 1440, height: 1000 }, colorScheme: 'light' })
  await desktopContext.addInitScript(() => localStorage.setItem('theme', 'light'))
  const page = await desktopContext.newPage()
  captureErrors(page)
  await login(page)

  for (const [route, heading] of routes) {
    await page.goto(`${baseURL}${route}`, { waitUntil: 'domcontentloaded' })
    await page.getByRole('heading', { level: 1, name: heading }).waitFor({ timeout: 20_000 })
    await page.waitForTimeout(100)
    report.desktop.push({
      route,
      ...(await page.evaluate(() => ({
        overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
        errorBoundary: [...document.querySelectorAll('body *')].some((element) => element.textContent === '页面发生错误'),
      }))),
    })
  }

  await page.goto(`${baseURL}/results`, { waitUntil: 'domcontentloaded' })
  await page.getByRole('heading', { level: 1, name: '结果排行' }).waitFor()

  const searchGeometry = await page.getByPlaceholder('搜索 IP').evaluate((input) => {
    const group = input.closest('[data-slot=input-group]')
    const icon = group?.querySelector('svg')
    if (!group || !icon) return null
    const groupBox = group.getBoundingClientRect()
    const iconBox = icon.getBoundingClientRect()
    return Math.abs((iconBox.top + iconBox.height / 2) - (groupBox.top + groupBox.height / 2))
  })
  report.interactions.searchIconCentered = searchGeometry !== null && searchGeometry <= 0.5

  const availableLabel = await page.getByRole('combobox', { name: '可用状态' }).innerText()
  const timeLabel = await page.getByRole('combobox', { name: '时间范围' }).innerText()
  report.interactions.selectLabels = availableLabel.includes('仅可用') && timeLabel === '最近 24 小时'

  const selectionSurfaces = await page.evaluate(() => {
    const select = document.querySelector('[data-slot=select-trigger][aria-label=\"可用状态\"]')
    const combobox = document.querySelector('[data-slot=combobox-trigger][aria-label=\"扫描任务\"]')
    const reset = document.querySelector('button[aria-label=\"重置筛选\"]')
    const viewOptions = [...document.querySelectorAll('button')].find((element) => element.textContent?.includes('显示列'))
    if (!select || !combobox || !reset || !viewOptions) return null
    const selectStyle = getComputedStyle(select)
    const comboboxStyle = getComputedStyle(combobox)
    const selectBox = select.getBoundingClientRect()
    const comboboxBox = combobox.getBoundingClientRect()
    const resetBox = reset.getBoundingClientRect()
    const viewOptionsBox = viewOptions.getBoundingClientRect()
    return {
      selectBackground: selectStyle.backgroundColor,
      comboboxBackground: comboboxStyle.backgroundColor,
      selectBorder: selectStyle.borderColor,
      comboboxBorder: comboboxStyle.borderColor,
      selectRadius: selectStyle.borderRadius,
      comboboxRadius: comboboxStyle.borderRadius,
      selectHeight: selectBox.height,
      comboboxHeight: comboboxBox.height,
      resetHeight: resetBox.height,
      viewOptionsHeight: viewOptionsBox.height,
    }
  })
  report.interactions.selectionControlSurfaces =
    selectionSurfaces !== null &&
    selectionSurfaces.selectBackground === selectionSurfaces.comboboxBackground &&
    selectionSurfaces.selectBorder === selectionSurfaces.comboboxBorder &&
    selectionSurfaces.selectRadius === selectionSurfaces.comboboxRadius &&
    Math.abs(selectionSurfaces.selectHeight - selectionSurfaces.comboboxHeight) <= 0.5 &&
    Math.abs(selectionSurfaces.selectHeight - selectionSurfaces.resetHeight) <= 0.5 &&
    Math.abs(selectionSurfaces.selectHeight - selectionSurfaces.viewOptionsHeight) <= 0.5


  const selectionTextAlignment = await page.evaluate(() => {
    const selectValue = document.querySelector('[data-slot=select-trigger][aria-label="可用状态"] [data-slot=select-value]')
    const comboboxValue = document.querySelector('[data-slot=combobox-trigger][aria-label="扫描任务"] [data-slot=combobox-value]')
    if (!selectValue || !comboboxValue) return null
    return {
      select: getComputedStyle(selectValue).textAlign,
      combobox: getComputedStyle(comboboxValue).textAlign,
    }
  })
  report.interactions.selectionTextAlignment =
    selectionTextAlignment !== null &&
    selectionTextAlignment.select === 'left' &&
    selectionTextAlignment.combobox === 'left'

  const resetCases = [
    { route: '/results', heading: '结果排行', placeholder: '搜索 IP' },
    { route: '/jobs', heading: '扫描任务', placeholder: '搜索任务名称' },
    { route: '/agents', heading: 'Agent 节点', placeholder: '搜索名称、地区或大洲' },
    { route: '/blacklist', heading: '黑名单', placeholder: '搜索 IP' },
    { route: '/sources', heading: 'IP 数据源', placeholder: '搜索 ASN、名称或组织' },
    { route: '/users', heading: '账号与权限', placeholder: '搜索用户名或显示名称' },
  ]
  const resetChecks = []
  for (const item of resetCases) resetChecks.push(await verifyPersistentReset(page, item))
  resetChecks.push(await verifyPersistentReset(page, {
    route: '/settings',
    heading: '设置',
    placeholder: '搜索计划名称',
    setup: () => page.getByRole('tab', { name: '扫描计划' }).click(),
  }))
  resetChecks.push(await verifyPersistentReset(page, {
    route: '/settings',
    heading: '设置',
    placeholder: '搜索自动化名称',
    setup: () => page.getByRole('tab', { name: '执行记录' }).click(),
  }))
  report.interactions.persistentResetButtons = resetChecks.every(Boolean)

  await page.goto(`${baseURL}/results`, { waitUntil: 'domcontentloaded' })
  await page.getByRole('heading', { level: 1, name: '结果排行' }).waitFor()
  await page.getByRole('button', { name: '显示列' }).click()
  const columnMenu = page.locator('[data-slot=dropdown-menu-content]')
  await columnMenu.waitFor()
  report.interactions.officialColumnMenu =
    (await columnMenu.locator('[data-slot=dropdown-menu-checkbox-item]').count()) > 0 &&
    !(await columnMenu.innerText()).includes('_')
  await page.keyboard.press('Escape')

  const tablePadding = await page.locator('[data-slot=table-container]').first().evaluate((container) => {
    const head = container.querySelector('thead th')
    const cell = container.querySelector('tbody td')
    return {
      head: head ? Number.parseFloat(getComputedStyle(head).paddingLeft) : 0,
      cell: cell ? Number.parseFloat(getComputedStyle(cell).paddingLeft) : 0,
    }
  })
  report.interactions.tableEdgePadding = tablePadding.head >= 16 && tablePadding.cell >= 16

  await page.getByRole('tab', { name: '历史记录' }).click()
  report.interactions.tabs = true

  await page.getByRole('combobox', { name: '扫描任务' }).click()
  const combobox = page.locator('[data-slot=combobox-content]')
  await combobox.waitFor()
  await combobox.locator('input').fill('v0')
  report.interactions.combobox = await combobox.isVisible()
  await page.keyboard.press('Escape')

  await page.getByRole('combobox', { name: '可用状态' }).click()
  const select = page.locator('[data-slot=select-content]')
  await select.waitFor()
  report.interactions.select = await select.isVisible()
  await page.keyboard.press('Escape')

  await page.locator('[data-slot=sidebar-footer] [data-slot=dropdown-menu-trigger]').click()
  const accountMenu = page.locator('[data-slot=dropdown-menu-content]')
  await accountMenu.waitFor()
  report.interactions.accountMenu = (await accountMenu.innerText()).includes('退出登录')
  await page.keyboard.press('Escape')

  await page.goto(`${baseURL}/settings`, { waitUntil: 'domcontentloaded' })
  await page.getByRole('heading', { level: 1, name: '设置' }).waitFor()
  const desktopSettingsTabs = page.locator('[data-slot=tabs]').first()
  const desktopSettingsLabels = await desktopSettingsTabs.locator('[data-slot=tabs-trigger]').allInnerTexts()
  const settingsGeometry = await desktopSettingsTabs.evaluate((tabs) => {
    const list = tabs.querySelector('[data-slot=tabs-list]')
    const content = [...tabs.querySelectorAll('[data-slot=tabs-content]')]
      .find((element) => element.getBoundingClientRect().height > 0)
    if (!list || !content) return null
    const listBox = list.getBoundingClientRect()
    const contentBox = content.getBoundingClientRect()
    return {
      listBottom: listBox.bottom,
      contentTop: contentBox.top,
      contentWidth: contentBox.width,
      tabsWidth: tabs.getBoundingClientRect().width,
    }
  })
  report.interactions.settingsNavigation =
    (await desktopSettingsTabs.getAttribute('data-orientation')) === 'horizontal' &&
    desktopSettingsLabels.join('|') === '自动化总览|扫描计划|黑名单复查|数据源同步|执行记录|外观' &&
    settingsGeometry !== null &&
    settingsGeometry.contentTop >= settingsGeometry.listBottom &&
    settingsGeometry.contentWidth >= settingsGeometry.tabsWidth - 1

  await page.getByRole('tab', { name: '扫描计划' }).click()
  await page.getByRole('button', { name: '显示列' }).click()
  const scheduleColumnMenu = page.locator('[data-slot=dropdown-menu-content]')
  await scheduleColumnMenu.waitFor()
  const scheduleColumnText = await scheduleColumnMenu.innerText()
  report.interactions.scheduleColumnLabels =
    ['计划', '频率', '扫描规模', '下次执行', '状态'].every((label) => scheduleColumnText.includes(label)) &&
    !scheduleColumnText.includes('_')
  await page.keyboard.press('Escape')

  await page.getByRole('tab', { name: '外观' }).click()
  const themeGroup = page.getByRole('radiogroup', { name: '界面主题' })
  await themeGroup.waitFor()
  const themeRadios = themeGroup.getByRole('radio')
  report.interactions.officialThemeRadioGroup = (await themeRadios.count()) === 3
  await themeGroup.getByRole('radio', { name: '深色' }).click()
  report.interactions.themeRadioSelection = await themeGroup.getByRole('radio', { name: '深色' }).isChecked()
  await themeGroup.getByRole('radio', { name: '浅色' }).click()

  await page.goto(`${baseURL}/sources`, { waitUntil: 'domcontentloaded' })
  await page.getByRole('heading', { level: 1, name: 'IP 数据源' }).waitFor()
  await page.getByRole('button', { name: '添加 ASN' }).click()
  report.interactions.dialog = await page.locator('[data-slot=dialog-content]').isVisible()
  await page.getByRole('button', { name: '取消' }).click()

  await page.goto(`${baseURL}/jobs`, { waitUntil: 'domcontentloaded' })
  await page.getByRole('heading', { level: 1, name: '扫描任务' }).waitFor()
  await page.getByRole('button', { name: '创建扫描任务' }).click()
  await page.getByLabel('任务名称').fill('Rhea smoke unsaved')
  report.interactions.sheet = await page.locator('[data-slot=sheet-content]').isVisible()
  await page.getByRole('button', { name: '取消' }).click()
  const alertDialog = page.locator('[data-slot=alert-dialog-content]')
  await alertDialog.waitFor()
  report.interactions.alertDialog = await alertDialog.isVisible()
  await page.getByRole('button', { name: '放弃修改' }).click()

  const rail = page.locator('[data-slot=sidebar-rail]')
  const railBeforeHover = await rail.evaluate((element) => {
    const elementBox = element.getBoundingClientRect()
    const indicator = getComputedStyle(element, '::after')
    return {
      elementHeight: elementBox.height,
      height: Number.parseFloat(indicator.height),
      opacity: Number.parseFloat(indicator.opacity),
      top: Number.parseFloat(indicator.top),
    }
  })
  await rail.hover()
  await page.waitForTimeout(200)
  const railOnHover = await rail.evaluate((element) => {
    const elementBox = element.getBoundingClientRect()
    const indicator = getComputedStyle(element, '::after')
    return {
      elementHeight: elementBox.height,
      height: Number.parseFloat(indicator.height),
      opacity: Number.parseFloat(indicator.opacity),
      top: Number.parseFloat(indicator.top),
    }
  })
  report.interactions.sidebarRail =
    railBeforeHover.opacity === 0 &&
    railBeforeHover.height <= 48.5 &&
    railOnHover.opacity >= 0.95 &&
    railOnHover.height >= 63.5 &&
    railOnHover.height <= 64.5 &&
    Math.abs(railOnHover.top - railOnHover.elementHeight / 2) <= 1

  await page.locator('header').getByRole('button', { name: 'Toggle Sidebar' }).click()
  await page.waitForTimeout(250)
  const collapsedGeometry = await page.evaluate(() => {
    const sidebar = document.querySelector('[data-slot=sidebar-inner]')
    const logoButton = sidebar?.querySelector('[data-slot=sidebar-header] [data-slot=sidebar-menu-button]')
    const links = [...(sidebar?.querySelectorAll('[data-slot=sidebar-content] a[data-slot=sidebar-menu-button]') ?? [])]
      .filter((element) => element.getBoundingClientRect().height > 0)
    const centers = links.map((element) => {
      const box = element.getBoundingClientRect()
      return box.top + box.height / 2
    })
    const gaps = centers.slice(1).map((center, index) => center - centers[index])
    const logoBox = logoButton?.getBoundingClientRect()
    const logoCenter = logoBox ? logoBox.top + logoBox.height / 2 : 0
    return {
      gaps,
      logoGap: centers.length ? centers[0] - logoCenter : 0,
      visibleLabels: [...(sidebar?.querySelectorAll('[data-slot=sidebar-group-label]') ?? [])]
        .filter((element) => getComputedStyle(element).display !== 'none' && element.getBoundingClientRect().height > 0).length,
      cloudLogo: Boolean(sidebar?.querySelector('[data-slot=sidebar-header] svg.lucide-cloud')),
      radarLogo: Boolean(sidebar?.querySelector('[data-slot=sidebar-header] svg.lucide-radar')),
      headerSeparators: document.querySelectorAll('header [data-slot=separator]').length,
    }
  })
  const gapSpread = collapsedGeometry.gaps.length
    ? Math.max(...collapsedGeometry.gaps) - Math.min(...collapsedGeometry.gaps)
    : Number.POSITIVE_INFINITY
  report.interactions.sidebarSpacing =
    gapSpread <= 1 &&
    collapsedGeometry.logoGap >= 44 &&
    collapsedGeometry.visibleLabels === 0 &&
    collapsedGeometry.cloudLogo &&
    !collapsedGeometry.radarLogo
  report.interactions.headerSeparatorRemoved = collapsedGeometry.headerSeparators === 0
  await page.waitForTimeout(50)
  report.interactions.sidebarNoTooltipFlash = (await page.locator('[data-slot=tooltip-content]').count()) === 0
  await page.waitForTimeout(300)
  await page.locator('[data-slot=sidebar-inner] a[href="/results"]').hover()
  const tooltip = page.locator('[data-slot=tooltip-content]')
  await tooltip.waitFor()
  report.interactions.tooltip = (await tooltip.innerText()) === '结果排行' && (await page.locator('[data-slot=tooltip-content]').count()) === 1

  let collapsedLinksWork = true
  for (const [route, heading] of routes) {
    try {
      await page.locator(`[data-slot=sidebar-inner] a[href="${route}"]`).first().click()
      await page.getByRole('heading', { level: 1, name: heading }).waitFor({ timeout: 10_000 })
      collapsedLinksWork &&= new URL(page.url()).pathname === route
    } catch {
      collapsedLinksWork = false
    }
  }
  report.interactions.collapsedSidebarLinks = collapsedLinksWork
  report.interactions.sidebar = await page.evaluate(() => ({
    state: document.querySelector('.group.peer[data-state]')?.getAttribute('data-state') ?? null,
    overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
  }))
  await desktopContext.close()

  const mobileContext = await browser.newContext({ viewport: { width: 390, height: 844 }, colorScheme: 'dark' })
  await mobileContext.addInitScript(() => localStorage.setItem('theme', 'dark'))
  const mobile = await mobileContext.newPage()
  captureErrors(mobile, 'mobile-')
  await login(mobile)

  for (const [route, heading] of routes) {
    await mobile.goto(`${baseURL}${route}`, { waitUntil: 'domcontentloaded' })
    await mobile.getByRole('heading', { level: 1, name: heading }).waitFor({ timeout: 20_000 })
    await mobile.waitForTimeout(100)
    report.mobile.push({
      route,
      ...(await mobile.evaluate(() => ({
        overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
        dark: document.documentElement.classList.contains('dark'),
        headerHeight: Math.round(document.querySelector('header')?.getBoundingClientRect().height ?? 0),
        errorBoundary: [...document.querySelectorAll('body *')].some((element) => element.textContent === '页面发生错误'),
      }))),
    })
  }

  await mobile.goto(`${baseURL}/settings`, { waitUntil: 'domcontentloaded' })
  await mobile.getByRole('heading', { level: 1, name: '设置' }).waitFor()
  const mobileSettingsTabs = mobile.locator('[data-slot=tabs]').first()
  report.interactions.mobileSettingsNavigation =
    (await mobileSettingsTabs.getAttribute('data-orientation')) === 'horizontal' &&
    (await mobile.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)) === 0

  await mobile.locator('header').getByRole('button', { name: 'Toggle Sidebar' }).click()
  const mobileSidebar = mobile.locator('[data-mobile=true]')
  await mobileSidebar.waitFor()
  report.interactions.mobileSidebar = {
    visible: await mobileSidebar.isVisible(),
    width: await mobileSidebar.evaluate((element) => Math.round(element.getBoundingClientRect().width)),
    overflow: await mobile.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth),
  }
  await mobileContext.close()
} finally {
  await browser.close()
}

const failedRoutes = [...report.desktop, ...report.mobile].filter((item) => item.overflow !== 0 || item.errorBoundary)
const failedInteractions = Object.entries(report.interactions).filter(([, value]) => value === false)
if (failedRoutes.length || failedInteractions.length || report.errors.length) {
  console.error(JSON.stringify(report, null, 2))
  process.exit(1)
}
console.log(JSON.stringify(report, null, 2))
