import { redirect, error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import type { Company } from '$lib/api/companies';
import type { TimesheetStatusResponse, AttendanceSummary } from '$lib/api/timesheets';

const BACKEND_URL = 'http://localhost:8080';

export const load: PageServerLoad = async ({ locals, params, request }) => {
	if (!locals.user) {
		throw redirect(303, '/');
	}

	const targetCompanyId = params.companyId;

	// Strict Multi-Tenant RBAC Guard:
	// If non-superadmin tries to access a different company's dashboard, immediately redirect them to their own
	if (locals.user.system_role !== 'super_admin' && targetCompanyId !== locals.user.company_id) {
		throw redirect(303, `/dashboard/${locals.user.company_id}`);
	}

	const cookieHeader = request.headers.get('cookie') || '';
	const headers = { cookie: cookieHeader };

	let company: Company | null = null;
	try {
		const res = await fetch(`${BACKEND_URL}/api/v1/companies/${targetCompanyId}`, { headers });

		if (res.ok) {
			const json = await res.json();
			if (json.success && json.data?.company) {
				company = json.data.company;
			}
		} else if (res.status === 404) {
			throw error(404, 'Organization not found');
		} else if (res.status === 403) {
			throw redirect(303, `/dashboard/${locals.user.company_id}`);
		}
	} catch (err: unknown) {
		if (err && typeof err === 'object' && 'status' in err) {
			throw err;
		}
		console.error('Failed to load company details:', err);
	}

	if (!company) {
		throw error(404, 'Organization not found or accessible');
	}

	// 2. Fetch User's Timesheet Status
	let timesheetStatus: TimesheetStatusResponse | null = null;
	try {
		const res = await fetch(`${BACKEND_URL}/api/v1/timesheets/status`, { headers });
		if (res.ok) {
			const json = await res.json();
			if (json.success && json.data) {
				timesheetStatus = json.data;
			}
		}
	} catch (err) {
		console.error('Failed to load timesheet status:', err);
	}

	// 3. Fetch Company Attendance Summary (for Admin, HR, Super Admin)
	let attendanceSummary: AttendanceSummary | null = null;
	if (
		locals.user.system_role === 'super_admin' ||
		locals.user.system_role === 'admin' ||
		locals.user.system_role === 'hr'
	) {
		try {
			const res = await fetch(`${BACKEND_URL}/api/v1/companies/${targetCompanyId}/attendance/summary`, { headers });
			if (res.ok) {
				const json = await res.json();
				if (json.success && json.data?.summary) {
					attendanceSummary = json.data.summary;
				}
			}
		} catch (err) {
			console.error('Failed to load attendance summary:', err);
		}
	}

	return {
		company,
		timesheetStatus,
		attendanceSummary
	};
};
