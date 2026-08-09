import { request } from './client';
import type { User } from '../../app';

export interface Company {
	id: string;
	name: string;
	currency_code: string;
	timezone: string;
	created_at: string;
}

export interface CompanyWithStats extends Company {
	total_users: number;
	admin_email?: string;
}

export interface CreateCompanyPayload {
	name: string;
	currency_code: string;
	timezone: string;
	admin_first_name: string;
	admin_last_name: string;
	admin_email: string;
	admin_password: string;
}

export interface CreateCompanyResponse {
	company: Company;
	admin?: User;
}

export async function getCompanies(customFetch?: typeof fetch): Promise<CompanyWithStats[]> {
	const data = await request<{ companies: CompanyWithStats[] }>(
		'/api/v1/companies',
		{ method: 'GET' },
		customFetch
	);
	return data.companies;
}

export async function createCompany(
	payload: CreateCompanyPayload,
	customFetch?: typeof fetch
): Promise<CreateCompanyResponse> {
	return request<CreateCompanyResponse>(
		'/api/v1/companies',
		{
			method: 'POST',
			body: JSON.stringify(payload)
		},
		customFetch
	);
}

export async function getCompany(
	id: string,
	customFetch?: typeof fetch
): Promise<Company> {
	const data = await request<{ company: Company }>(
		`/api/v1/companies/${id}`,
		{ method: 'GET' },
		customFetch
	);
	return data.company;
}
