// Dynamically register mobile routes into the existing Vue Router instance
// This file enables mode B: inject mobile routes at runtime without touching
// the main router configuration file.
import type { Router } from 'vue-router';

export async function registerMobileRoutes(router: Router) {
  try {
    // Import the mobile routes lazily to avoid eager coupling
    const mod = await import('./index');
    const mobileRoutes = (mod?.default) || [];
    if (!mobileRoutes || !Array.isArray(mobileRoutes)) return;
    for (const r of mobileRoutes) {
      // If a route with the same name already exists, skip to avoid duplicates
      if (r?.name && router.hasRoute?.(r.name)) continue;
      // If no name, try to dedupe by path
      if (!r?.name && r?.path && router.getRoutes().some((rr: any) => rr.path === r.path)) continue;
      router.addRoute(r as any);
    }
  } catch (e) {
    // Best-effort: if dynamic import fails, continue without mobile routes
    // This keeps the desktop site functional.
    // eslint-disable-next-line no-console
    console.warn('Failed to register mobile routes:', e);
  }
}
