import { mkdir, writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import process from 'node:process'

const nodeModules = join(process.cwd(), 'node_modules')
const boundary = join(nodeModules, 'go.mod')

await mkdir(nodeModules, { recursive: true })
await writeFile(
  boundary,
  [
    'module github.com/greadee/agent-action-visualizer/apps/desktop/frontend/node_modules',
    '',
    'go 1.26.6',
    '',
  ].join('\n'),
)
