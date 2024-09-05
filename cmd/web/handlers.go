package main

import (
	"Portfolio/internal/data"
	"Portfolio/internal/uploads"
	"Portfolio/internal/validator"
	"errors"
	"fmt"
	"github.com/alexedwards/flow"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

/* #############################################################################
/*	COMMON
/* #############################################################################*/

func (app *application) notFound(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Not Found"

	// rendering the template
	app.render(w, r, http.StatusOK, "error.tmpl", tmplData)
}

func (app *application) methodNotAllowed(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Oooops"

	// setting the error title and message
	tmplData.Error.Title = "Error 405"
	tmplData.Error.Message = "Something went wrong!"

	// rendering the template
	app.render(w, r, http.StatusOK, "error.tmpl", tmplData)
}

func (app *application) index(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Home"

	// setting the contact form
	tmplData.Form = newContactForm()

	// rendering the template
	app.render(w, r, http.StatusOK, "home.tmpl", tmplData)
}

func (app *application) policies(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Policies"

	// rendering the template
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

	// rendering the template
	app.render(w, r, http.StatusOK, "search.tmpl", tmplData)
}

func (app *application) latestPosts(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Latest Posts"

	// get the latest posts
	var err error
	tmplData.Posts.List, tmplData.Posts.Metadata, err = app.models.PostModel.Get("", data.NewPostFilters(url.Values{"sort": []string{"-created_at"}}))
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.clientError(w, r, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// rendering the template
	app.render(w, r, http.StatusOK, "latest.tmpl", tmplData)
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

	// activating the PostIncrementView AJAX call in the template
	tmplData.IsPostView = true

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

	// rendering the template
	app.render(w, r, http.StatusOK, "post.tmpl", tmplData)
}

func (app *application) contact(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newContactForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// DEBUG
	app.logger.Debug(fmt.Sprintf("form: %+v", form))

	// checking the form data
	form.StringCheck(form.Name, 2, 70, true, "name")
	form.ValidateEmail(form.Email)
	form.StringCheck(form.Message, 10, 2_500, true, "message")

	// redirect if the data is invalid
	if !form.Valid() {
		app.failedValidationError(w, r, form, &form.Validator, "home.tmpl")
		return
	}

	// retrieving the author email address
	author, err := app.models.AuthorModel.Get()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// sending the mail
	app.background(func() {

		err = app.mailer.Send(author.Email, "contact-message.tmpl", form)
		if err != nil {
			app.logger.Error(err.Error())
		}
	})

	// notifying the user with a flash message and redirecting to the home page
	app.sessionManager.Put(r.Context(), "flash", "Your message has been sent successfully!")
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

/* #############################################################################
/*	USER ACCESS
/* #############################################################################*/

func (app *application) register(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Register"

	// filling the form with empty values
	tmplData.Form = newUserRegisterForm()

	// rendering the template
	app.render(w, r, http.StatusOK, "register.tmpl", tmplData)
}

func (app *application) registerPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newUserRegisterForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
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
		Avatar: "/static/img/avatar.png", // TODO -> implement avatar in all user handlers
		Status: data.UserToActivate,
	}

	// setting the password hash
	err = user.Password.Set(form.Password)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// verifying the user data
	if data.ValidateUser(&form.Validator, user); !form.Valid() {

		// redirect to login page with errors
		app.failedValidationError(w, r, form, &form.Validator, "register.tmpl")
		return
	}

	// inserting the user in the DB
	err = app.models.UserModel.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			form.AddFieldError("email", "a user with this email address already exists")
			app.failedValidationError(w, r, form, &form.Validator, "register.tmpl")
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// Generating an activation token to send it via mail to the user
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

func (app *application) activate(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Activation"

	// retrieving the activation token from the URL and checking it
	form := newUserActivationForm()
	form.ActivationToken = flow.Param(r.Context(), "token")
	if form.ValidateToken(form.ActivationToken); !form.Valid() {
		app.failedValidationError(w, r, form, &form.Validator, "activation.tmpl")
		return
	}

	// putting the form in the template data
	tmplData.Form = form

	// rendering the template
	app.render(w, r, http.StatusOK, "activation.tmpl", tmplData)
}

func (app *application) activatePost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newUserActivationForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// checking the data from the user and return to activation page if there is an error
	if form.ValidateToken(form.ActivationToken); !form.Valid() {
		app.failedValidationError(w, r, form, &form.Validator, "activation.tmpl")
		return
	}

	// fetching the user with the token
	user, err := app.models.UserModel.GetForToken(data.TokenActivation, form.ActivationToken)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			form.AddFieldError("token", "invalid or expired activation link")
			app.failedValidationError(w, r, form, &form.Validator, "activation.tmpl")
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

	// filling the form with empty values
	tmplData.Form = newUserLoginForm()

	// rendering the template
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

	// filling the form with empty values
	tmplData.Form = newForgotPasswordForm()

	// rendering the template
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

	// filling the form with empty values
	form := newResetPasswordForm()

	// retrieving the reset token from the URL
	form.Token = flow.Param(r.Context(), "token")
	if form.ValidateToken(form.Token); !form.Valid() {
		app.failedValidationError(w, r, form, &form.Validator, "reset-password.tmpl")
		return
	}

	// putting the form in the template data
	tmplData.Form = form

	// rendering the template
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

	// fetching the user with the token
	user, err := app.models.UserModel.GetForToken(data.TokenReset, form.Token)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			form.AddFieldError("token", "invalid or expired link")
			app.failedValidationError(w, r, form, &form.Validator, "reset-password.tmpl")
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

	// rendering the template
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

func (app *application) updateAuthor(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Update Author"

	// filling the form with values
	author, err := app.models.AuthorModel.Get()
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	form := app.newAuthorUpdateForm()
	form.Name = &author.Name
	form.Email = &author.Email
	form.Avatar = &author.Avatar
	form.Presentation = author.Presentation
	form.Experiences = author.Experiences
	form.Formations = author.Formations
	form.Location = &author.Location
	form.Birth = &author.Birth
	form.Tags = author.Tags
	form.CVFile = &author.CVFile
	form.StatusActivity = &author.StatusActivity
	tmplData.Form = form

	app.render(w, r, http.StatusOK, "author-update.tmpl", tmplData)
}

func (app *application) updateAuthorPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := app.newAuthorUpdateForm()
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// getting the author data
	author, err := app.models.AuthorModel.Get()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// checking the data from the user
	if form.Name != nil {
		author.Name = *form.Name
	}
	if form.Email != nil {
		author.Email = *form.Email
	}
	if form.Avatar != nil {
		author.Avatar = *form.Avatar
	}
	author.Presentation = form.Presentation
	if form.Location != nil {
		author.Location = *form.Location
	}
	if form.Birth != nil {
		author.Birth = *form.Birth
	}
	author.Formations = form.Formations
	author.Experiences = form.Experiences
	author.Tags = form.Tags
	if form.CVFile != nil {
		author.CVFile = *form.CVFile
	}
	if form.StatusActivity != nil {
		author.StatusActivity = *form.StatusActivity
	}

	if author.Validate(&form.Validator); !form.Valid() {
		app.failedValidationError(w, r, form, &form.Validator, "author-update.tmpl")
		return
	}

	// updating the user
	err = app.models.AuthorModel.Update(author)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.clientError(w, r, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "The author data has been updated successfully!")
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (app *application) updateUser(w http.ResponseWriter, r *http.Request) {

	// retrieving basic template data
	tmplData := app.newTemplateData(r)
	tmplData.Title = "Antoine de Barbarin - Update user"

	// retrieving user ID
	id := app.getUserID(r)

	// fetching user data
	user, err := app.models.UserModel.GetByID(id)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// filling the form with user values
	tmplData.Form = newUserUpdateForm(user)

	// rendering the template
	app.render(w, r, http.StatusOK, "user-update.tmpl", tmplData)
}

func (app *application) updateUserPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newUserUpdateForm(nil)
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
	err = app.models.UserModel.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.clientError(w, r, http.StatusNotFound)
		case errors.Is(err, data.ErrDuplicateEmail):
			form.AddFieldError("email", "email is already in use")
			app.failedValidationError(w, r, form, &form.Validator, "user-update.tmpl")
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
	tmplData.Title = "Antoine de Barbarin - Create Post"

	// filling the form with empty values
	tmplData.Form = newPostForm(nil)

	// rendering the template
	app.render(w, r, http.StatusOK, "post-form.tmpl", tmplData)
}

func (app *application) createPostPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newPostForm(nil)
	err := app.decodePostForm(r, &form)
	if err != nil {
		// DEBUG
		app.logger.Debug(fmt.Errorf("decoding postForm error: %w", err).Error())
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// DEBUG
	app.logger.Debug(fmt.Sprintf("form: %+v", *form))

	// creating the new thread
	post := &data.Post{}

	// checking the data from the user
	form.StringCheck(form.Content, 2, 10_000, true, "content")
	post.Content = []byte(form.Content)
	if form.Title == nil {
		form.AddFieldError("title", "must be provided")
	} else {
		form.StringCheck(*form.Title, 2, 120, true, "title")
		post.Title = *form.Title
	}
	if form.Images == nil {
		form.AddFieldError("images", "must be provided")
	} else {
		post.Images = form.Images
	}

	// return to post-create page if there is an error
	if !form.Valid() {
		app.failedValidationError(w, r, form, &form.Validator, "post-form.tmpl")
		return
	}

	// creating the post
	err = app.models.PostModel.Insert(post)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicatePostTitle):
			form.AddFieldError("title", "is already in use")
			app.failedValidationError(w, r, form, &form.Validator, "post-form.tmpl")
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
	tmplData.Form = newPostForm(post)

	// rendering the template
	app.render(w, r, http.StatusOK, "post-form.tmpl", tmplData)
}

