const DEVICE_KEY = 'notesd_device_id';

export function getDeviceId() {
	if (typeof localStorage === 'undefined') return 'web';
	let id = localStorage.getItem(DEVICE_KEY);
	if (!id) {
		id = 'web-' + crypto.randomUUID().slice(0, 8);
		localStorage.setItem(DEVICE_KEY, id);
	}
	return id;
}
