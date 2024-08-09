package main

import (
	"Portfolio/ui"
	"github.com/alexedwards/flow"
	"io/fs"
	"net/http"
)

func (app *application) routes() http.Handler {

	// setting the files to put in the static handler
	staticFs, err := fs.Sub(ui.StaticFiles, "assets")
	if err != nil {
		panic(err)
	}

	router := flow.New()

	router.NotFound = http.HandlerFunc(app.notFound)                 // error 404 page
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowed) // error 405 page

	router.Handle("/static/...", http.StripPrefix("/static/", http.FileServerFS(staticFs)), http.MethodGet) // static files

	router.Use(app.recoverPanic, app.logRequest, commonHeaders, app.sessionManager.LoadAndSave, noSurf, app.authenticate)

	/* #############################################################################
	/*	COMMON
	/* #############################################################################*/

	router.HandleFunc("/", app.index, http.MethodGet)      // landing page
	router.HandleFunc("/home", app.index, http.MethodGet)  // landing page
	router.HandleFunc("/about", app.about, http.MethodGet) // about page

	router.HandleFunc("/thread/:id", app.postGet, http.MethodGet) // post page

	router.HandleFunc("/search", app.search, http.MethodGet) // search page

	/* #############################################################################
	/*	USER ACCESS
	/* #############################################################################*/

	router.HandleFunc("/login", app.login, http.MethodGet)      // login page
	router.HandleFunc("/login", app.loginPost, http.MethodPost) // login treatment route

	router.HandleFunc("/register", app.register, http.MethodGet)      // register page
	router.HandleFunc("/register", app.registerPost, http.MethodPost) // register treatment route

	router.HandleFunc("/confirm/:token", app.confirm, http.MethodGet) // confirmation page
	router.HandleFunc("/confirm", app.confirmPost, http.MethodPost)   // confirmation treatment route

	router.HandleFunc("/forgot-password", app.forgotPassword, http.MethodGet)      // forgot password page
	router.HandleFunc("/forgot-password", app.forgotPasswordPost, http.MethodPost) // forgot password treatment route

	router.HandleFunc("/reset-password/:token", app.resetPassword, http.MethodGet) // reset password page
	router.HandleFunc("/reset-password", app.resetPasswordPost, http.MethodPost)   // reset password treatment route

	/* #############################################################################
	/*	RESTRICTED
	/* #############################################################################*/

	router.Use(app.requireAuthentication)

	router.HandleFunc("/dashboard", app.dashboard, http.MethodGet) // dashboard page
	router.HandleFunc("/logout", app.logoutPost, http.MethodPost)  // logout route
	router.HandleFunc("/user", app.updateUser, http.MethodGet)     // update user page
	router.HandleFunc("/user", app.updateUserPut, http.MethodPut)  // update user treatment route

	router.HandleFunc("/post/create", app.createPost, http.MethodGet)        // post creation page
	router.HandleFunc("/post", app.createPostPost, http.MethodPost)          // post creation treatment route
	router.HandleFunc("/post/:id/update", app.updatePost, http.MethodGet)    // post update page
	router.HandleFunc("/post/:id/update", app.updatePostPut, http.MethodPut) // category update treatment route

	return router
}
