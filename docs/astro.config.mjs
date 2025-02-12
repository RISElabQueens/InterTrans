// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

import tailwind from '@astrojs/tailwind';

// https://astro.build/config
export default defineConfig({
    site: 'https://codetransengine-anon.surge.sh',
    base: '/',
    output: 'static',
    integrations: [starlight({
        title: '🛤️ CodeTransEngine',
        customCss: [
            './src/styles/global.css',
        ],
        social: {
            github: 'https://anonymizedsubmissionlink/CodeTransEngine',
        },
        sidebar: [
            {
                label: 'Guides',
				autogenerate: { directory: 'guides' },
            },
            {
                label: 'Reference',
                autogenerate: { directory: 'reference' },
            },
        ],
		}), tailwind()],
});