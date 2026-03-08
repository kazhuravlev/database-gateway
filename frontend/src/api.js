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

const TOKEN_STORAGE_KEY = "dbgw_api_access_token";

export const API_BASE = import.meta.env.DEV
  ? "http://localhost:8080"
  : window.location.origin;
export const AUTH_URL = `${API_BASE}/auth`;
const BASE_URL = `${API_BASE}/api/v1`;

function buildRPCError(payload, fallbackMessage = "Request failed") {
  if (!payload?.error) {
    return null;
  }

  const message =
    typeof payload.error.message === "string" && payload.error.message
      ? payload.error.message
      : `${fallbackMessage}${payload.error.code ? ` (${payload.error.code})` : ""}`;

  return new Error(message);
}

export function getErrorMessage(error, fallbackMessage = "Request failed") {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return fallbackMessage;
}

function createRequestID() {
  if (typeof crypto !== "undefined" && crypto.randomUUID) {
    return crypto.randomUUID();
  }

  return `request-${Date.now()}`;
}

function isUnauthorizedStatus(status) {
  return status === 401 || status === 403;
}

export function startAuthFlow() {
  window.location.href = AUTH_URL;
}

export function getStoredToken() {
  return window.localStorage.getItem(TOKEN_STORAGE_KEY) || "";
}

export function setStoredToken(token) {
  if (!token) {
    window.localStorage.removeItem(TOKEN_STORAGE_KEY);
    return;
  }

  window.localStorage.setItem(TOKEN_STORAGE_KEY, token);
}

export function clearStoredToken() {
  setStoredToken("");
}

export function consumeTokenFromURL() {
  const hash = window.location.hash.startsWith("#")
    ? window.location.hash.slice(1)
    : window.location.hash;
  if (!hash) {
    return "";
  }

  const params = new URLSearchParams(hash);
  const token = params.get("access_token") || "";
  if (!token) {
    return "";
  }

  setStoredToken(token);
  window.history.replaceState({}, document.title, window.location.pathname + window.location.search);

  return token;
}

async function rpcCall(token, method, params) {
  let response;

  try {
    response = await fetch(`${BASE_URL}/${method}`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json"
      },
      body: JSON.stringify({
        lrpc: "1",
        id: createRequestID(),
        params
      })
    });
  } catch (error) {
    throw new Error(getErrorMessage(error, "Network request failed"));
  }

  if (isUnauthorizedStatus(response.status)) {
    throw new Error(`Unauthorized (${response.status})`);
  }

  let payload = null;

  try {
    payload = await response.json();
  } catch (error) {
    if (!response.ok) {
      throw new Error(`Failed to do request (${response.status})`);
    }

    throw new Error(getErrorMessage(error, "Failed to parse server response"));
  }

  const rpcError = buildRPCError(payload);
  if (rpcError) {
    throw rpcError;
  }

  if (!response.ok) {
    throw new Error(`Failed to do request (${response.status})`);
  }

  return payload.result;
}

export async function withAuthorizedRequest(run) {
  const token = getStoredToken();
  if (!token) {
    startAuthFlow();
    return null;
  }

  try {
    return await run(token);
  } catch (error) {
    const message = getErrorMessage(error, "");
    if (!message.includes("Unauthorized")) {
      throw error;
    }

    clearStoredToken();
    startAuthFlow();
    return null;
  }
}

export function listServers(token) {
  return rpcCall(token, "targets.list.v1", {});
}

export function getServer(token, targetID) {
  return rpcCall(token, "targets.get.v1", {
    target_id: targetID
  });
}

export function getProfile(token) {
  return rpcCall(token, "profile.get.v1", {});
}

export function listBookmarks(token, targetID = "") {
  return rpcCall(token, "bookmarks.list.v1", targetID ? { target_id: targetID } : {});
}

export function listQueries(token, limit) {
  return rpcCall(token, "queries.list.v1", typeof limit === "number" ? { limit } : {});
}

export function runQuery(token, targetID, query) {
  return rpcCall(token, "query.run.v1", {
    target_id: targetID,
    query
  });
}

export function listAdminRequests(token, page) {
  return rpcCall(token, "admin.requests.list.v1", {
    page
  });
}

export function getQueryResults(token, queryResultID) {
  return rpcCall(token, "query-results.get.v1", {
    id: queryResultID
  });
}

export function getQueryResultsExportLink(token, queryResultID, format) {
  return rpcCall(token, "query-results.export-link.v1", {
    query_result_id: queryResultID,
    format
  });
}

export function addBookmark(token, targetID, title, query) {
  return rpcCall(token, "bookmarks.add.v1", {
    target_id: targetID,
    title,
    query
  });
}

export function deleteBookmark(token, bookmarkID) {
  return rpcCall(token, "bookmarks.delete.v1", {
    id: bookmarkID
  });
}
