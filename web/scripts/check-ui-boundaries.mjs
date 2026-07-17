import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = path.resolve(import.meta.dirname, '..')
const srcRoot = path.join(root, 'src')
const uiRoot = path.join(srcRoot, 'components', 'ui')
const errors = []

function walk(directory) {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const fullPath = path.join(directory, entry.name)
    if (entry.isDirectory()) return walk(fullPath)
    return /\.(ts|tsx)$/.test(entry.name) ? [fullPath] : []
  })
}

for (const file of walk(srcRoot)) {
  const content = fs.readFileSync(file, 'utf8')
  const relative = path.relative(root, file)
  if (content.includes('@radix-ui/')) {
    errors.push(`${relative}: Radix primitive imports are not allowed after the Rhea/Base UI migration.`)
  }
  if (!file.startsWith(`${uiRoot}${path.sep}`) && content.includes('@base-ui/react')) {
    errors.push(`${relative}: business and shared code must import project UI wrappers instead of @base-ui/react.`)
  }
  if (!file.startsWith(`${uiRoot}${path.sep}`) && /<(button|select|textarea)(?:\s|>)/.test(content)) {
    errors.push(`${relative}: use project UI components instead of raw interactive HTML elements.`)
  }
}

const pkg = JSON.parse(fs.readFileSync(path.join(root, 'package.json'), 'utf8'))
const dependencies = { ...pkg.dependencies, ...pkg.devDependencies }
for (const name of Object.keys(dependencies)) {
  if (name.startsWith('@radix-ui/')) errors.push(`package.json: remove legacy dependency ${name}.`)
}
if (!dependencies['@base-ui/react']) errors.push('package.json: @base-ui/react is required as the default primitive library.')
if (!String(dependencies.tailwindcss ?? '').startsWith('^4')) errors.push('package.json: Tailwind CSS 4 is required.')

const config = JSON.parse(fs.readFileSync(path.join(root, 'components.json'), 'utf8'))
if (config.style !== 'base-rhea') errors.push(`components.json: expected style "base-rhea", received "${config.style}".`)
if (config.tailwind?.config) errors.push('components.json: Tailwind 4 projects must not reference a legacy tailwind.config file.')

for (const legacy of ['tailwind.config.js', 'postcss.config.js']) {
  if (fs.existsSync(path.join(root, legacy))) errors.push(`${legacy}: legacy Tailwind 3 configuration must remain removed.`)
}

if (errors.length) {
  console.error(errors.join('\n'))
  process.exit(1)
}
console.log('UI boundaries: Rhea + Base UI + Tailwind 4')
