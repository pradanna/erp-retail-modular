<script lang="ts">
	/**
	 * Login Page — Halaman Autentikasi Backoffice ElectroERP.
	 *
	 * Mengikuti aturan & arsitektur:
	 * - Desain split-screen presisi sesuai mockup ElectroERP Retail Management System
	 * - 100% menggunakan komponen atomic dari @erp/ui (Input, Button, Checkbox, Alert)
	 * - Zero emoji, zero sparkle
	 * - Heroicons SVG standard 24x24 outline
	 * - Svelte 5 runes ($state, $derived)
	 * - Responsif: Left showcase di desktop, hidden di layar kecil (mobile-first)
	 */
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { login, isAuthenticated, isLoading, getError, clearError } from '$lib/stores/auth.svelte';
	import { Input, Button, Checkbox, Alert } from '@erp/ui';

	let username = $state('');
	let password = $state('');
	let rememberMe = $state(false);
	let noticeMessage = $state<string | null>(null);

	onMount(() => {
		if (isAuthenticated()) {
			goto(resolve('/dashboard'));
		}
	});

	async function handleLogin(e: SubmitEvent) {
		e.preventDefault();
		clearError();
		noticeMessage = null;

		const success = await login(username, password);
		if (success) {
			await goto(resolve('/dashboard'));
		}
	}

	function handleForgotPassword(e: MouseEvent) {
		e.preventDefault();
		noticeMessage =
			'Untuk mereset password, silakan hubungi Superadmin atau IT Administrator toko Anda.';
	}

	function handleGoogleLogin() {
		noticeMessage = 'Fitur Masuk dengan Google (Google OAuth SSO) sedang disiapkan.';
	}
</script>

<svelte:head>
	<title>Gen-E — GEN ENTERPRISE Modular ERP System</title>
</svelte:head>

