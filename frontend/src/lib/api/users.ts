import { request } from './client';

export interface UserResponse {
	id: string;
	company_id: string;
	first_name: string;
	last_name: string;
	email: string;
	mobile_number?: string | null;
	system_role: 'admin' | 'hr' | 'employee';
	created_at: string;
}

export interface CreateStaffPayload {
	first_name: string;
	last_name: string;
	email: string;
	password: string;
	mobile_number?: string;
	system_role: 'hr' | 'employee';
}

export async function getCompanyUsers(
	companyId: string,
	customFetch: typeof fetch = fetch
): Promise<UserResponse[]> {
	const data = await request<{ users: UserResponse[] }>(
		`/api/v1/companies/${companyId}/users`,
		{ method: 'GET' },
		customFetch
	);
	return data.users;
}

export async function createCompanyUser(
	companyId: string,
	payload: CreateStaffPayload,
	customFetch: typeof fetch = fetch
): Promise<UserResponse> {
	const data = await request<{ user: UserResponse }>(
		`/api/v1/companies/${companyId}/users`,
		{
			method: 'POST',
			body: JSON.stringify(payload)
		},
		customFetch
	);
	return data.user;
}
