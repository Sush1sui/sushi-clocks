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

export async function request<T>(
	endpoint: string,
	options: RequestInit = {},
	customFetch: typeof fetch = fetch
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
