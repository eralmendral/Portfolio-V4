import { defineConfig } from '@lynx-js/rspeedy'

import { pluginQRCode } from '@lynx-js/qrcode-rsbuild-plugin'
import { pluginReactLynx } from '@lynx-js/react-rsbuild-plugin'
import { pluginTypeCheck } from '@rsbuild/plugin-type-check'

const publicApiBaseUrl = process.env.PUBLIC_API_BASE_URL ?? 'http://localhost:8080'

export default defineConfig({
  environments: {
    web: {},
    lynx: {},
  },
  source: {
    define: {
      'import.meta.env.PUBLIC_API_BASE_URL': JSON.stringify(publicApiBaseUrl),
    },
  },
  server: {
    port: 3000,
  },
  plugins: [
    pluginQRCode({
      schema(url) {
        return `${url}?fullscreen=true`
      },
    }),
    pluginReactLynx({
      defineDCE: {
        define: {
          __PUBLIC_API_BASE_URL__: JSON.stringify(publicApiBaseUrl),
        },
      },
    }),
    pluginTypeCheck(),
  ],
})
