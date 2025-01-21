// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

import tailwind from '@astrojs/tailwind';

// https://astro.build/config
export default defineConfig({
    site: 'https://riselabqueens.github.io',
    base: 'intertrans',
    integrations: [starlight({
        title: '🛤️ InterTrans Engine',
        customCss: [
            './src/styles/global.css',
        ],
        social: {
            github: 'https://github.com/RISElabQueens/InterTrans',
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