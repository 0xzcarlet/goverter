package ui

const dashboardTemplate = `
{{define "dashboard"}}
<section class="dashboard-grid">
	<div class="panel convert-panel">
		<div class="panel-header">
			<div>
				<p class="eyebrow">Convert workspace</p>
				<h1>Converter dashboard</h1>
			</div>
			<div class="quota-chip{{if .Quota.Exhausted}} quota-chip-danger{{end}}">
				<span class="quota-chip-label">Sisa hari ini</span>
				<strong>{{.Quota.Remaining}} / {{.Quota.Limit}}</strong>
			</div>
		</div>
		<p class="lede compact-lede">Upload file PDF atau EPUB, pilih output yang dibutuhkan, lalu pantau proses dan unduh hasilnya dari panel riwayat.</p>
		{{if .Flash}}<p class="{{flashClass .Flash.Kind}}">{{.Flash.Message}}</p>{{end}}
		<div class="quota-summary{{if .Quota.Exhausted}} quota-summary-danger{{end}}">
			<div>
				<p class="quota-label">Quota harian</p>
				<h2>{{.Quota.CompletedCount}} selesai, {{.Quota.ReservedCount}} sedang dipakai</h2>
			</div>
			<p class="muted-copy">Limit diambil dari environment server. Slot di-reserve saat submit dan dikembalikan otomatis jika conversion gagal.</p>
		</div>
		<form method="post" action="/app/conversions" enctype="multipart/form-data" class="stack-form convert-form">
			{{.CSRFField}}
			<div class="form-grid">
				<label>Source file
					<input type="file" name="file" accept=".pdf,.epub,application/pdf,application/epub+zip" required>
				</label>
				<label>Convert to
					<select name="target_format" required>
						<option value="epub">EPUB</option>
						<option value="pdf">PDF</option>
					</select>
				</label>
			</div>
			<div class="convert-form-footer">
				<p class="muted-copy">Upload cap: {{.MaxUploadMB}} MB.</p>
				<button type="submit" {{if .Quota.Exhausted}}disabled{{end}}>Queue conversion</button>
			</div>
		</form>
	</div>
	<div class="panel queue-panel">
		<div class="panel-header">
			<div>
				<p class="eyebrow">Live queue</p>
				<h2>Riwayat dan hasil convert</h2>
			</div>
			<p class="muted-copy panel-meta">Auto refresh setiap 5 detik.</p>
		</div>
		<div id="jobs-panel" hx-get="/app/jobs" hx-trigger="load, every 5s" hx-swap="outerHTML">
			{{template "jobs_panel" .}}
		</div>
	</div>
</section>
{{end}}
`

const jobsTemplate = `
{{define "jobs_panel"}}
<div id="jobs-panel" class="jobs-panel-shell">
	<div class="jobs-quota-row">
		<div class="jobs-quota-card">
			<span class="jobs-quota-label">Active slot</span>
			<strong>{{.Quota.ActiveCount}} / {{.Quota.Limit}}</strong>
		</div>
		<div class="jobs-quota-card">
			<span class="jobs-quota-label">Remaining</span>
			<strong>{{.Quota.Remaining}}</strong>
		</div>
		<div class="jobs-quota-card">
			<span class="jobs-quota-label">Completed</span>
			<strong>{{.Quota.CompletedCount}}</strong>
		</div>
	</div>
	{{if .Jobs}}
	<table class="jobs-table">
		<thead>
			<tr>
				<th>Source</th>
				<th>Target</th>
				<th>Status</th>
				<th>Created</th>
				<th>Finished</th>
				<th>Action</th>
			</tr>
		</thead>
		<tbody>
			{{range .Jobs}}
			<tr class="job-row">
				<td>
					<div class="job-source">
						<strong>{{fileStem .SourceFileName}}</strong>
						<span>{{.SourceFileName}}</span>
					</div>
				</td>
				<td><span class="job-target">{{.TargetFormat}}</span></td>
				<td><span class="{{statusClass .Status}}">{{statusText .Status}}</span></td>
				<td>{{formatCreated .CreatedAt}}</td>
				<td>{{formatTime .FinishedAt}}</td>
				<td>
					{{if and (eq .Status "done") .OutputStorageKey}}
					<a href="{{downloadPath .ID}}" class="button-link jobs-download">Download</a>
					{{else if eq .Status "failed"}}
					<span class="jobs-action-muted">Check error</span>
					{{else}}
					<span class="jobs-action-muted">Processing</span>
					{{end}}
				</td>
			</tr>
			{{if .ErrorMessage}}
			<tr>
				<td colspan="6" class="job-error">{{.ErrorMessage}}</td>
			</tr>
			{{end}}
			{{end}}
		</tbody>
	</table>
	{{else}}
	<div class="empty-state">
		<h3>Belum ada conversion</h3>
		<p class="muted-copy">Job yang Anda queue akan muncul di sini lengkap dengan status dan tombol download saat hasil sudah siap.</p>
	</div>
	{{end}}
</div>
{{end}}
`
