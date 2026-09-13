import type { Handle } from '@sveltejs/kit';
import type { User } from './app';
import { env } from '$env/dynamic/private';

const BACKEND_URL = env.BACKEND_URL ?? 'http://localhost:8080';

/**
 * Parse and forward Set-Cookie headers from a backend Response via event.cookies.set().
 *
 * WHY: Node.js fetch merges multiple Set-Cookie response headers into a single
 * comma-joined string when you call headers.get('set-cookie'). Forwarding that
 * raw string via response.headers.append() corrupts the cookie values because
 * the browser sees one malformed cookie instead of two separate ones.
 *
 * FIX: Use getSetCookie() (Node 18+, returns each header as a separate string),
 * then apply each cookie cleanly through SvelteKit's event.cookies.set() API.
 */
function forwardCookies(
	backendRes: Response,
	event: Parameters<Handle>[0]['event']
): void {
	// getSetCookie() is available in Node 18+ undici fetch and returns each
	// Set-Cookie header as a separate entry — exactly what we need.
	const setCookieHeaders: string[] =
		typeof (backendRes.headers as unknown as { getSetCookie?: () => string[] }).getSetCookie ===
		'function'
			? (backendRes.headers as unknown as { getSetCookie: () => string[] }).getSetCookie()
			: [];

	// Fallback: split the comma-joined string for older Node versions.
	// Regex splits on ", " only when followed by a cookie attribute name or token=,
	// avoiding splits inside cookie values that contain commas.
	if (setCookieHeaders.length === 0) {
		const raw = backendRes.headers.get('set-cookie');
		if (raw) {
			setCookieHeaders.push(...raw.split(/,\s*(?=[a-zA-Z0-9_-]+=)/));
		}
	}

	for (const cookieStr of setCookieHeaders) {
		const parts = cookieStr.split(';').map((s) => s.trim());
		const [nameValue, ...attrs] = parts;
		const eqIdx = nameValue.indexOf('=');
		if (eqIdx === -1) continue;

		const name = nameValue.slice(0, eqIdx).trim();
		const value = nameValue.slice(eqIdx + 1).trim();

		const options: Parameters<typeof event.cookies.set>[2] = { path: '/' };
		for (const attr of attrs) {
			const lower = attr.toLowerCase();
			if (lower === 'httponly') options.httpOnly = true;
			else if (lower === 'secure') options.secure = true;
			else if (lower.startsWith('max-age='))
				options.maxAge = parseInt(attr.slice('max-age='.length), 10);
			else if (lower.startsWith('samesite=')) {
				const sv = attr.slice('samesite='.length).toLowerCase();
				options.sameSite = sv === 'strict' ? 'strict' : sv === 'none' ? 'none' : 'lax';
			} else if (lower.startsWith('path=')) options.path = attr.slice('path='.length);
		}

		event.cookies.set(name, value, options);
	}
}

export const handle: Handle = async ({ event, resolve }) => {
	event.locals.user = null;
	event.locals.accessToken = null;

	const accessToken = event.cookies.get('access_token');
	const refreshToken = event.cookies.get('refresh_token');

	if (!accessToken && !refreshToken) {
		return resolve(event);
	}

	const cookieHeader = event.request.headers.get('cookie') || '';

	try {
		// 1. Try /me with current access_token
		if (accessToken) {
			const meRes = await fetch(`${BACKEND_URL}/api/v1/auth/me`, {
				headers: {
					cookie: cookieHeader,
					Authorization: `Bearer ${accessToken}`
				}
			});

			if (meRes.ok) {
				const meData = await meRes.json();
				if (meData.success && meData.data?.user) {
					event.locals.user = meData.data.user as User;
					event.locals.accessToken = accessToken;
					return resolve(event);
				}
			}
		}

		// 2. access_token expired/missing — silent refresh via refresh_token
		if (refreshToken) {
			const refreshRes = await fetch(`${BACKEND_URL}/api/v1/auth/refresh`, {
				method: 'POST',
				headers: { cookie: cookieHeader }
			});

			if (refreshRes.ok) {
				const refreshData = await refreshRes.json();
				const newAccessToken = refreshData.data?.access_token;

				// Apply refreshed cookies to the browser correctly (no header corruption)
				forwardCookies(refreshRes, event);

				if (newAccessToken) {
					const meRes = await fetch(`${BACKEND_URL}/api/v1/auth/me`, {
						headers: { Authorization: `Bearer ${newAccessToken}` }
					});

					if (meRes.ok) {
						const meData = await meRes.json();
						if (meData.success && meData.data?.user) {
							event.locals.user = meData.data.user as User;
							event.locals.accessToken = newAccessToken;
						}
					}
				}

				return resolve(event);
			}
		}
	} catch (err) {
		console.error('hooks.server auth error:', err);
	}

	return resolve(event);
};
