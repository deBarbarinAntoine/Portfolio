package main

import (
	"Portfolio/internal/data"
	"Portfolio/internal/validator"
	"database/sql"
	"errors"
	"fmt"
	"github.com/alexedwards/flow"
	"html/template"
	"net/http"
	"strings"
)

/* #############################################################################
/*	COMMON
/* #############################################################################*/

func (app *application) notFound(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Not Found"

	// render the template
	app.render(w, r, http.StatusOK, "error.tmpl", tmplData)
}

func (app *application) methodNotAllowed(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Oooops"

	// setting the error title and message
	tmplData.Error.Title = "Error 405"
	tmplData.Error.Message = "Something went wrong!"

	// render the template
	app.render(w, r, http.StatusOK, "error.tmpl", tmplData)
}

func (app *application) index(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Home"

	// render the template
	app.render(w, r, http.StatusOK, "home.tmpl", tmplData)
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - About"

	// render the template
	app.render(w, r, http.StatusOK, "policies.tmpl", tmplData)
}

func (app *application) search(w http.ResponseWriter, r *http.Request) {

	// checking the query
	if r.URL.Query() == nil {
		app.clientError(r, w, http.StatusBadRequest)
		return
	}

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Search"

	// retrieving the research text
	tmplData.Search = r.URL.Query().Get("q")
	if tmplData.Search == "" {
		tmplData.Search = "*"
	}

	// TODO -> search in the posts

	// render the template
	app.render(w, r, http.StatusOK, "search.tmpl", tmplData)
}

/* #############################################################################
/*	USER ACCESS
/* #############################################################################*/

func (app *application) register(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Register"

	// render the template
	app.render(w, r, http.StatusOK, "home.tmpl", tmplData)
}

