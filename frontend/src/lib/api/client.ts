export interface APIResponse<T = unknown> {
	success: boolean;
	data?: T;
	error?: string;
}

export class APIError extends Error {
	statusCode: number;

	constructor(message: string, statusCode: number) {
		super(message);
		this.name = 'APIError';
		this.statusCode = statusCode;
	}
}

// Track ongoing refresh promise to prevent concurrent refresh stampedes
let refreshPromise: Promise<boolean> | null = null;

async function attemptSilentRefresh(customFetch: typeof fetch): Promise<boolean> {
	if (refreshPromise) {
		return refreshPromise;
	}

	refreshPromise = (async () => {
		try {
			const res = await customFetch('/api/v1/auth/refresh', {
				method: 'POST',
				credentials: 'include'
			});
			if (!res.ok) return false;
			const data = await res.json();
			return !!data.success;
		} catch {
			return false;
		} finally {
			refreshPromise = null;
		}
	})();

	return refreshPromise;
}

export async function request<T>(
	endpoint: string,
	options: RequestInit = {},
	customFetch: typeof fetch = fetch,
	isRetry = false
): Promise<T> {
	const headers = new Headers(options.headers || {});
	if (options.body && !(options.body instanceof FormData)) {
		headers.set('Content-Type', 'application/json');
	}

	const response = await customFetch(endpoint, {
		...options,
		headers,
		credentials: 'include'
	});

	// Handle 401 Unauthorized by attempting a silent token refresh
	if (
		response.status === 401 &&
		!isRetry &&
		!endpoint.includes('/api/v1/auth/login') &&
		!endpoint.includes('/api/v1/auth/refresh')
	) {
		const refreshed = await attemptSilentRefresh(customFetch);
		if (refreshed) {
			// Retry original request once with fresh cookies
			return request<T>(endpoint, options, customFetch, true);
		}
	}

	let json: APIResponse<T>;
	try {
		json = await response.json();
	} catch {
		throw new APIError(`Failed to parse response (status ${response.status})`, response.status);
	}

	if (!response.ok || !json.success) {
		throw new APIError(json.error || `Request failed with status ${response.status}`, response.status);
	}

	return json.data as T;
}
