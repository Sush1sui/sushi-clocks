import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import type { CompanyWithStats } from '$lib/api/companies';

const BACKEND_URL = 'http://localhost:8080';

export const load: PageServerLoad = async ({ locals, request }) => {
	if (!locals.user) {
		throw redirect(303, '/');
	}

	// RBAC Guard: Non-SuperAdmin users belong to 1 company and are redirected to their company dashboard
	if (locals.user.system_role !== 'super_admin') {
		throw redirect(303, `/dashboard/${locals.user.company_id}`);
	}

	let companies: CompanyWithStats[] = [];
	try {
		const cookieHeader = request.headers.get('cookie') || '';
		const res = await fetch(`${BACKEND_URL}/api/v1/companies`, {
			headers: {
				cookie: cookieHeader
			}
		});

		if (res.ok) {
			const json = await res.json();
			if (json.success && json.data?.companies) {
				companies = json.data.companies;
			}
		}
	} catch (err) {
		console.error('Failed to load companies in superadmin dashboard:', err);
	}

	return {
		companies
	};
};