func (app *application) updatePostPost(w http.ResponseWriter, r *http.Request) {

	// retrieving the form data
	form := newPostForm(nil)
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// DEBUG
	app.logger.Debug(fmt.Sprintf("form: %+v", *form))

	// creating the updated post
	post := &data.Post{}

	// retrieving the post id from the path
	post.ID, err = getPathID(r)
	if err != nil {
		app.clientError(w, r, http.StatusBadRequest)
		return
	}

	// retrieving the post from the DB
	post, err = app.models.PostModel.GetByID(post.ID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.clientError(w, r, http.StatusNotFound)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	// checking the data from the user
	form.StringCheck(form.Content, 2, 10_000, true, "content")
	post.Content = []byte(form.Content)
	if form.Title != nil {
		form.StringCheck(*form.Title, 1, 120, false, "title")
		post.Title = *form.Title
	}
	if form.Images != nil {
		form.Check(len(form.Images) < 6, "images", "limit: 5 images max")
		post.Images = form.Images
	}

	// return to post-update page if there is an error
	if !form.Valid() {
		app.failedValidationError(w, r, form, &form.Validator, "post-form.tmpl")
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
			app.failedValidationError(w, r, form, &form.Validator, "post-form.tmpl")
		default:
			app.serverError(w, r, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Post updated successfully!")
	http.Redirect(w, r, fmt.Sprintf("/post/%d", post.ID), http.StatusSeeOther)
}

/* #############################################################################
/*	AJAX CALLS
/* #############################################################################*/

func (app *application) postIncrementView(w http.ResponseWriter, r *http.Request) {

	// fetch post id
	id, err := getPathID(r)
	if err != nil {
		// send the error back in JSON
		app.ajaxResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// increment post view
	err = app.models.PostModel.IncrementViews(id)
	if err != nil {
		// send the error back in JSON
		app.ajaxResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// send the positive response back in JSON
	app.ajaxResponse(w, http.StatusOK, "post view incremented successfully!")
}

func (app *application) uploadFile(w http.ResponseWriter, r *http.Request) {

	// getting the file from the form
	file, header, err := r.FormFile("file")
	if err != nil {
		app.ajaxResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer file.Close()

	// uploading the file
	msg, err := uploads.Add(file, header)
	if err != nil {
		app.ajaxResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// DEBUG
	app.logger.Debug(msg)

	// respond with a validation message and the image name
	app.ajaxResponse(w, http.StatusOK, msg)
}

func (app *application) deleteFile(w http.ResponseWriter, r *http.Request) {

	// getting directory and file names
	dir := flow.Param(r.Context(), "dir")
	file := flow.Param(r.Context(), "file")

	// retrieving the unescaped path for the directory
	var err error
	if dir != "" {
		dir, err = url.PathUnescape(dir)
		if err != nil {
			app.ajaxResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// deleting the file
	err = uploads.Remove(filepath.Join(dir, file))
	if err != nil {
		app.ajaxResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// sending the positive response
	app.ajaxResponse(w, http.StatusOK, fmt.Sprintf("file %s successfully deleted from %s!", file, dir))
}

func (app *application) getFiles(w http.ResponseWriter, r *http.Request) {

	// setting the Browser variable
	var browser uploads.Browser

	// getting directory name
	browser.Dirname = flow.Param(r.Context(), "dir")

	// retrieving the unescaped path for the directory
	var err error
	if browser.Dirname != "" {
		browser.Dirname = strings.ReplaceAll(browser.Dirname, "|2F", "/")
	}

	// getting the files
	browser.Files, err = uploads.Get(browser.Dirname)
	if err != nil {
		app.ajaxResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// parsing the template for the file browser
	err = uploads.Render(w, browser)
	if err != nil {
		app.ajaxResponse(w, http.StatusInternalServerError, err.Error())
	}
}