<div class="flex min-h-screen w-full bg-neutral-900">
	<!-- ========================================================
       KIRI: Showcase & Hero Banner (Desktop Only)
       ======================================================== -->
	<aside
		aria-label="Showcase Modul ERP"
		class="relative hidden w-full overflow-hidden bg-slate-950 lg:flex lg:w-[54%] xl:w-[57%] 2xl:w-[60%]"
	>
		<!-- Background Image Interior Toko Elektronik -->
		<img
			src="/images/electronics_store_bg.jpg"
			alt="Interior Toko Retail Elektronik Modern"
			class="absolute inset-0 h-full w-full object-cover object-center brightness-[0.78] contrast-[1.08] filter"
		/>

		<!-- Gradient Overlay Biru & Slate Gelap untuk Kontras Maksimal -->
		<div
			class="absolute inset-0 bg-gradient-to-tr from-slate-950 via-slate-950/80 to-blue-950/70 backdrop-blur-[1px]"
		></div>

		<!-- Konten Showcase -->
		<div
			class="relative z-10 flex h-full w-full flex-col justify-between p-8 text-white xl:p-12 2xl:p-16"
		>
			<!-- Top Brand Header -->
			<div class="flex items-center gap-3.5">
				<img
					src="/brand/gene_mark_white.png"
					alt="Gen-E Logo"
					class="h-11 w-11 object-contain drop-shadow-md"
				/>
				<div>
					<div class="flex items-center gap-1.5 text-xl font-bold tracking-tight text-white">
						GEN <span class="font-extrabold text-neutral-300">ENTERPRISE</span>
					</div>
					<p class="text-xs font-normal tracking-wide text-neutral-400">
						Modular ERP System &bull; Gen-E
					</p>
				</div>
			</div>

			<!-- Center Hero -->
			<div class="my-auto max-w-xl py-12">
				<!-- Tagline Eyebrow -->
				<div class="mb-4 text-xs font-semibold tracking-widest text-neutral-300 uppercase">
					Satu Sistem, Semua Proses
				</div>

				<!-- Main Title -->
				<h1
					class="text-3xl leading-tight font-extrabold tracking-tight text-white xl:text-4xl 2xl:text-5xl"
				>
					Solusi Retail Modern<br />
					Lebih
					<span
						class="bg-gradient-to-r from-white via-neutral-200 to-neutral-400 bg-clip-text text-transparent"
						>Mudah &amp; Efisien</span
					>
				</h1>

				<!-- Subtitle Description -->
				<p class="mt-5 max-w-lg text-sm leading-relaxed text-neutral-300 xl:text-base">
					Gen-E menghubungkan seluruh proses operasional toko, transaksi kasir, hingga belanja
					online pelanggan dalam satu ekosistem modular terintegrasi.
				</p>
			</div>

			<!-- Bottom Tagline & Accent Bar -->
			<div class="flex items-center gap-3 border-t border-white/10 pt-6">
				<div class="h-8 w-1 rounded-full bg-gradient-to-b from-white to-neutral-500"></div>
				<div>
					<div class="text-xs font-bold tracking-wider text-neutral-200 uppercase">
						Smart Retail
					</div>
					<div class="text-[10px] font-medium tracking-widest text-neutral-400 uppercase">
						For a Brighter Future
					</div>
				</div>
			</div>
		</div>
	</aside>

	<!-- ========================================================
       KANAN: Form Autentikasi Pengguna
       ======================================================== -->
	<main
		class="flex min-h-screen w-full flex-1 flex-col justify-between bg-white px-6 py-8 sm:px-10 lg:w-[46%] xl:w-[43%] xl:px-14 2xl:w-[40%]"
	>
		<!-- Kontainer Tengah Form -->
		<div class="mx-auto my-auto w-full max-w-md">
			<!-- Welcome Title -->
			<div>
				<h2 class="text-2xl font-bold tracking-tight text-neutral-900 sm:text-3xl">
					Selamat Datang
				</h2>
				<p class="mt-2 text-sm text-neutral-500">
					Masuk ke akun Anda untuk melanjutkan ke sistem Gen-E.
				</p>
			</div>

			<!-- Notifikasi Error dari Backend -->
			{#if getError()}
				<div class="mt-5">
					<Alert variant="error" dismissible>
						{getError()}
					</Alert>
				</div>
			{/if}

			<!-- Notifikasi Info / Toast Lokal -->
			{#if noticeMessage}
				<div class="mt-5">
					<Alert variant="info" dismissible>
						{noticeMessage}
					</Alert>
				</div>
			{/if}

			<!-- Form Login -->
			<form onsubmit={handleLogin} class="mt-6 space-y-5">
				<!-- Field Username / Email -->
				<Input
					id="login-username"
					label="Username / Email"
					type="text"
					placeholder="Masukkan username atau email"
					bind:value={username}
					required
					autocomplete="username"
					disabled={isLoading()}
				>
					{#snippet leadingIcon()}
						<!-- Heroicons User -->
						<svg
							class="h-5 w-5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z"
							/>
						</svg>
					{/snippet}
				</Input>

				<!-- Field Password dengan Password Toggle -->
				<Input
					id="login-password"
					label="Password"
					type="password"
					placeholder="Masukkan password"
					bind:value={password}
					required
					autocomplete="current-password"
					disabled={isLoading()}
					showPasswordToggle={true}
				>
					{#snippet leadingIcon()}
						<!-- Heroicons LockClosed -->
						<svg
							class="h-5 w-5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z"
							/>
						</svg>
					{/snippet}
				</Input>

				<!-- Baris Pilihan: Ingat Saya & Lupa Password -->
				<div class="flex items-center justify-between pt-1">
					<Checkbox
						id="remember-me"
						label="Ingat saya"
						bind:checked={rememberMe}
						disabled={isLoading()}
					/>

					<a
						href="#forgot-password"
						onclick={handleForgotPassword}
						class="text-sm font-medium text-neutral-700 underline-offset-4 transition-colors hover:text-black hover:underline"
					>
						Lupa password?
					</a>
				</div>

				<!-- Tombol Masuk Utama -->
				<div class="pt-2">
					<Button
						type="submit"
						variant="primary"
						size="lg"
						fullWidth
						loading={isLoading()}
						disabled={!username.trim() || !password.trim()}
					>
						<span class="inline-flex items-center justify-center gap-2">
							Masuk
							<!-- Heroicons ArrowRight -->
							<svg
								class="h-4 w-4"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="2"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M13.5 4.5 21 12m0 0-7.5 7.5M21 12H3"
								/>
							</svg>
						</span>
					</Button>
				</div>
			</form>

			<!-- Garis Pemisah (Divider "atau") -->
			<div class="relative my-6">
				<div class="absolute inset-0 flex items-center">
					<div class="w-full border-t border-neutral-200"></div>
				</div>
				<div class="relative flex justify-center text-xs">
					<span class="bg-white px-3 font-normal text-neutral-400">atau</span>
				</div>
			</div>

			<!-- Tombol Alternatif: Masuk dengan Google -->
			<Button type="button" variant="outline" size="lg" fullWidth onclick={handleGoogleLogin}>
				<span class="inline-flex items-center justify-center gap-3 text-neutral-700">
					<!-- Official Google "G" Icon SVG -->
					<svg class="h-5 w-5" viewBox="0 0 24 24">
						<path
							fill="#4285F4"
							d="M23.745 12.27c0-.7-.06-1.4-.19-2.07H12v4.51h6.6c-.29 1.52-1.14 2.82-2.4 3.68v3.05h3.88c2.27-2.09 3.665-5.17 3.665-9.17Z"
						/>
						<path
							fill="#34A853"
							d="M12 24c3.24 0 5.95-1.08 7.93-2.91l-3.88-3.05c-1.08.72-2.45 1.16-4.05 1.16-3.12 0-5.77-2.1-6.72-4.93H1.25v3.15C3.27 21.36 7.34 24 12 24Z"
						/>
						<path
							fill="#FBBC05"
							d="M5.28 14.27c-.25-.72-.38-1.49-.38-2.27s.13-1.55.38-2.27V6.58H1.25C.45 8.18 0 9.99 0 12s.45 3.82 1.25 5.42l4.03-3.15Z"
						/>
						<path
							fill="#EA4335"
							d="M12 4.75c1.77 0 3.35.61 4.6 1.8l3.42-3.42C17.95 1.19 15.24 0 12 0 7.34 0 3.27 2.64 1.25 6.58l4.03 3.15c.95-2.83 3.6-4.98 6.72-4.98Z"
						/>
					</svg>
					Masuk dengan Google
				</span>
			</Button>
		</div>

		<!-- Footer Copyright -->
		<footer class="mt-8 text-center text-xs text-neutral-400">
			v1.0.0 &bull; Gen-E &bull; GEN ENTERPRISE &copy; {new Date().getFullYear()}
		</footer>
	</main>
</div>
