import { defineConfig } from 'wxt';

// See https://wxt.dev/api/config.html
export default defineConfig({
  manifest: {
    permissions: [
      'tabs',
      'scripting',
      'activeTab',
      'storage',
      'sidePanel',
    ],
    host_permissions: ['*://*/*'],
  },
});
