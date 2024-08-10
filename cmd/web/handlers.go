package main

import (
	"Portfolio/internal/data"
	"Portfolio/internal/validator"
	"errors"
	"fmt"
	"github.com/alexedwards/flow"
	"net/http"
	"time"
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
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Search"

	// retrieving the research text
	tmplData.Search = r.URL.Query().Get("q")

	// search in the posts
	var err error
	tmplData.Posts.List, tmplData.Posts.Metadata, err = app.models.PostModel.Get(tmplData.Search, data.NewPostFilters(r.URL.Query()))
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.clientError(w, r, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// render the template
	app.render(w, r, http.StatusOK, "search.tmpl", tmplData)
}

func (app *application) postGet(w http.ResponseWriter, r *http.Request) {

	// fetching the post ID
	id, err := getPathID(r)
	if err != nil {
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// retrieving basic template data
	tmplData := app.newTemplateData(r)

	// fetching the post
	tmplData.Post, err = app.models.PostModel.GetByID(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.clientError(w, r, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}
	tmplData.Title = fmt.Sprintf("Antoine de Barbarin - %s", tmplData.Post.Title)

	// render the template
	app.render(w, r, http.StatusOK, "post.tmpl", tmplData)
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
		app.clientError(w, r, http.StatusBadRequest)
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
		app.failedValidationError(w, r, form, &form.Validator, "register.tmpl")
		return
	}

	// creating the user
	user := &data.User{
		Name:   form.Username,
		Email:  form.Email,
		Status: data.UserToActivate,
	}

	// setting the password hash
	err = user.Password.Set(form.Password)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// verifying the user data
	v := validator.New()
	if data.ValidateUser(v, user); !v.Valid() {

		// redirect to login page with errors
		app.failedValidationError(w, r, form, v, "register.tmpl")
		return
	}

	// inserting the user in the DB
	err = app.models.UserModel.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddFieldError("email", "a user with this email address already exists")
			app.failedValidationError(w, r, form, v, "login.tmpl")
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// Generate an activation token and send a mail to the user
	token, err := app.models.TokenModel.New(user.ID, 3*24*time.Hour, data.TokenActivation)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.background(func() {

		mailData := map[string]any{
			"userID":          user.ID,
			"username":        user.Name,
			"activationToken": token.Plaintext,
		}

		err = app.mailer.Send(user.Email, "user_welcome.tmpl", mailData)
		if err != nil {
			app.logger.Error(err.Error())
		}
	})

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
		app.clientError(w, r, http.StatusBadRequest)
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
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// checking the data from the user and return to confirm page if there is an error
	if form.ValidateToken(form.Token); !form.Valid() {
		app.failedValidationError(w, r, form, &form.Validator, "confirm.tmpl")
		return
	}

	v := validator.New()

	// fetching the user with the token
	user, err := app.models.UserModel.GetForToken(data.TokenActivation, form.Token)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddFieldError("token", "invalid or expired activation link")
			app.failedValidationError(w, r, form, v, "confirm.tmpl")
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// activating the user
	user.Status = data.UserActivated

	// updating the user data
	err = app.models.UserModel.Update(user)
	if err != nil {
		app.serverError(w, r, err)
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
	app.render(w, r, http.StatusOK, "login.tmpl", tmplData)
}

func (app *application) loginPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newUserLoginForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// checking the data from the user
	form.Check(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.Check(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	if form.ValidatePassword(form.Password); !form.Valid() {
		app.failedValidationError(w, r, form, &form.Validator, "login.tmpl")
		return
	}

	// fetching the user with the mail address
	user, err := app.models.UserModel.GetByEmail(form.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			form.AddNonFieldError("invalid credentials")
			app.failedValidationError(w, r, form, &form.Validator, "login.tmpl")
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// matching the password
	match, err := user.Password.Matches(form.Password)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// checking the password match
	if !match {
		form.AddNonFieldError("invalid credentials")
		app.failedValidationError(w, r, form, &form.Validator, "login.tmpl")
		return
	}

	// renewing the user session
	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// storing the user id in the user session
	app.sessionManager.Put(r.Context(), authenticatedUserIDSessionManager, user.ID)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (app *application) forgotPassword(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Forgot password"

	// render the template
	app.render(w, r, http.StatusOK, "forgot-password.tmpl", tmplData)
}

func (app *application) forgotPasswordPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newForgotPasswordForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// checking the data from the user
	form.ValidateEmail(form.Email)

	// return to forgot-password page if there is an error
	if !form.Valid() {
		app.failedValidationError(w, r, form, &form.Validator, "forgot-password.tmpl")
		return
	}

	// fetching the user
	user, err := app.models.UserModel.GetByEmail(form.Email)

	// if the user exists
	if nil == err {

		// Generate a reset token and send a mail if the user exists
		token, err := app.models.TokenModel.New(user.ID, 24*time.Hour, data.TokenReset)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		app.background(func() {

			mailData := map[string]any{
				"userID":     user.ID,
				"username":   user.Name,
				"resetToken": token.Plaintext,
			}

			err = app.mailer.Send(user.Email, "forgot_password.tmpl", mailData)
			if err != nil {
				app.logger.Error(err.Error())
			}
		})

	} else if !errors.Is(err, data.ErrRecordNotFound) {
		app.serverError(w, r, err)
		return
	}

	// do this anyway (even if the user doesn't exist)
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
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// render the template
	app.render(w, r, http.StatusOK, "reset-password.tmpl", tmplData)
}

func (app *application) resetPasswordPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newResetPasswordForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// checking the data from the user and return to reset password page if there is an error
	form.ValidateNewPassword(form.NewPassword, form.ConfirmPassword)
	if form.ValidateToken(form.Token); !form.Valid() {
		app.failedValidationError(w, r, form, &form.Validator, "reset-password.tmpl")
		return
	}

	v := validator.New()

	// fetching the user with the token
	user, err := app.models.UserModel.GetForToken(data.TokenReset, form.Token)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddFieldError("token", "invalid or expired link")
			app.failedValidationError(w, r, form, v, "reset-password.tmpl")
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// setting the new password
	err = user.Password.Set(form.NewPassword)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// updating the user with the new password
	err = app.models.UserModel.Update(user)
	if err != nil {
		app.serverError(w, r, err)
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

	// logging the user out
	err := app.logout(r)
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
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// fetching the authenticated user
	user, err := app.models.UserModel.GetByID(app.getUserID(r))
	if err != nil {
		app.serverError(w, r, err)
		return
	}

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
		err = user.Password.Set(*form.NewPassword)
		if err != nil {
			app.serverError(w, r, err)
			return
		}
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
		app.failedValidationError(w, r, form, &form.Validator, "user-update.tmpl")
		return
	}

	// updating the user
	v := validator.New()
	err = app.models.UserModel.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.clientError(w, r, http.StatusNotFound)
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddFieldError("email", "email is already in use")
			app.failedValidationError(w, r, form, v, "user-update.tmpl")
		default:
			app.serverError(w, r, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Your data has been updated successfully!")
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (app *application) createPost(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Insert Post"

	// render the template
	app.render(w, r, http.StatusOK, "post-create.tmpl", tmplData)
}

func (app *application) createPostPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newPostForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// DEBUG
	app.logger.Debug(fmt.Sprintf("content: %+v", form.Content))
	app.logger.Debug(fmt.Sprintf("title: %+v", *form.Title))

	// creating the new thread
	post := &data.Post{}

	// checking the data from the user
	if form.Content == nil {
		form.AddFieldError("content", "must be provided")
	} else {
		form.Check(len(post.Content) > 2, "content", "must be at least 2 bytes long")
		form.Check(len(post.Content) < 1_020, "content", "must not be more than 1.020 bytes long")
		post.Content = form.Content
	}
	if form.Title == nil {
		form.AddFieldError("title", "must be provided")
	} else {
		form.StringCheck(*form.Title, 2, 120, true, "title")
	}

	// return to post-create page if there is an error
	if !form.Valid() {
		app.failedValidationError(w, r, form, &form.Validator, "post-create.tmpl")
		return
	}

	// creating the post
	err = app.models.PostModel.Insert(post)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicatePostTitle):
			form.AddFieldError("title", "is already in use")
			app.failedValidationError(w, r, form, &form.Validator, "post-create.tmpl")
		default:
			app.serverError(w, r, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Post created successfully!")
	http.Redirect(w, r, fmt.Sprintf("/post/%d", post.ID), http.StatusSeeOther)
}

func (app *application) updatePost(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Update post"

	// retrieving the post id from the path
	id, err := getPathID(r)
	if err != nil {
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// retrieving the post from the API
	post, err := app.models.PostModel.GetByID(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.clientError(w, r, http.StatusNotFound)
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
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// creating the updated post
	post := &data.Post{}

	// checking the data from the user
	if form.Content != nil {
		form.Check(len(post.Content) > 2, "content", "must be at least 2 bytes long")
		form.Check(len(post.Content) < 1_020, "content", "must not be more than 1.020 bytes long")
		post.Content = form.Content
	}
	if form.Title != nil {
		form.StringCheck(*form.Title, 1, 120, false, "title")
		post.Title = *form.Title
	}

	// return to post-update page if there is an error
	if !form.Valid() {
		app.failedValidationError(w, r, form, &form.Validator, "post-update.tmpl")
		return
	}

	// retrieving the post id from the path
	post.ID, err = getPathID(r)
	if err != nil {
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// API request to update a post
	err = app.models.PostModel.Update(*post)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.clientError(w, r, http.StatusNotFound)
		case errors.Is(err, data.ErrDuplicatePostTitle):
			form.AddFieldError("title", "title is already in use")
			app.failedValidationError(w, r, form, &form.Validator, "post-update.tmpl")
		default:
			app.serverError(w, r, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Post updated successfully!")
	http.Redirect(w, r, fmt.Sprintf("/post/%d", post.ID), http.StatusSeeOther)
}
