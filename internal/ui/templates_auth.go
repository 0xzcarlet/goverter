package ui

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
