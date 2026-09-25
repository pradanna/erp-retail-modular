<script lang="ts">
	import { onMount } from 'svelte';

	interface Props {
		latitude?: number | null | undefined;
		longitude?: number | null | undefined;
		label?: string;
		placeholder?: string;
		readonly?: boolean;
		height?: string;
		zoom?: number;
		class?: string;
	}

	let {
		latitude = $bindable(null),
		longitude = $bindable(null),
		label = 'Titik Koordinat & Peta Lokasi',
		placeholder = 'Cari nama kota, daerah, atau jalan...',
		readonly = false,
		height = '280px',
		zoom = 13,
		class: className = ''
	}: Props = $props();

	let mapContainer: HTMLDivElement | null = $state(null);
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let mapInstance: any = null;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let markerInstance: any = null;
	let leafletLoaded = $state(false);
	let mapReady = $state(false);

	// Geocoding search state
	let searchQuery = $state('');
	let searching = $state(false);
	let searchResults = $state<Array<{ display_name: string; lat: string; lon: string }>>([]);
	let showResultsDropdown = $state(false);
	let geolocationLoading = $state(false);

	// Default fallback: Monas, Jakarta Pusat
	const DEFAULT_LAT = -6.175392;
	const DEFAULT_LNG = 106.827153;

	const latInputId = `map-lat-${Math.random().toString(36).slice(2, 8)}`;
	const lngInputId = `map-lng-${Math.random().toString(36).slice(2, 8)}`;

	// Load Leaflet Script & CSS dynamically on client
	async function loadLeaflet(): Promise<void> {
		if (typeof window === 'undefined') return;

		// Check if already in window
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		if ((window as any).L) {
			leafletLoaded = true;
			return;
		}

		// Inject CSS
		if (!document.getElementById('leaflet-css')) {
			const link = document.createElement('link');
			link.id = 'leaflet-css';
			link.rel = 'stylesheet';
			link.href = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.css';
			document.head.appendChild(link);
		}

		// Inject JS
		return new Promise((resolve, reject) => {
			if (document.getElementById('leaflet-js')) {
				const checkInterval = setInterval(() => {
					// eslint-disable-next-line @typescript-eslint/no-explicit-any
					if ((window as any).L) {
						clearInterval(checkInterval);
						leafletLoaded = true;
						resolve();
					}
				}, 50);
				return;
			}

			const script = document.createElement('script');
			script.id = 'leaflet-js';
			script.src = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js';
			script.async = true;
			script.onload = () => {
				leafletLoaded = true;
				resolve();
			};
			script.onerror = (err) => reject(err);
			document.head.appendChild(script);
		});
	}

	function initMap() {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		const L = (window as any).L;
		if (!L || !mapContainer || mapInstance) return;

		const currentLat = latitude ?? DEFAULT_LAT;
		const currentLng = longitude ?? DEFAULT_LNG;
		const initialZoom = latitude && longitude ? zoom : 11;

		mapInstance = L.map(mapContainer, {
			center: [currentLat, currentLng],
			zoom: initialZoom,
			attributionControl: false
		});

		L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
			maxZoom: 19
		}).addTo(mapInstance);

		// Custom SVG Marker Icon matching Obsidian / Monochrome Design System
		const customIcon = L.divIcon({
			className: 'custom-erp-map-pin',
			html: `
				<div style="position: relative; width: 34px; height: 34px; display: flex; align-items: center; justify-content: center;">
					<div style="width: 32px; height: 32px; background: #0f172a; border: 2.5px solid #ffffff; border-radius: 50% 50% 50% 0; transform: rotate(-45deg); box-shadow: 0 4px 10px rgba(0,0,0,0.35); display: flex; align-items: center; justify-content: center;">
						<div style="width: 10px; height: 10px; background: #38bdf8; border-radius: 50%; transform: rotate(45deg);"></div>
					</div>
				</div>
			`,
			iconSize: [34, 34],
			iconAnchor: [17, 34],
			popupAnchor: [0, -34]
		});

		if (latitude !== null && latitude !== undefined && longitude !== null && longitude !== undefined) {
			markerInstance = L.marker([latitude, longitude], {
				icon: customIcon,
				draggable: !readonly
			}).addTo(mapInstance);

			if (!readonly) {
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				markerInstance.on('dragend', (e: any) => {
					const pos = e.target.getLatLng();
					latitude = parseFloat(pos.lat.toFixed(8));
					longitude = parseFloat(pos.lng.toFixed(8));
				});
			}
		}

		if (!readonly) {
			// Click to place / move marker
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			mapInstance.on('click', (e: any) => {
				const lat = parseFloat(e.latlng.lat.toFixed(8));
				const lng = parseFloat(e.latlng.lng.toFixed(8));
				setCoordinates(lat, lng);
			});
		}

		mapReady = true;
	}

	function setCoordinates(lat: number, lng: number) {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		const L = (window as any).L;
		latitude = lat;
		longitude = lng;

		if (!mapInstance || !L) return;

		const customIcon = L.divIcon({
			className: 'custom-erp-map-pin',
			html: `
				<div style="position: relative; width: 34px; height: 34px; display: flex; align-items: center; justify-content: center;">
					<div style="width: 32px; height: 32px; background: #0f172a; border: 2.5px solid #ffffff; border-radius: 50% 50% 50% 0; transform: rotate(-45deg); box-shadow: 0 4px 10px rgba(0,0,0,0.35); display: flex; align-items: center; justify-content: center;">
						<div style="width: 10px; height: 10px; background: #38bdf8; border-radius: 50%; transform: rotate(45deg);"></div>
					</div>
				</div>
			`,
			iconSize: [34, 34],
			iconAnchor: [17, 34]
		});

		if (markerInstance) {
			markerInstance.setLatLng([lat, lng]);
		} else {
			markerInstance = L.marker([lat, lng], {
				icon: customIcon,
				draggable: !readonly
			}).addTo(mapInstance);

			if (!readonly) {
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				markerInstance.on('dragend', (ev: any) => {
					const pos = ev.target.getLatLng();
					latitude = parseFloat(pos.lat.toFixed(8));
					longitude = parseFloat(pos.lng.toFixed(8));
				});
			}
		}

		mapInstance.panTo([lat, lng]);
	}

	// Geocoding via OpenStreetMap Nominatim
	async function searchLocation() {
		if (!searchQuery.trim()) return;
		searching = true;
		showResultsDropdown = true;
		try {
			const url = `https://nominatim.openstreetmap.org/search?format=json&q=${encodeURIComponent(
				searchQuery.trim()
			)}&limit=5&countrycodes=id`;
			const res = await fetch(url, {
				headers: {
					'Accept-Language': 'id'
				}
			});
			if (res.ok) {
				const data = await res.json();
				searchResults = data;
			}
		} catch (err) {
			console.error('Geocoding search failed:', err);
		} finally {
			searching = false;
		}
	}

	function selectSearchResult(res: { display_name: string; lat: string; lon: string }) {
		const lat = parseFloat(parseFloat(res.lat).toFixed(8));
		const lng = parseFloat(parseFloat(res.lon).toFixed(8));
		setCoordinates(lat, lng);
		showResultsDropdown = false;
		searchQuery = res.display_name.split(',')[0];
	}

	function handleGetCurrentLocation() {
		if (!navigator.geolocation) {
			alert('Browser Anda tidak mendukung deteksi lokasi (Geolocation API).');
			return;
		}
		geolocationLoading = true;
		navigator.geolocation.getCurrentPosition(
			(pos) => {
				const lat = parseFloat(pos.coords.latitude.toFixed(8));
				const lng = parseFloat(pos.coords.longitude.toFixed(8));
				setCoordinates(lat, lng);
				geolocationLoading = false;
			},
			(err) => {
				console.error('Geolocation error:', err);
				geolocationLoading = false;
				alert('Gagal mendapatkan lokasi saat ini. Pastikan izin lokasi diizinkan di peramban.');
			},
			{ enableHighAccuracy: true, timeout: 10000 }
		);
	}

	function clearCoordinates() {
		latitude = null;
		longitude = null;
		if (markerInstance && mapInstance) {
			mapInstance.removeLayer(markerInstance);
			markerInstance = null;
		}
	}

	onMount(() => {
		loadLeaflet().then(() => {
			initMap();
		});

		return () => {
			if (mapInstance) {
				mapInstance.remove();
				mapInstance = null;
			}
		};
	});

	// Synchronize when inputs change externally
	$effect(() => {
		if (mapReady && mapInstance && latitude !== null && latitude !== undefined && longitude !== null && longitude !== undefined) {
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			const L = (window as any).L;
			if (!markerInstance && L) {
				setCoordinates(latitude, longitude);
			} else if (markerInstance) {
				const current = markerInstance.getLatLng();
				if (current.lat !== latitude || current.lng !== longitude) {
					markerInstance.setLatLng([latitude, longitude]);
					mapInstance.panTo([latitude, longitude]);
				}
			}
		}
	});
