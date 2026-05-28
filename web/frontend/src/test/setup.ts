import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach, vi } from 'vitest';

// Polyfill ResizeObserver (Tremor/Headless UI may reference it)
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

// Polyfill IntersectionObserver (about page tour-rail uses it)
globalThis.IntersectionObserver = class IntersectionObserver {
  root = null;
  rootMargin = '';
  thresholds = [];
  observe() {}
  unobserve() {}
  disconnect() {}
  takeRecords() {
    return [];
  }
} as unknown as typeof IntersectionObserver;

// The Claude Code harness intercepts node's globalThis.localStorage with a
// stub that lacks .clear/.setItem/.getItem (you only see this in CI inside
// the harness; a vanilla `node` repl returns a fully-functional jsdom
// localStorage). Install a Map-backed shim before any test runs so suites
// that rely on persistent state behave the same in both environments.
function installLocalStorageShim() {
  const store = new Map<string, string>();
  const shim: Storage = {
    get length() {
      return store.size;
    },
    clear() {
      store.clear();
    },
    getItem(key: string) {
      return store.has(key) ? (store.get(key) as string) : null;
    },
    key(index: number) {
      return Array.from(store.keys())[index] ?? null;
    },
    removeItem(key: string) {
      store.delete(key);
    },
    setItem(key: string, value: string) {
      store.set(key, String(value));
    },
  };
  Object.defineProperty(window, 'localStorage', { value: shim, configurable: true });
}
installLocalStorageShim();

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});
