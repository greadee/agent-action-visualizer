import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  build: {
    target: 'es2022',
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: 'three-core',
              test: /node_modules[\\/]three[\\/]/,
              priority: 20,
              maxSize: 450 * 1024,
              includeDependenciesRecursively: false,
            },
            {
              name: 'react-three',
              test: /node_modules[\\/]@react-three[\\/]/,
              priority: 10,
              maxSize: 350 * 1024,
              includeDependenciesRecursively: false,
            },
          ],
        },
      },
    },
  },
})
