import { request } from './client';

export interface PayrollModifier {
	id: string;
	name: string;
	modifier_type: 'addition' | 'deduction';
	calculation_method: 'fixed' | 'percentage';
	value: number;
	amount: number;
}

export interface EmployeePayrollLine {
	user_id: string;
	first_name: string;
	last_name: string;
	email: string;
	role_name: string;
	currency_code: string;
	wage_type: 'hourly' | 'daily' | 'monthly';
	base_rate: number;
	wage_source: 'override' | 'role' | 'none';
	worked_minutes: number;
	worked_days: number;
	paid_leave_days: number;
	unpaid_leave_days: number;
	gross_pay: number;
	additions: number;
	deductions: number;
	net_pay: number;
	modifiers: PayrollModifier[];
}

export interface PayrollPreview {
	company_id: string;
	currency_code: string;
	period_start: string;
	period_end: string;
	employees: EmployeePayrollLine[];
	total_gross: number;
	total_net: number;
}

export async function calculatePayroll(
	companyId: string,
	startDate: string,
	endDate: string,
	customFetch?: typeof fetch
): Promise<PayrollPreview> {
	return request<PayrollPreview>(
		`/api/v1/companies/${companyId}/payroll/calculate?start_date=${encodeURIComponent(startDate)}&end_date=${encodeURIComponent(endDate)}`,
		{ method: 'GET' },
		customFetch
	);
}

export async function exportPayrollCSV(
	companyId: string,
	startDate: string,
	endDate: string
): Promise<void> {
	const url = `/api/v1/companies/${companyId}/payroll/export?start_date=${encodeURIComponent(startDate)}&end_date=${encodeURIComponent(endDate)}`;
	const res = await fetch(url, {
		method: 'GET',
		credentials: 'include'
	});
	if (!res.ok) {
		throw new Error(`Failed to export CSV: ${res.statusText}`);
	}
	const blob = await res.blob();
	const downloadUrl = window.URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = downloadUrl;
	a.download = `payroll_${startDate}.csv`;
	document.body.appendChild(a);
	a.click();
	window.URL.revokeObjectURL(downloadUrl);
	document.body.removeChild(a);
}
