package ui

import (
	"bytes"
	"context"
	"html/template"
	"io"

	"github.com/a-h/templ"
	"github.com/alexanderzull/file-converter/internal/auth"
	"github.com/alexanderzull/file-converter/internal/quota"
)

var templates = template.Must(template.New("base").Funcs(templateFuncs()).Parse(
	baseTemplate +
		landingTemplate +
		loginTemplate +
		registerTemplate +
		forgotTemplate +
		resetTemplate +
		callbackTemplate +
		dashboardTemplate +
		jobsTemplate,
))

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

func Dashboard(appName string, currentUser *auth.User, csrfToken string, csrfField template.HTML, flash *Flash, jobs []Job, maxUploadMB int64, usage quota.Summary) templ.Component {
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
		Quota:       usage,
	})
}

func JobsPanel(jobs []Job, usage quota.Summary) templ.Component {
	return renderFragment("jobs_panel", struct {
		Jobs  []Job
		Quota quota.Summary
	}{
		Jobs:  jobs,
		Quota: usage,
	})
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
