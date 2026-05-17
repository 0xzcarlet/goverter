package ui

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
				<a href="/#landing-details">Konversi</a>
				<a href="/#format-support">Format</a>
				<a href="/#guest-member">Guest vs Login</a>
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
