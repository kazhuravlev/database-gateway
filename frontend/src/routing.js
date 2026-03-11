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

const BASE_PATH = normalizeBasePath(import.meta.env.BASE_URL || "/");

function normalizeBasePath(value) {
  const trimmed = value.endsWith("/") ? value.slice(0, -1) : value;
  return trimmed === "" ? "" : trimmed;
}

export function appHref(pathname, search = "") {
  const normalizedPath = pathname.startsWith("/") ? pathname : `/${pathname}`;
  return `${BASE_PATH}${normalizedPath}${search}`;
}

export function getCurrentRoute() {
  const pathname = stripBasePath(window.location.pathname);
  const searchParams = new URLSearchParams(window.location.search);

  return {
    pathname,
    searchParams
  };
}

export function navigate(pathname, search = "") {
  window.history.pushState({}, "", appHref(pathname, search));
  window.dispatchEvent(new PopStateEvent("popstate"));
}

export function matchRoute(pathname, pattern) {
  const actualParts = splitPath(pathname);
  const patternParts = splitPath(pattern);

  if (actualParts.length !== patternParts.length) {
    return null;
  }

  const params = {};

  for (let index = 0; index < patternParts.length; index += 1) {
    const actualPart = actualParts[index];
    const patternPart = patternParts[index];

    if (patternPart.startsWith(":")) {
      params[patternPart.slice(1)] = decodeURIComponent(actualPart);
      continue;
    }

    if (actualPart !== patternPart) {
      return null;
    }
  }

  return params;
}

function stripBasePath(pathname) {
  if (BASE_PATH && pathname.startsWith(BASE_PATH)) {
    const candidate = pathname.slice(BASE_PATH.length);
    return candidate || "/";
  }

  return pathname || "/";
}

function splitPath(pathname) {
  return pathname
    .split("/")
    .filter(Boolean);
}
