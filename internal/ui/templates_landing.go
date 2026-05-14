package ui

const landingTemplate = `
{{define "landing"}}
<div class="landing-page">
	<section class="hero landing-section">
		<div class="hero-copy">
			<p class="eyebrow">File Converter untuk PDF dan EPUB</p>
			<h1>Ubah file PDF dan EPUB dengan alur yang simpel, rapi, dan mudah dipantau.</h1>
			<p class="lede">Upload file dari browser, pilih format tujuan, lalu pantau status konversi di Dashboard tanpa perlu alur yang rumit.</p>
			<div class="actions">
				<a href="/register" class="button-link">Create Account</a>
				<a href="/login" class="text-link hero-secondary-link">Login</a>
			</div>
		</div>
		<div class="hero-card panel">
			<p class="eyebrow">Info Singkat</p>
			<h2>Siap dipakai untuk kebutuhan konversi harian.</h2>
			<ul class="hero-points">
				<li>Support konversi <strong>PDF to EPUB</strong> dan <strong>EPUB to PDF</strong>.</li>
				<li>Upload file langsung dari Dashboard dengan langkah yang ringkas.</li>
				<li>Status proses bisa dipantau dari antrean konversi.</li>
				<li>Cocok untuk kebutuhan personal maupun tim kecil.</li>
			</ul>
		</div>
	</section>

	<section class="landing-section">
		<div class="section-heading">
			<p class="eyebrow">Cara Kerja</p>
			<h2>Tiga langkah untuk mulai konversi.</h2>
			<p class="lede">Flow dibuat singkat agar user baru bisa langsung paham sejak kunjungan pertama.</p>
		</div>
		<div class="steps-grid">
			<article class="panel step-card">
				<span class="step-number">01</span>
				<h3>Upload file</h3>
				<p>Pilih file PDF atau EPUB yang ingin diproses langsung dari Dashboard.</p>
			</article>
			<article class="panel step-card">
				<span class="step-number">02</span>
				<h3>Pilih format tujuan</h3>
				<p>Tentukan hasil akhir yang dibutuhkan, apakah EPUB atau PDF.</p>
			</article>
			<article class="panel step-card">
				<span class="step-number">03</span>
				<h3>Tunggu dan cek status</h3>
				<p>Monitor antrean konversi sampai proses selesai dan hasilnya siap dipakai.</p>
			</article>
		</div>
	</section>

	<section class="landing-section">
		<div class="section-heading">
			<p class="eyebrow">Kenapa Pakai Ini</p>
			<h2>Didesain untuk kebutuhan yang fokus dan langsung ke tujuan.</h2>
		</div>
		<div class="benefits-grid">
			<article class="panel benefit-card">
				<h3>Alur sederhana</h3>
				<p>User tidak perlu mempelajari banyak menu untuk mulai memakai fitur utama.</p>
			</article>
			<article class="panel benefit-card">
				<h3>Fokus pada PDF dan EPUB</h3>
				<p>Halaman ini menjelaskan dengan jelas jenis file yang memang didukung aplikasi.</p>
			</article>
			<article class="panel benefit-card">
				<h3>Status lebih transparan</h3>
				<p>Progress konversi bisa dilihat dari antrean sehingga user tidak menebak-nebak prosesnya.</p>
			</article>
			<article class="panel benefit-card">
				<h3>UI tidak bertele-tele</h3>
				<p>Tampilan dibuat ringkas agar cepat dipahami oleh user yang baru pertama datang.</p>
			</article>
		</div>
	</section>

	<section class="landing-section">
		<div class="panel landing-cta">
			<p class="eyebrow">Mulai Sekarang</p>
			<h2>Siapkan akun Anda dan mulai konversi file dalam beberapa langkah.</h2>
			<p class="lede">Buat akun untuk mulai upload file, atau masuk jika Anda sudah punya akses.</p>
			<div class="actions">
				<a href="/register" class="button-link">Create Account</a>
				<a href="/login" class="text-link hero-secondary-link">Login</a>
			</div>
		</div>
	</section>
</div>
{{end}}
`
