import type { Action } from 'svelte/action';

/**
 * Action portal — Memindahkan elemen DOM langsung ke target penampung (default: document.body).
 *
 * Mencegah stacking context trapping dari parent layout (overflow-hidden, transform, backdrop-filter).
 * Menjamin elemen global melayang (Toast, Dialog, Drawer) selalu berada di root DOM terluar.
 */
export const portal: Action<HTMLElement, HTMLElement | string | undefined> = (node, target) => {
	if (typeof document === 'undefined') return;

	function resolveTarget(t: HTMLElement | string | undefined): HTMLElement | null {
		if (typeof t === 'string') {
			return document.querySelector<HTMLElement>(t);
		}
		if (t instanceof HTMLElement) {
			return t;
		}
		return document.body;
	}

	let targetEl = resolveTarget(target);
	if (targetEl) {
		targetEl.appendChild(node);
	}

	return {
		update(newTarget) {
			const newTargetEl = resolveTarget(newTarget);
			if (newTargetEl && newTargetEl !== targetEl) {
				newTargetEl.appendChild(node);
				targetEl = newTargetEl;
			}
		},
		destroy() {
			if (node.parentNode) {
				node.parentNode.removeChild(node);
			}
		}
	};
};
