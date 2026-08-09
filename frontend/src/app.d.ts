// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces

export interface User {
	id: string;
	company_id: string;
	first_name: string;
	last_name: string;
	email: string;
	mobile_number?: string;
	system_role: 'super_admin' | 'admin' | 'hr' | 'employee';
	created_at: string;
}

declare global {
	namespace App {
		// interface Error {}
		interface Locals {
			user: User | null;
		}
		interface PageData {
			user: User | null;
		}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};
