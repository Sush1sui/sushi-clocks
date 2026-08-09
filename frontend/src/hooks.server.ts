import type { Handle } from '@sveltejs/kit';
import type { User } from './app';

const BACKEND_URL = 'http://localhost:8080';

export const handle: Handle = async ({ event, resolve }) => {
	event.locals.user = null;

	const accessToken = event.cookies.get('access_token');
	const refreshToken = event.cookies.get('refresh_token');

	if (!accessToken && !refreshToken) {
		return resolve(event);
	}

	const cookieHeader = event.request.headers.get('cookie') || '';

	try {
		// 1. Try to fetch /me with current access_token
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
					return resolve(event);
				}
			}
		}

		// 2. If access_token failed or missing, try refresh
		if (refreshToken) {
			const refreshRes = await fetch(`${BACKEND_URL}/api/v1/auth/refresh`, {
				method: 'POST',
				headers: {
					cookie: cookieHeader
				}
			});

			if (refreshRes.ok) {
				const refreshData = await refreshRes.json();
				const newAccessToken = refreshData.data?.access_token;

				// Forward new Set-Cookie headers if any
				const setCookie = refreshRes.headers.get('set-cookie');

				if (newAccessToken) {
					const meRes = await fetch(`${BACKEND_URL}/api/v1/auth/me`, {
						headers: {
							Authorization: `Bearer ${newAccessToken}`
						}
					});

					if (meRes.ok) {
						const meData = await meRes.json();
						if (meData.success && meData.data?.user) {
							event.locals.user = meData.data.user as User;
						}
					}
				}

				const response = await resolve(event);
				if (setCookie) {
					response.headers.append('set-cookie', setCookie);
				}
				return response;
			}
		}
	} catch (err) {
		console.error('hooks.server auth error:', err);
	}

	return resolve(event);
};
