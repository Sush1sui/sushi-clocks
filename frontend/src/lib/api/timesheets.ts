import { request } from './client';

export interface Timesheet {
	id: string;
	user_id: string;
	company_id: string;
	clock_in_time: string;
	clock_out_time?: string | null;
	status: 'active' | 'completed' | 'flagged_for_review' | 'rejected';
	adjustment_reason?: string | null;
	reviewed_by?: string | null;
	reviewed_at?: string | null;
	created_at: string;
}

export interface TimesheetWithUser extends Timesheet {
	user_email: string;
	user_first_name: string;
	user_last_name: string;
	role_name: string;
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

export interface LiveRosterResponse {
	roster: TimesheetWithUser[];
}

export interface ShiftHistoryResponse {
	history: Timesheet[];
	page: number;
	limit: number;
	total: number;
}

export interface AdjustmentsResponse {
	adjustments: TimesheetWithUser[];
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

export async function getLiveRoster(
	companyId: string,
	customFetch?: typeof fetch
): Promise<TimesheetWithUser[]> {
	const data = await request<LiveRosterResponse>(
		`/api/v1/companies/${companyId}/attendance/roster`,
		{ method: 'GET' },
		customFetch
	);
	return data.roster;
}

export async function getShiftHistory(
	page = 1,
	limit = 10,
	customFetch?: typeof fetch
): Promise<ShiftHistoryResponse> {
	return request<ShiftHistoryResponse>(
		`/api/v1/timesheets/history?page=${page}&limit=${limit}`,
		{ method: 'GET' },
		customFetch
	);
}

export async function requestAdjustment(
	timesheetId: string,
	reason: string,
	customFetch?: typeof fetch
): Promise<{ message: string; timesheet: Timesheet }> {
	return request<{ message: string; timesheet: Timesheet }>(
		`/api/v1/timesheets/${timesheetId}/adjustment-request`,
		{
			method: 'POST',
			body: JSON.stringify({ reason })
		},
		customFetch
	);
}

export async function getPendingAdjustments(
	companyId: string,
	customFetch?: typeof fetch
): Promise<TimesheetWithUser[]> {
	const data = await request<AdjustmentsResponse>(
		`/api/v1/companies/${companyId}/adjustments`,
		{ method: 'GET' },
		customFetch
	);
	return data.adjustments;
}

export async function resolveAdjustment(
	timesheetId: string,
	action: 'approve' | 'reject',
	newClockIn?: string,
	newClockOut?: string,
	customFetch?: typeof fetch
): Promise<{ message: string; timesheet: Timesheet }> {
	return request<{ message: string; timesheet: Timesheet }>(
		`/api/v1/timesheets/adjustments/${timesheetId}`,
		{
			method: 'PATCH',
			body: JSON.stringify({
				action,
				new_clock_in: newClockIn ? new Date(newClockIn).toISOString() : undefined,
				new_clock_out: newClockOut ? new Date(newClockOut).toISOString() : undefined
			})
		},
		customFetch
	);
}

export async function directOverrideTimesheet(
	timesheetId: string,
	clockInTime: string,
	clockOutTime: string | null,
	reason: string,
	customFetch?: typeof fetch
): Promise<{ message: string; timesheet: Timesheet }> {
	return request<{ message: string; timesheet: Timesheet }>(
		`/api/v1/timesheets/${timesheetId}`,
		{
			method: 'PUT',
			body: JSON.stringify({
				clock_in_time: new Date(clockInTime).toISOString(),
				clock_out_time: clockOutTime ? new Date(clockOutTime).toISOString() : null,
				reason
			})
		},
		customFetch
	);
}
