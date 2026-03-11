/**
 * Database Gateway provides access to servers with ACL for safe and restricted database interactions.
 * Copyright (C) 2024  Kirill Zhuravlev
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from '@tailwindcss/vite'


export default defineConfig(({ command }) => {
  const basePath =
    process.env.VITE_BASE_PATH || (command === "build" ? "/ui/" : "/");

  return {
    base: basePath,
    plugins: [
		svelte(),
		tailwindcss(),
	],
    server: {
      host: "0.0.0.0",
      port: 5173
    }
  };
});
