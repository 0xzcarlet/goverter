package ui

import (
	"bytes"
	"context"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/alexanderzull/file-converter/internal/auth"
	"github.com/alexanderzull/file-converter/internal/db"
)

type Flash struct {
	Kind    string
	Message string
}

type layoutData struct {
	Title       string
	AppName     string
	CurrentUser *auth.User
	CSRFToken   string
	Body        template.HTML
}

type authView struct {
	Flash     *Flash
	CSRFField template.HTML
	Email     string
}

type resetView struct {
	Flash      *Flash
	CSRFField  template.HTML
	HasSession bool
}

type dashboardView struct {
	Flash       *Flash
	CSRFField   template.HTML
	CurrentUser *auth.User
	Jobs        []db.ConversionJob
	MaxUploadMB int64
}

type callbackView struct {
	Heading    string
	Message    string
	RedirectTo string
	PostPath   string
	CSRFToken  string
}

var templates = template.Must(template.New("base").Funcs(template.FuncMap{
	"formatTime": func(v *time.Time) string {
		if v == nil {
			return "—"
		}
		return v.Local().Format("02 Jan 2006 15:04")
	},
	"formatCreated": func(v time.Time) string {
		return v.Local().Format("02 Jan 2006 15:04")
	},
	"statusClass": func(status string) string {
		switch strings.ToLower(status) {
		case "done":
			return "status status-done"
		case "failed":
			return "status status-failed"
		case "processing":
			return "status status-processing"
		default:
			return "status status-queued"
		}
	},
	"flashClass": func(kind string) string {
		switch strings.ToLower(kind) {
		case "error":
			return "flash flash-error"
		case "success":
			return "flash flash-success"
		default:
			return "flash"
		}
	},
}).Parse(baseTemplate + landingTemplate + loginTemplate + registerTemplate + forgotTemplate + resetTemplate + callbackTemplate + dashboardTemplate + jobsTemplate))

func Landing(appName string, currentUser *auth.User) templ.Component {
	return renderPage("landing", layoutData{
		Title:       appName,
		AppName:     appName,
		CurrentUser: currentUser,
	})
}

func Login(appName, csrfToken string, csrfField template.HTML, flash *Flash, email string) templ.Component {
	return renderPage("login", layoutData{
		Title:     "Login",
		AppName:   appName,
		CSRFToken: csrfToken,
	}, authView{Flash: flash, CSRFField: csrfField, Email: email})
}

func Register(appName, csrfToken string, csrfField template.HTML, flash *Flash, email string) templ.Component {
	return renderPage("register", layoutData{
		Title:     "Register",
		AppName:   appName,
		CSRFToken: csrfToken,
	}, authView{Flash: flash, CSRFField: csrfField, Email: email})
}

func ForgotPassword(appName, csrfToken string, csrfField template.HTML, flash *Flash, email string) templ.Component {
	return renderPage("forgot", layoutData{
		Title:     "Forgot Password",
		AppName:   appName,
		CSRFToken: csrfToken,
	}, authView{Flash: flash, CSRFField: csrfField, Email: email})
}

func ResetPassword(appName string, currentUser *auth.User, csrfToken string, csrfField template.HTML, flash *Flash, hasSession bool) templ.Component {
	return renderPage("reset", layoutData{
		Title:       "Reset Password",
		AppName:     appName,
		CurrentUser: currentUser,
		CSRFToken:   csrfToken,
	}, resetView{Flash: flash, CSRFField: csrfField, HasSession: hasSession})
}

func Dashboard(appName string, currentUser *auth.User, csrfToken string, csrfField template.HTML, flash *Flash, jobs []db.ConversionJob, maxUploadMB int64) templ.Component {
	return renderPage("dashboard", layoutData{
		Title:       "Dashboard",
		AppName:     appName,
		CurrentUser: currentUser,
		CSRFToken:   csrfToken,
	}, dashboardView{
		Flash:       flash,
		CSRFField:   csrfField,
		CurrentUser: currentUser,
		Jobs:        jobs,
		MaxUploadMB: maxUploadMB,
	})
}

func JobsPanel(jobs []db.ConversionJob) templ.Component {
	return renderFragment("jobs_panel", struct {
		Jobs []db.ConversionJob
	}{Jobs: jobs})
}

func Callback(appName, csrfToken, heading, message, redirectTo string) templ.Component {
	return renderPage("callback", layoutData{
		Title:     heading,
		AppName:   appName,
		CSRFToken: csrfToken,
	}, callbackView{
		Heading:    heading,
		Message:    message,
		RedirectTo: redirectTo,
		PostPath:   "/auth/callback/session",
		CSRFToken:  csrfToken,
	})
}

