import type { GraphNode } from './types'

export const nodeColors: Record<GraphNode['kind'], string> = {
  root: '#e8f6f3',
  directory: '#6f8fa9',
  source: '#42c7ad',
  test: '#b18cff',
  config: '#f0b45d',
  documentation: '#74a8ff',
  asset: '#ec7ea8',
  tombstone: '#66717c',
}