func (app *application) registerPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newUserRegisterForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.logger.Error(err.Error())
		app.clientError(r, w, http.StatusBadRequest)
		return
	}

	// DEBUG
	app.logger.Debug(fmt.Sprintf("form: %+v", form))

	// checking the data from the user
	form.StringCheck(form.Username, 2, 70, true, "username")
	form.ValidateEmail(form.Email)
	form.ValidateRegisterPassword(form.Password, form.ConfirmPassword)

	// return to register page if there is an error
	if !form.Valid() {

		// DEBUG
		app.logger.Debug(fmt.Sprintf("errors: %+v", form.FieldErrors))

		// retrieving basic template data
		tmplData := app.newTemplateData(r)
		tmplData.Form = form

		// render the template
		app.render(w, r, http.StatusUnprocessableEntity, "home.tmpl", tmplData)
		return
	}

	// creating and registering the user
	v := validator.New()
	user := &data.User{
		Name:     form.Username,
		Email:    form.Email,
		Password: form.Password,
	}
	err = app.models.UserModel.Create(app.getToken(r, authTokenSessionManager), user, v)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			app.clientError(r, w, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// looking for errors from the API
	if !v.Valid() {

		// DEBUG
		app.logger.Debug(fmt.Sprintf("errors: %+v", v.NonFieldErrors))

		// retrieving basic template data
		tmplData := app.newTemplateData(r)

		tmplData.Form = form

		tmplData.FieldErrors = v.FieldErrors
		tmplData.NonFieldErrors = v.NonFieldErrors

		// render the template
		app.render(w, r, http.StatusUnprocessableEntity, "home.tmpl", tmplData)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "We've sent you a confirmation email!")

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (app *application) confirm(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Confirm"

	// retrieving the activation token from the URL
	tmplData.ActivationToken = flow.Param(r.Context(), "token")
	if tmplData.ActivationToken == "" {
		app.clientError(r, w, http.StatusBadRequest)
		return
	}

	// render the template
	app.render(w, r, http.StatusOK, "confirm.tmpl", tmplData)
}

func (app *application) confirmPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newUserConfirmForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(r, w, http.StatusBadRequest)
		return
	}

	// checking the data from the user
	form.ValidateToken(form.Token)

	// return to confirm page if there is an error
	if !form.Valid() {

		// retrieving basic template data
		tmplData := app.newTemplateData(r)

		tmplData.FieldErrors = form.FieldErrors
		tmplData.ActivationToken = form.Token

		// render the template
		app.render(w, r, http.StatusOK, "confirm.tmpl", tmplData)
		return
	}

	// API request to activate the user account
	v := validator.New()
	err = app.models.UserModel.Activate(form.Token, v)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			app.clientError(r, w, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// looking for errors from the API
	if !v.Valid() {

		// retrieving basic template data
		tmplData := app.newTemplateData(r)

		tmplData.FieldErrors = form.FieldErrors
		tmplData.ActivationToken = form.Token

		// render the template
		app.render(w, r, http.StatusOK, "confirm.tmpl", tmplData)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Your session has been activated successfully!")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (app *application) login(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Login"

	// render the template
	app.render(w, r, http.StatusOK, "home.tmpl", tmplData)
}

func (app *application) loginPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newUserLoginForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(r, w, http.StatusBadRequest)
		return
	}

	// checking the data from the user
	form.Check(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.Check(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.ValidatePassword(form.Password)

	// return to login page if there is an error
	if !form.Valid() {

		// retrieving basic template data
		tmplData := app.newTemplateData(r)

		tmplData.Form = form

		// render the template
		app.render(w, r, http.StatusUnprocessableEntity, "home.tmpl", tmplData)
		return
	}

	// building the API request body
	body := map[string]string{
		"email":    form.Email,
		"password": form.Password,
	}

	// API request to authenticate the user
	v := validator.New()
	tokens, err := app.models.TokenModel.Authenticate(body, v)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			app.clientError(r, w, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// looking for errors from the API
	if !v.Valid() {

		// retrieving basic template data
		tmplData := app.newTemplateData(r)

		tmplData.FieldErrors = form.FieldErrors

		form.Password = ""
		tmplData.Form = form

		// render the template
		app.render(w, r, http.StatusUnprocessableEntity, "home.tmpl", tmplData)
		return
	}

	// renewing the user session
	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// fetching the user id from the API
	user, err := app.models.UserModel.GetByID(tokens.Authentication.Token, "me", nil, v)
	if err != nil || !v.Valid() {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			app.clientError(r, w, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// storing the user id and tokens in the user session
	app.sessionManager.Put(r.Context(), authenticatedUserIDSessionManager, user.ID)
	app.putToken(r, *tokens)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (app *application) forgotPassword(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Forgot password"

	// render the template
	app.render(w, r, http.StatusOK, "home.tmpl", tmplData)
}

func (app *application) forgotPasswordPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newForgotPasswordForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(r, w, http.StatusBadRequest)
		return
	}

	// checking the data from the user
	form.ValidateEmail(form.Email)

	// return to forgot-password page if there is an error
	if !form.Valid() {

		// retrieving basic template data
		tmplData := app.newTemplateData(r)

		tmplData.NonFieldErrors = form.NonFieldErrors
		tmplData.FieldErrors = form.FieldErrors

		tmplData.Form = form

		// render the template
		app.render(w, r, http.StatusOK, "home.tmpl", tmplData)
		return
	}

	// API request to send a reset password token
	v := validator.New()
	err = app.models.UserModel.ForgotPassword(form.Email, v)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			app.clientError(r, w, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// looking for errors from the API
	if !v.Valid() {

		// retrieving basic template data
		tmplData := app.newTemplateData(r)

		tmplData.NonFieldErrors = form.NonFieldErrors
		tmplData.FieldErrors = form.FieldErrors

		tmplData.Form = form

		// render the template
		app.render(w, r, http.StatusOK, "home.tmpl", tmplData)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "We've sent you a mail to reset your password!")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (app *application) resetPassword(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Reset password"

	// retrieving the reset token from the URL
	tmplData.ResetToken = flow.Param(r.Context(), "token")
	if tmplData.ResetToken == "" {
		app.clientError(r, w, http.StatusBadRequest)
		return
	}

	// render the template
	app.render(w, r, http.StatusOK, "home.tmpl", tmplData)
}

func (app *application) resetPasswordPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newResetPasswordForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(r, w, http.StatusBadRequest)
		return
	}

	// checking the data from the user
	form.ValidateNewPassword(form.NewPassword, form.ConfirmPassword)
	form.ValidateToken(form.Token)

	// return to reset-password page if there is an error
	if !form.Valid() {

		// retrieving basic template data
		tmplData := app.newTemplateData(r)

		tmplData.NonFieldErrors = form.NonFieldErrors
		tmplData.FieldErrors = form.FieldErrors

		// render the template
		app.render(w, r, http.StatusOK, "home.tmpl", tmplData)
		return
	}

	// building the API request body
	body := map[string]string{
		"token":            form.Token,
		"new_password":     form.NewPassword,
		"confirm_password": form.ConfirmPassword,
	}

	// API request to send a reset password token
	v := validator.New()
	err = app.models.UserModel.ResetPassword(body, v)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			app.clientError(r, w, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// looking for errors from the API
	if !v.Valid() {

		// retrieving basic template data
		tmplData := app.newTemplateData(r)

		tmplData.NonFieldErrors = form.NonFieldErrors
		tmplData.FieldErrors = form.FieldErrors

		// render the template
		app.render(w, r, http.StatusOK, "home.tmpl", tmplData)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Your password has been updated successfully!")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

/* #############################################################################
/*	RESTRICTED
/* #############################################################################*/

func (app *application) dashboard(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Dashboard"

	// render the template
	app.render(w, r, http.StatusOK, "dashboard.tmpl", tmplData)
}

func (app *application) logoutPost(w http.ResponseWriter, r *http.Request) {

	// revoking the user's tokens
	v := validator.New()
	err := app.models.TokenModel.Logout(app.getToken(r, authTokenSessionManager), v)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			app.clientError(r, w, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// looking for errors from the API
	if !v.Valid() {
		app.logger.Error(fmt.Sprintf("errors: %s", string(v.Errors())))
		app.clientError(r, w, http.StatusBadRequest)
		return
	}

	// logging the user out
	err = app.logout(r)
	if err != nil {

		// DEBUG
		app.logger.Debug(fmt.Sprintf("error: %s", err.Error()))

		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) updateUser(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Update user"

	// render the template
	app.render(w, r, http.StatusOK, "user-update.tmpl", tmplData)
}

func (app *application) updateUserPut(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newUserUpdateForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(r, w, http.StatusBadRequest)
		return
	}

	// creating the User struct to insert the new data into it
	user := &data.User{}

	// checking the data from the user
	var isEmpty = true
	if form.Username != nil {
		isEmpty = false
		form.StringCheck(*form.Username, 2, 70, false, "username")
		user.Name = *form.Username
	}
	if form.Password != nil || form.NewPassword != nil || form.ConfirmationPassword != nil {
		isEmpty = false
		form.ValidateNewPassword(*form.NewPassword, *form.ConfirmationPassword)
		user.Password = *form.NewPassword
	}
	if form.Email != nil {
		isEmpty = false
		form.ValidateEmail(*form.Email)
		user.Email = *form.Email
	}
	if isEmpty {
		form.AddNonFieldError("at least one field is required")
	}

	// return to update-user page if there is an error
	if !form.Valid() {

		// retrieving basic template data
		tmplData := app.newTemplateData(r)

		tmplData.NonFieldErrors = form.NonFieldErrors
		tmplData.FieldErrors = form.FieldErrors

		// render the template
		app.render(w, r, http.StatusOK, "user-update.tmpl", tmplData)
		return
	}

	// API request to send a reset password token
	v := validator.New()
	err = app.models.UserModel.Update(app.getToken(r, authTokenSessionManager), *form.Password, user, v)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			app.clientError(r, w, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// looking for errors from the API
	if !v.Valid() {

		// retrieving basic template data
		tmplData := app.newTemplateData(r)

		tmplData.NonFieldErrors = form.NonFieldErrors
		tmplData.FieldErrors = form.FieldErrors

		// render the template
		app.render(w, r, http.StatusOK, "user-update.tmpl", tmplData)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Your data has been updated successfully!")
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (app *application) createPost(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Create Post"

	// render the template
	app.render(w, r, http.StatusOK, "post-create.tmpl", tmplData)
}

func (app *application) createPostPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newPostForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(r, w, http.StatusBadRequest)
		return
	}

	// DEBUG
	app.logger.Debug(fmt.Sprintf("content: %+v", *form.Content))
	app.logger.Debug(fmt.Sprintf("thread_id: %+v", *form.ThreadID))

	// creating the new thread
	post := &data.Post{}

	// checking the data from the user
	if form.Content == nil {
		form.AddFieldError("content", "must be provided")
	} else {
		form.StringCheck(*form.Content, 2, 1_020, true, "content")
		content := strings.ReplaceAll(*form.Content, "\n", "<br>")
		post.Content = template.HTML(content)
	}
	if form.ThreadID == nil {
		form.AddFieldError("thread_id", "must be provided")
	} else {
		form.CheckID(*form.ThreadID, "thread_id")
		post.Thread.ID = *form.ThreadID
	}
	if form.ParentPostID != nil {
		form.CheckID(*form.ParentPostID, "parent_post_id")
		post.IDParentPost = *form.ParentPostID
	}

	// return to post-create page if there is an error
	if !form.Valid() {
		tmplData := app.newTemplateData(r) // FIXME
		tmplData.Form = form

		tmplData.NonFieldErrors = form.NonFieldErrors
		tmplData.FieldErrors = form.FieldErrors

		// render the template
		app.render(w, r, http.StatusUnprocessableEntity, "home.tmpl", tmplData)
		return
	}

	// API request to create a category
	v := validator.New()
	err = app.models.PostModel.Create(app.getToken(r, authTokenSessionManager), post, v)
	// looking for errors from the API
	if !v.Valid() {

		// retrieving basic template data
		tmplData := app.newTemplateData(r) // FIXME

		tmplData.NonFieldErrors = form.NonFieldErrors
		tmplData.FieldErrors = form.FieldErrors

		// render the template
		app.render(w, r, http.StatusOK, "home.tmpl", tmplData)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Post created successfully!")
	http.Redirect(w, r, fmt.Sprintf("/thread/%d", post.Thread.ID), http.StatusSeeOther)
}

func (app *application) updatePost(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Update post"

	// retrieving the post id from the path
	id, err := getPathID(r)
	if err != nil {
		app.clientError(r, w, http.StatusBadRequest)
		return
	}

	// retrieving the post from the API
	v := validator.New()
	post, err := app.models.PostModel.GetByID(app.getToken(r, authTokenSessionManager), id, v)
	if err != nil {
		switch {
		case errors.Is(err, api.ErrRecordNotFound):
			app.clientError(r, w, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// inserting the post values in the TemplateData's Form
	tmplData.Form = post

	// render the template
	app.render(w, r, http.StatusOK, "post-update.tmpl", tmplData)
}

func (app *application) updatePostPut(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newPostForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(r, w, http.StatusBadRequest)
		return
	}

	// creating the updated post
	post := &data.Post{}

	// checking the data from the user
	if form.Content != nil {
		form.StringCheck(*form.Content, 1, 1_020, false, "content")
		post.Content = template.HTML(*form.Content)
	}
	if form.ParentPostID != nil {
		form.CheckID(*form.ParentPostID, "parent_post_id")
		post.IDParentPost = *form.ParentPostID
	}

	// return to post-update page if there is an error
	if !form.Valid() {
		tmplData := app.newTemplateData(r)
		tmplData.Form = form

		tmplData.NonFieldErrors = form.NonFieldErrors
		tmplData.FieldErrors = form.FieldErrors

		// render the template
		app.render(w, r, http.StatusUnprocessableEntity, "post-update.tmpl", tmplData)
		return
	}

	// retrieving the post id from the path
	post.ID, err = getPathID(r)
	if err != nil {
		app.clientError(r, w, http.StatusBadRequest)
		return
	}

	// API request to update a post
	v := validator.New()
	err = app.models.PostModel.Update(app.getToken(r, authTokenSessionManager), post, v)
	if err != nil {
		switch {
		case errors.Is(err, api.ErrRecordNotFound):
			app.clientError(r, w, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// looking for errors from the API
	if !v.Valid() {

		// retrieving basic template data
		tmplData := app.newTemplateData(r)

		tmplData.NonFieldErrors = form.NonFieldErrors
		tmplData.FieldErrors = form.FieldErrors

		// render the template
		app.render(w, r, http.StatusOK, "post-update.tmpl", tmplData)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Post updated successfully!")
	http.Redirect(w, r, fmt.Sprintf("/thread/%d", post.Thread.ID), http.StatusSeeOther)
}