func renderPage(bodyName string, layout layoutData, bodyData ...any) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		body := ""
		if len(bodyData) > 0 {
			var buf bytes.Buffer
			if err := templates.ExecuteTemplate(&buf, bodyName, bodyData[0]); err != nil {
				return err
			}
			body = buf.String()
		} else {
			var buf bytes.Buffer
			if err := templates.ExecuteTemplate(&buf, bodyName, nil); err != nil {
				return err
			}
			body = buf.String()
		}
		layout.Body = template.HTML(body)
		return templates.ExecuteTemplate(w, "base", layout)
	})
}

func renderFragment(name string, data any) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return templates.ExecuteTemplate(w, name, data)
	})
}

const baseTemplate = `
{{define "base"}}
<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<meta name="csrf-token" content="{{.CSRFToken}}">
	<title>{{.Title}} · {{.AppName}}</title>
	<link rel="stylesheet" href="/assets/css/app.css">
	<script src="/assets/js/htmx.min.js" defer></script>
</head>
<body>
	<div class="shell">
		<header class="topbar">
			<a href="/" class="brand">{{.AppName}}</a>
			<nav class="nav">
				{{if .CurrentUser}}
				<a href="/dashboard">Dashboard</a>
				<form method="post" action="/logout" class="inline-form">
					<button class="ghost-button" type="submit">Logout</button>
				</form>
				{{else}}
				<a href="/login">Login</a>
				<a href="/register" class="button-link">Create Account</a>
				{{end}}
			</nav>
		</header>
		<main class="main">
			{{.Body}}
		</main>
	</div>
</body>
</html>
{{end}}
`

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

const loginTemplate = `
{{define "login"}}
<section class="auth-wrap panel">
	<h1>Login</h1>
	{{if .Flash}}<p class="{{flashClass .Flash.Kind}}">{{.Flash.Message}}</p>{{end}}
	<form method="post" action="/login" class="stack-form">
		{{.CSRFField}}
		<label>Email
			<input type="email" name="email" value="{{.Email}}" autocomplete="email" required>
		</label>
		<label>Password
			<input type="password" name="password" autocomplete="current-password" required>
		</label>
		<button type="submit">Login</button>
	</form>
	<div class="muted-links">
		<a href="/auth/google">Continue with Google</a>
		<a href="/forgot-password">Forgot password?</a>
	</div>
</section>
{{end}}
`

const registerTemplate = `
{{define "register"}}
<section class="auth-wrap panel">
	<h1>Create account</h1>
	{{if .Flash}}<p class="{{flashClass .Flash.Kind}}">{{.Flash.Message}}</p>{{end}}
	<form method="post" action="/register" class="stack-form">
		{{.CSRFField}}
		<label>Email
			<input type="email" name="email" value="{{.Email}}" autocomplete="email" required>
		</label>
		<label>Password
			<input type="password" name="password" minlength="8" autocomplete="new-password" required>
		</label>
		<button type="submit">Register</button>
	</form>
	<p class="muted-copy">If Supabase email confirmation is enabled, the app will ask the user to verify the address before the first session is created.</p>
</section>
{{end}}
`

const forgotTemplate = `
{{define "forgot"}}
<section class="auth-wrap panel">
	<h1>Forgot password</h1>
	{{if .Flash}}<p class="{{flashClass .Flash.Kind}}">{{.Flash.Message}}</p>{{end}}
	<form method="post" action="/forgot-password" class="stack-form">
		{{.CSRFField}}
		<label>Email
			<input type="email" name="email" value="{{.Email}}" autocomplete="email" required>
		</label>
		<button type="submit">Send reset link</button>
	</form>
</section>
{{end}}
`

