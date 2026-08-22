import { request } from './client';

export interface Timesheet {
	id: string;
	user_id: string;
	company_id: string;
	clock_in_time: string;
	clock_out_time?: string | null;
	status: 'active' | 'completed' | 'flagged_for_review';
	created_at: string;
}

export interface TimesheetStatusResponse {
	is_clocked_in: boolean;
	shift: Timesheet | null;
}

export interface AttendanceSummary {
	total_staff: number;
	clocked_in_count: number;
	clocked_out_count: number;
}

export interface AttendanceSummaryResponse {
	summary: AttendanceSummary;
}

export async function clockIn(customFetch?: typeof fetch): Promise<{ message: string; timesheet: Timesheet }> {
	return request<{ message: string; timesheet: Timesheet }>(
		'/api/v1/timesheets/clock-in',
		{ method: 'POST' },
		customFetch
	);
}

export async function clockOut(customFetch?: typeof fetch): Promise<{ message: string; timesheet: Timesheet }> {
	return request<{ message: string; timesheet: Timesheet }>(
		'/api/v1/timesheets/clock-out',
		{ method: 'POST' },
		customFetch
	);
}

export async function getTimesheetStatus(customFetch?: typeof fetch): Promise<TimesheetStatusResponse> {
	return request<TimesheetStatusResponse>(
		'/api/v1/timesheets/status',
		{ method: 'GET' },
		customFetch
	);
}

export async function getCompanyAttendanceSummary(
	companyId: string,
	customFetch?: typeof fetch
): Promise<AttendanceSummary> {
	const data = await request<AttendanceSummaryResponse>(
		`/api/v1/companies/${companyId}/attendance/summary`,
		{ method: 'GET' },
		customFetch
	);
	return data.summary;
}
