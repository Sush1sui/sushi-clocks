import { request } from './client';
import type { User } from '../../app';

export interface LoginResponse {
	user: User;
	access_token: string;
	expires_in: number;
}

export interface RefreshResponse {
	access_token: string;
	expires_in: number;
}

export interface MeResponse {
	user: User;
}

export async function login(email: string, password: string): Promise<LoginResponse> {
	return request<LoginResponse>('/api/v1/auth/login', {
		method: 'POST',
		body: JSON.stringify({ email, password })
	});
}

export async function refresh(customFetch?: typeof fetch): Promise<RefreshResponse> {
	return request<RefreshResponse>('/api/v1/auth/refresh', {
		method: 'POST'
	}, customFetch);
}

export async function logout(): Promise<{ status: string }> {
	return request<{ status: string }>('/api/v1/auth/logout', {
		method: 'POST'
	});
}

export async function getMe(customFetch?: typeof fetch): Promise<MeResponse> {
	return request<MeResponse>('/api/v1/auth/me', {
		method: 'GET'
	}, customFetch);
}