const resetTemplate = `
{{define "reset"}}
<section class="auth-wrap panel">
	<h1>Reset password</h1>
	{{if .Flash}}<p class="{{flashClass .Flash.Kind}}">{{.Flash.Message}}</p>{{end}}
	{{if .HasSession}}
	<form method="post" action="/auth/reset" class="stack-form">
		{{.CSRFField}}
		<label>New password
			<input type="password" name="password" minlength="8" autocomplete="new-password" required>
		</label>
		<button type="submit">Update password</button>
	</form>
	{{else}}
	<p class="muted-copy">Open this page from the recovery email so the app can establish a recovery session first.</p>
	{{end}}
	<script>
	(function () {
		if (!window.location.hash || window.location.hash.length < 2) return;
		const params = new URLSearchParams(window.location.hash.slice(1));
		const accessToken = params.get('access_token');
		const refreshToken = params.get('refresh_token');
		if (!accessToken || !refreshToken) return;
		fetch('/auth/callback/session', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				'X-CSRF-Token': document.querySelector('meta[name="csrf-token"]').content
			},
			body: JSON.stringify({
				access_token: accessToken,
				refresh_token: refreshToken,
				expires_in: Number(params.get('expires_in') || 3600)
			})
		}).then(function () {
			window.location.replace('/auth/reset?ready=1');
		});
	})();
	</script>
</section>
{{end}}
`

const callbackTemplate = `
{{define "callback"}}
<section class="panel callback-panel">
	<h1>{{.Heading}}</h1>
	<p>{{.Message}}</p>
	<p id="callback-status" class="muted-copy">Waiting for Supabase to hand the session back to the app.</p>
</section>
<script>
	(function () {
		const params = new URLSearchParams(window.location.hash.slice(1));
		const status = document.getElementById('callback-status');
		const accessToken = params.get('access_token');
		const refreshToken = params.get('refresh_token');
		const errorDescription = params.get('error_description');
		if (errorDescription) {
			status.textContent = errorDescription;
			return;
		}
		if (!accessToken || !refreshToken) {
			status.textContent = 'No auth tokens returned. Check Supabase redirect configuration.';
			return;
		}
		status.textContent = 'Establishing secure server session...';
		fetch('{{.PostPath}}', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				'X-CSRF-Token': '{{.CSRFToken}}'
			},
			body: JSON.stringify({
				access_token: accessToken,
				refresh_token: refreshToken,
				expires_in: Number(params.get('expires_in') || 3600)
			})
		}).then(async function (response) {
			if (!response.ok) {
				const data = await response.json().catch(function () { return null; });
				throw new Error(data && data.error ? data.error : 'Session handoff failed');
			}
			window.location.replace('{{.RedirectTo}}');
		}).catch(function (err) {
			status.textContent = err.message;
		});
	})();
</script>
{{end}}
`

const dashboardTemplate = `
{{define "dashboard"}}
<section class="dashboard-grid">
	<div class="panel">
		<p class="eyebrow">Upload and queue</p>
		<h1>Converter dashboard</h1>
		{{if .Flash}}<p class="{{flashClass .Flash.Kind}}">{{.Flash.Message}}</p>{{end}}
		<form method="post" action="/app/conversions" enctype="multipart/form-data" class="stack-form">
			{{.CSRFField}}
			<label>Source file
				<input type="file" name="file" accept=".pdf,.epub,application/pdf,application/epub+zip" required>
			</label>
			<label>Convert to
				<select name="target_format" required>
					<option value="epub">EPUB</option>
					<option value="pdf">PDF</option>
				</select>
			</label>
			<p class="muted-copy">Current upload cap: {{.MaxUploadMB}} MB. Runtime worker concurrency stays at 1 for small VPS stability.</p>
			<button type="submit">Queue conversion</button>
		</form>
	</div>
	<div class="panel">
		<p class="eyebrow">Live queue</p>
		<div id="jobs-panel" hx-get="/app/jobs" hx-trigger="load, every 5s" hx-swap="outerHTML">
			{{template "jobs_panel" .}}
		</div>
	</div>
</section>
{{end}}
`

const jobsTemplate = `
{{define "jobs_panel"}}
<div id="jobs-panel">
	{{if .Jobs}}
	<table class="jobs-table">
		<thead>
			<tr>
				<th>Source</th>
				<th>Target</th>
				<th>Status</th>
				<th>Created</th>
				<th>Finished</th>
			</tr>
		</thead>
		<tbody>
			{{range .Jobs}}
			<tr>
				<td>{{.SourceFileName}}</td>
				<td>{{.TargetFormat}}</td>
				<td><span class="{{statusClass .Status}}">{{.Status}}</span></td>
				<td>{{formatCreated .CreatedAt}}</td>
				<td>{{formatTime .FinishedAt}}</td>
			</tr>
			{{if .ErrorMessage}}
			<tr>
				<td colspan="5" class="job-error">{{.ErrorMessage}}</td>
			</tr>
			{{end}}
			{{end}}
		</tbody>
	</table>
	{{else}}
	<p class="muted-copy">No conversion jobs yet.</p>
	{{end}}
</div>
{{end}}
`