</script>

<div class="flex flex-col gap-2.5 {className}">
	{#if label}
		<div class="flex items-center justify-between">
			<span class="text-xs font-semibold tracking-wide text-neutral-800 uppercase">
				{label}
			</span>
			{#if latitude !== null && latitude !== undefined && longitude !== null && longitude !== undefined}
				<a
					href="https://www.google.com/maps?q={latitude},{longitude}"
					target="_blank"
					rel="noreferrer"
					class="inline-flex items-center gap-1 text-2xs font-medium text-primary-600 hover:text-primary-700 hover:underline"
				>
					<!-- External link icon -->
					<svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M13.5 6H5.25A2.25 2.25 0 003 8.25v10.5A2.25 2.25 0 005.25 21h10.5A2.25 2.25 0 0018 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25" />
					</svg>
					Google Maps
				</a>
			{/if}
		</div>
	{/if}

	<!-- Search & Tool Buttons Bar (Only if not readonly) -->
	{#if !readonly}
		<div class="relative flex flex-col gap-2 sm:flex-row sm:items-center">
			<div class="relative flex-1">
				<input
					type="text"
					bind:value={searchQuery}
					onkeydown={(e) => e.key === 'Enter' && (e.preventDefault(), searchLocation())}
					{placeholder}
					class="w-full rounded-lg border border-neutral-300 bg-white py-1.5 pr-8 pl-8 text-xs text-neutral-800 placeholder-neutral-400 shadow-2xs transition-colors focus:border-neutral-900 focus:outline-hidden focus:ring-1 focus:ring-neutral-900"
				/>
				<!-- Search Icon -->
				<div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-2.5 text-neutral-400">
					<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
					</svg>
				</div>
				{#if searching}
					<div class="absolute inset-y-0 right-0 flex items-center pr-2.5 text-neutral-400">
						<svg class="h-3.5 w-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					</div>
				{/if}

				<!-- Dropdown Hasil Pencarian -->
				{#if showResultsDropdown && searchResults.length > 0}
					<div class="absolute z-50 mt-1 max-h-52 w-full overflow-y-auto rounded-lg border border-neutral-200 bg-white py-1 shadow-lg">
						{#each searchResults as item}
							<button
								type="button"
								onclick={() => selectSearchResult(item)}
								class="flex w-full items-start gap-2 px-3 py-1.5 text-left text-xs text-neutral-700 transition-colors hover:bg-neutral-100"
							>
								<!-- Map Pin -->
								<svg class="mt-0.5 h-3.5 w-3.5 shrink-0 text-neutral-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M15 10.5a3 3 0 11-6 0 3 3 0 016 0z" />
									<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 10.5c0 7.142-7.5 11.25-7.5 11.25S4.5 17.642 4.5 10.5a7.5 7.5 0 1115 0z" />
								</svg>
								<span class="line-clamp-2 leading-tight">{item.display_name}</span>
							</button>
						{/each}
					</div>
				{/if}
			</div>

			<div class="flex items-center gap-1.5 shrink-0">
				<button
					type="button"
					onclick={searchLocation}
					disabled={searching || !searchQuery.trim()}
					class="inline-flex items-center gap-1 rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-xs font-medium text-neutral-700 shadow-2xs transition-colors hover:bg-neutral-50 disabled:opacity-50"
				>
					Cari
				</button>

				<button
					type="button"
					onclick={handleGetCurrentLocation}
					disabled={geolocationLoading}
					class="inline-flex items-center gap-1 rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-xs font-medium text-neutral-700 shadow-2xs transition-colors hover:bg-neutral-50 disabled:opacity-50"
					title="Gunakan lokasi GPS saat ini"
				>
					<!-- Compass / Crosshair -->
					<svg class="h-3.5 w-3.5 text-neutral-600 {geolocationLoading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 2v2m0 16v2M2 12h2m16 0h2m-6 0a4 4 0 11-8 0 4 4 0 018 0z" />
					</svg>
					Lokasi Saya
				</button>

				{#if latitude !== null && longitude !== null}
					<button
						type="button"
						onclick={clearCoordinates}
						class="inline-flex items-center rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs font-medium text-rose-600 shadow-2xs transition-colors hover:bg-rose-50"
						title="Hapus koordinat"
					>
						Reset
					</button>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Map Container -->
	<div
		class="relative overflow-hidden rounded-xl border border-neutral-300/80 bg-neutral-100 shadow-2xs"
		style="height: {height};"
	>
		<div bind:this={mapContainer} class="h-full w-full"></div>

		{#if !leafletLoaded}
			<div class="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-neutral-100 text-neutral-500">
				<svg class="h-6 w-6 animate-spin text-neutral-600" fill="none" viewBox="0 0 24 24">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
					<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
				</svg>
				<span class="text-xs font-medium">Memuat peta OpenStreetMap...</span>
			</div>
		{/if}

		{#if !readonly}
			<div class="pointer-events-none absolute right-2 bottom-2 z-400 rounded-md bg-white/90 px-2 py-1 text-3xs font-medium text-neutral-600 backdrop-blur-xs shadow-xs">
				Klik peta untuk menentukan titik
			</div>
		{/if}
	</div>

	<!-- Coordinate Inputs Row -->
	<div class="grid grid-cols-2 gap-3">
		<div>
			<label for={latInputId} class="block text-2xs font-medium text-neutral-500 mb-1">Latitude</label>
			<input
				id={latInputId}
				type="number"
				step="any"
				placeholder="-6.175392"
				disabled={readonly}
				value={latitude ?? ''}
				oninput={(e) => {
					const val = (e.target as HTMLInputElement).value;
					latitude = val === '' ? null : parseFloat(val);
				}}
				class="w-full rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-xs font-mono text-neutral-800 placeholder-neutral-400 shadow-2xs transition-colors focus:border-neutral-900 focus:outline-hidden focus:ring-1 focus:ring-neutral-900 disabled:bg-neutral-50 disabled:text-neutral-500"
			/>
		</div>

		<div>
			<label for={lngInputId} class="block text-2xs font-medium text-neutral-500 mb-1">Longitude</label>
			<input
				id={lngInputId}
				type="number"
				step="any"
				placeholder="106.827153"
				disabled={readonly}
				value={longitude ?? ''}
				oninput={(e) => {
					const val = (e.target as HTMLInputElement).value;
					longitude = val === '' ? null : parseFloat(val);
				}}
				class="w-full rounded-lg border border-neutral-300 bg-white px-2.5 py-1.5 text-xs font-mono text-neutral-800 placeholder-neutral-400 shadow-2xs transition-colors focus:border-neutral-900 focus:outline-hidden focus:ring-1 focus:ring-neutral-900 disabled:bg-neutral-50 disabled:text-neutral-500"
			/>
		</div>
	</div>
</div>
