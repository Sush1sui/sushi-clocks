/**
 * Lightweight real-time SSE client for live presence, punch events, and attendance updates.
 * Zero external packages; uses native browser EventSource with automatic reconnection.
 */
export type SSEEventHandler = (eventType: string, data: any) => void;

export function createSSEConnection(companyId: string, onEvent: SSEEventHandler): () => void {
	if (typeof window === 'undefined') {
		return () => {};
	}

	let eventSource: EventSource | null = null;
	let isClosed = false;
	let reconnectTimer: any = null;

	function connect() {
		if (isClosed) return;

		// withCredentials sends the HttpOnly access_token cookie
		const url = `/api/v1/events?company_id=${encodeURIComponent(companyId)}`;
		eventSource = new EventSource(url, { withCredentials: true });

		eventSource.addEventListener('connected', (e) => {
			try {
				const data = JSON.parse(e.data);
				onEvent('connected', data);
			} catch {}
		});

		eventSource.addEventListener('punch', (e) => {
			try {
				const data = JSON.parse(e.data);
				onEvent('punch', data);
			} catch (err) {
				console.error('Failed to parse SSE punch event', err);
			}
		});

		eventSource.addEventListener('adjustment_request', (e) => {
			try {
				const data = JSON.parse(e.data);
				onEvent('adjustment_request', data);
			} catch (err) {
				console.error('Failed to parse adjustment_request event', err);
			}
		});

		eventSource.addEventListener('adjustment_resolved', (e) => {
			try {
				const data = JSON.parse(e.data);
				onEvent('adjustment_resolved', data);
			} catch (err) {
				console.error('Failed to parse adjustment_resolved event', err);
			}
		});

		eventSource.addEventListener('timesheet_updated', (e) => {
			try {
				const data = JSON.parse(e.data);
				onEvent('timesheet_updated', data);
			} catch (err) {
				console.error('Failed to parse timesheet_updated event', err);
			}
		});

		eventSource.onerror = () => {
			if (eventSource) {
				eventSource.close();
				eventSource = null;
			}
			if (!isClosed && !reconnectTimer) {
				reconnectTimer = setTimeout(() => {
					reconnectTimer = null;
					connect();
				}, 5000);
			}
		};
	}

	connect();

	return () => {
		isClosed = true;
		if (reconnectTimer) {
			clearTimeout(reconnectTimer);
			reconnectTimer = null;
		}
		if (eventSource) {
			eventSource.close();
			eventSource = null;
		}
	};
}
