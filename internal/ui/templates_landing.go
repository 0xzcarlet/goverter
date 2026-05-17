package ui

const landingTemplate = `
{{define "landing"}}
<div class="landing-page">
	<section class="landing-hero">
		<div class="hero-copy editorial-copy">
			<h1>Convert PDF dan EPUB dalam alur yang lebih bersih, cepat, dan gampang dipahami.</h1>
			<p class="lede">Goverter membantu visitor baru langsung mulai tanpa onboarding yang ribet. Guest bisa mencoba 1 file per hari dari hero ini, lalu lanjut login kalau butuh quota lebih, riwayat, dan dashboard monitoring.</p>
			<div class="hero-inline-points">
				<div class="hero-inline-point">
					<span class="feature-icon" aria-hidden="true">
						<svg viewBox="0 0 24 24" fill="none"><path d="M12 4v10m0 0-4-4m4 4 4-4M5 18h14" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>
					</span>
					<div>
						<strong>Langsung convert dari landing</strong>
						<span>Hero dibuat sebagai titik aksi utama, bukan sekadar promosi.</span>
					</div>
				</div>
				<div class="hero-inline-point">
					<span class="feature-icon" aria-hidden="true">
						<svg viewBox="0 0 24 24" fill="none"><path d="M4 12h16M12 4l8 8-8 8" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>
					</span>
					<div>
						<strong>Flow tetap fokus</strong>
						<span>Format yang didukung jelas: PDF ke EPUB dan EPUB ke PDF.</span>
					</div>
				</div>
			</div>
			<div class="actions hero-actions">
				{{if eq .HeroMode "guest"}}
				<a href="/register" class="button-link">Create Account</a>
				<a href="/login" class="button-link button-link-secondary">Login</a>
				{{else}}
				<a href="/dashboard" class="button-link">Buka Dashboard</a>
				<a href="/login" class="button-link button-link-secondary">Pindah akun</a>
				{{end}}
			</div>
		</div>
		<div class="hero-card hero-convert-card panel">
			{{if eq .HeroMode "guest"}}
			<div class="hero-card-topline">
				<div>
					<h2>Coba 1 file per hari tanpa login.</h2>
					<p class="muted-copy">Guest bisa convert 1 file per hari. Login memberi akses ke quota penuh, dashboard, dan riwayat hasil convert.</p>
				</div>
				<div class="quota-chip hero-quota-chip{{if .GuestQuota.Exhausted}} quota-chip-danger{{end}}">
					<span class="quota-chip-label">Sisa guest hari ini</span>
					<strong>{{.GuestQuota.Remaining}} / {{.GuestQuota.Limit}}</strong>
				</div>
			</div>
			{{if .Flash}}<p class="{{flashClass .Flash.Kind}}">{{.Flash.Message}}</p>{{end}}
			<form method="post" action="/guest/conversions" enctype="multipart/form-data" class="stack-form hero-convert-form">
				{{.CSRFField}}
				<label>Upload file
					<input type="file" name="file" accept=".pdf,.epub,application/pdf,application/epub+zip" required>
				</label>
				<label>Convert ke
					<select name="target_format" required>
						<option value="epub">EPUB</option>
						<option value="pdf">PDF</option>
					</select>
				</label>
				<div class="hero-form-meta">
					<div class="hero-meta-row">
						<span class="feature-icon feature-icon-soft" aria-hidden="true">
							<svg viewBox="0 0 24 24" fill="none"><path d="M7 4h7l5 5v11H7z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"/><path d="M14 4v5h5" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"/></svg>
						</span>
						<span>Support PDF ke EPUB dan EPUB ke PDF.</span>
					</div>
					<div class="hero-meta-row">
						<span class="feature-icon feature-icon-soft" aria-hidden="true">
							<svg viewBox="0 0 24 24" fill="none"><path d="M12 7v5l3 3m6-3a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>
						</span>
						<span>Ukuran upload maksimum {{.MaxUploadMB}} MB.</span>
					</div>
				</div>
				<button type="submit" class="hero-submit" {{if .GuestQuota.Exhausted}}disabled{{end}}>Convert Sekarang</button>
			</form>
			<div class="hero-login-prompt">
				<p class="muted-copy">Butuh quota lebih, history, dan tracking status?</p>
				<a href="/register" class="button-link button-link-secondary button-link-full">Buat akun untuk akses penuh</a>
			</div>
			{{else}}
			<div class="hero-card-topline">
				<div>
					<h2>Lanjutkan convert dari dashboard Anda.</h2>
					<p class="muted-copy">Saat sudah login, flow terbaik tetap lewat dashboard karena di sana Anda bisa upload, pantau queue, dan download hasil dalam satu tempat.</p>
				</div>
				<div class="member-badge">
					<span class="feature-icon" aria-hidden="true">
						<svg viewBox="0 0 24 24" fill="none"><path d="M12 12a4 4 0 1 0-4-4 4 4 0 0 0 4 4Zm-7 8a7 7 0 0 1 14 0" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>
					</span>
					<span>Member mode</span>
				</div>
			</div>
			<div class="member-hero-grid">
				<div class="member-hero-card">
					<h3>Dashboard terpusat</h3>
					<p>Status queue, quota harian, dan download hasil ada di satu halaman.</p>
				</div>
				<div class="member-hero-card">
					<h3>Riwayat lebih rapi</h3>
					<p>Job yang sedang jalan dan yang sudah selesai bisa dicek ulang tanpa kehilangan konteks.</p>
				</div>
			</div>
			<div class="actions hero-actions">
				<a href="/dashboard" class="button-link">Masuk ke Dashboard</a>
				<a href="#landing-details" class="button-link button-link-secondary">Lihat detail flow</a>
			</div>
			{{end}}
		</div>
	</section>

	<section class="landing-section" id="landing-details">
		<div class="section-heading">
			<h2>Tiga langkah yang sengaja dibuat ringkas.</h2>
			<p class="lede">Landing ini mengantar user baru ke aksi utama dengan cepat, lalu detail di bawahnya menjelaskan kenapa flow-nya tetap nyaman dipakai untuk kebutuhan harian.</p>
		</div>
		<div class="steps-grid editorial-steps">
			<article class="panel step-card">
				<span class="step-number">01</span>
				<h3>Upload sekali, tanpa muter.</h3>
				<p>Guest bisa mulai dari hero, sedangkan member lanjut dengan quota dan history yang lebih lengkap di dashboard.</p>
			</article>
			<article class="panel step-card">
				<span class="step-number">02</span>
				<h3>Pilih arah convert yang jelas.</h3>
				<p>Tidak ada format yang membingungkan. User hanya memilih PDF ke EPUB atau EPUB ke PDF sesuai kebutuhan.</p>
			</article>
			<article class="panel step-card">
				<span class="step-number">03</span>
				<h3>Lanjutkan dengan flow yang pas.</h3>
				<p>Guest menerima hasil langsung, sementara member mendapatkan pengalaman dashboard untuk kerja yang lebih rutin.</p>
			</article>
		</div>
	</section>

	<section class="landing-section" id="format-support">
		<div class="section-heading">
			<h2>Detail page di bawah hero menjelaskan value yang paling penting.</h2>
		</div>
		<div class="benefit-showcase">
			<article class="panel benefit-card benefit-card-large">
				<div class="benefit-card-heading">
					<span class="feature-icon" aria-hidden="true">
						<svg viewBox="0 0 24 24" fill="none"><path d="M4 7h16M7 12h10M9 17h6" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/></svg>
					</span>
					<h3>UX yang tetap bersih walau fungsional.</h3>
				</div>
				<p>Bagian atas fokus pada satu tugas utama. Bagian bawah baru menjelaskan benefit, support format, dan alasan kenapa login memberi pengalaman yang lebih lengkap.</p>
			</article>
			<article class="panel benefit-card">
				<div class="benefit-card-heading">
					<span class="feature-icon" aria-hidden="true">
						<svg viewBox="0 0 24 24" fill="none"><path d="M8 5H6a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2M8 5a2 2 0 0 0 2 2h4a2 2 0 0 0 2-2M8 5a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2m-4 7h6m0 0-3-3m3 3-3 3" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>
					</span>
					<h3>Support format yang tegas</h3>
				</div>
				<p>Produk tidak menjanjikan terlalu banyak hal. Justru fokus pada dua arah convert yang memang tersedia hari ini.</p>
			</article>
			<article class="panel benefit-card">
				<div class="benefit-card-heading">
					<span class="feature-icon" aria-hidden="true">
						<svg viewBox="0 0 24 24" fill="none"><path d="M12 6v6l4 2m5-2a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>
					</span>
					<h3>Quota dan ekspektasi jelas</h3>
				</div>
				<p>Guest limit dipaparkan langsung di hero agar user tahu apa yang bisa dicoba sekarang dan kapan perlu login.</p>
			</article>
		</div>
	</section>

	<section class="landing-section" id="guest-member">
		<div class="panel landing-cta landing-compare">
			<div>
				<h2>Pakai sebagai guest untuk mencoba. Login saat workflow Anda mulai rutin.</h2>
				<p class="lede">Struktur halaman ini dibuat agar perbedaan value guest versus member terbaca sejak awal tanpa memaksa user baru masuk dashboard terlalu cepat.</p>
			</div>
			<div class="compare-grid">
				<div class="compare-card">
					<span class="compare-label">Guest</span>
					<strong>1 file per hari</strong>
					<p>Langsung convert dari hero dan dapatkan hasilnya saat itu juga.</p>
				</div>
				<div class="compare-card compare-card-strong">
					<span class="compare-label">Logged-in</span>
					<strong>Quota penuh + dashboard</strong>
					<p>Upload dari workspace, cek status, lihat riwayat, dan download hasil convert dengan alur yang lebih lengkap.</p>
				</div>
			</div>
			<div class="actions">
				<a href="/register" class="button-link">Create Account</a>
				<a href="/login" class="button-link button-link-secondary">Login</a>
			</div>
		</div>
	</section>
</div>
{{end}}
`
