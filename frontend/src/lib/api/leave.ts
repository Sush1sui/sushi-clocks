import { request } from './client';

export interface LeaveType {
	id: string;
	company_id: string;
	name: string;
	is_paid: boolean;
	allow_carryover: boolean;
	max_carryover_days: number;
	created_at: string;
}

export interface LeavePeriod {
	start_date: string;
	end_date: string;
	prev_start_date: string;
	prev_end_date: string;
}

export interface LeaveBalance {
	leave_type_id: string;
	leave_type_name: string;
	is_paid: boolean;
	base_allocated_days: number;
	carried_over_days: number;
	total_allocated_days: number;
	used_days: number;
	pending_days: number;
	remaining_days: number;
	is_flexible: boolean;
}

export interface LeaveBalancesResponse {
	balances: LeaveBalance[];
	period: LeavePeriod;
}

export interface LeaveRequest {
	id: string;
	company_id: string;
	user_id: string;
	leave_type_id: string;
	start_date: string;
	end_date: string;
	status: 'pending' | 'approved' | 'rejected';
	reason?: string | null;
	reviewed_by_user_id?: string | null;
	reviewed_at?: string | null;
	review_notes?: string | null;
	created_at: string;
	user_first_name?: string;
	user_last_name?: string;
	user_email?: string;
	role_name?: string;
	leave_type_name?: string;
	is_paid?: boolean;
	calendar_days?: number;
}

export interface CompanyLeavePolicy {
	company_id: string;
	leave_reset_month: number;
	leave_reset_day: number;
	allow_leave_carryover: boolean;
	max_carryover_days: number;
}

export interface SubmitLeavePayload {
	leave_type_id: string;
	start_date: string;
	end_date: string;
	reason: string;
}

export interface ResolveLeavePayload {
	action: 'approve' | 'reject';
	review_notes?: string;
}

export interface UpdateLeavePolicyPayload {
	leave_reset_month: number;
	leave_reset_day: number;
	allow_leave_carryover: boolean;
	max_carryover_days: number;
}

export async function getLeaveTypes(customFetch?: typeof fetch): Promise<LeaveType[]> {
	const res = await request<{ types: LeaveType[] }>('/api/v1/leave/types', { method: 'GET' }, customFetch);
	return res.types;
}

export async function getLeaveBalances(
	userId?: string,
	customFetch?: typeof fetch
): Promise<LeaveBalancesResponse> {
	const url = userId ? `/api/v1/leave/balances?user_id=${encodeURIComponent(userId)}` : '/api/v1/leave/balances';
	return request<LeaveBalancesResponse>(url, { method: 'GET' }, customFetch);
}

export async function submitLeaveRequest(
	payload: SubmitLeavePayload,
	customFetch?: typeof fetch
): Promise<{ message: string; leave_request: LeaveRequest; calendar_days: number }> {
	return request<{ message: string; leave_request: LeaveRequest; calendar_days: number }>(
		'/api/v1/leave/requests',
		{
			method: 'POST',
			body: JSON.stringify(payload)
		},
		customFetch
	);
}

export async function getMyLeaveRequests(
	page = 1,
	limit = 20,
	customFetch?: typeof fetch
): Promise<{ requests: LeaveRequest[]; total: number; limit: number; offset: number }> {
	return request<{ requests: LeaveRequest[]; total: number; limit: number; offset: number }>(
		`/api/v1/leave/requests?page=${page}&limit=${limit}`,
		{ method: 'GET' },
		customFetch
	);
}

export async function getCompanyLeaveRequests(
	companyId: string,
	status = 'all',
	page = 1,
	limit = 50,
	customFetch?: typeof fetch
): Promise<{ requests: LeaveRequest[]; total: number; limit: number; offset: number }> {
	const url = `/api/v1/companies/${companyId}/leave-requests?status=${status}&page=${page}&limit=${limit}`;
	return request<{ requests: LeaveRequest[]; total: number; limit: number; offset: number }>(
		url,
		{ method: 'GET' },
		customFetch
	);
}

export async function resolveLeaveRequest(
	requestId: string,
	action: 'approve' | 'reject',
	reviewNotes?: string,
	customFetch?: typeof fetch
): Promise<{ message: string; leave_request: LeaveRequest }> {
	return request<{ message: string; leave_request: LeaveRequest }>(
		`/api/v1/leave/requests/${requestId}`,
		{
			method: 'PATCH',
			body: JSON.stringify({ action, review_notes: reviewNotes })
		},
		customFetch
	);
}

export async function getCompanyLeavePolicy(
	companyId: string,
	customFetch?: typeof fetch
): Promise<CompanyLeavePolicy> {
	const res = await request<{ policy: CompanyLeavePolicy }>(
		`/api/v1/companies/${companyId}/leave-policy`,
		{ method: 'GET' },
		customFetch
	);
	return res.policy;
}

export async function updateCompanyLeavePolicy(
	companyId: string,
	payload: UpdateLeavePolicyPayload,
	customFetch?: typeof fetch
): Promise<{ message: string; policy: CompanyLeavePolicy }> {
	return request<{ message: string; policy: CompanyLeavePolicy }>(
		`/api/v1/companies/${companyId}/leave-policy`,
		{
			method: 'PUT',
			body: JSON.stringify(payload)
		},
		customFetch
	);
}
