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
	/*	RESTRICTED
	/* #############################################################################*/

	router.Group(func(group *flow.Mux) {

		group.Use(app.requireAuthentication)

		group.HandleFunc("/dashboard", app.dashboard, http.MethodGet)  // dashboard page
		group.HandleFunc("/logout", app.logoutPost, http.MethodPost)   // logout route
		group.HandleFunc("/user", app.updateUser, http.MethodGet)      // update user page
		group.HandleFunc("/user", app.updateUserPost, http.MethodPost) // update user treatment route

		// TODO -> add delete user and more to complete the user management options

		group.HandleFunc("/post/create", app.createPost, http.MethodGet)          // post creation page
		group.HandleFunc("/post/create", app.createPostPost, http.MethodPost)     // post creation treatment route
		group.HandleFunc("/post/:id/update", app.updatePost, http.MethodGet)      // post update page
		group.HandleFunc("/post/:id/update", app.updatePostPost, http.MethodPost) // post update treatment route

		group.HandleFunc("/author", app.updateAuthor, http.MethodGet)      // author update page
		group.HandleFunc("/author", app.updateAuthorPost, http.MethodPost) // author update treatment route

		// TODO -> add delete post and more to complete the posts management options

	})

	/* #############################################################################
	/*	COMMON
	/* #############################################################################*/

	router.HandleFunc("/", app.index, http.MethodGet)            // landing page
	router.HandleFunc("/home", app.index, http.MethodGet)        // landing page
	router.HandleFunc("/policies", app.policies, http.MethodGet) // policies page

	router.HandleFunc("/post/:id", app.postIncrementView, http.MethodPost) // AJAX call increment post view
	router.HandleFunc("/post/:id", app.postGet, http.MethodGet)            // post page

	router.HandleFunc("/search", app.search, http.MethodGet) // search page

	router.HandleFunc("/contact", app.contact, http.MethodPost) // contact message treatment page

	/* #############################################################################
	/*	USER ACCESS
	/* #############################################################################*/

	router.HandleFunc("/login", app.login, http.MethodGet)      // login page
	router.HandleFunc("/login", app.loginPost, http.MethodPost) // login treatment route

	router.HandleFunc("/register", app.register, http.MethodGet)      // register page
	router.HandleFunc("/register", app.registerPost, http.MethodPost) // register treatment route

	router.HandleFunc("/activation/:token", app.activate, http.MethodGet) // activation page
	router.HandleFunc("/activation", app.activatePost, http.MethodPost)   // activation treatment route

	router.HandleFunc("/forgot-password", app.forgotPassword, http.MethodGet)      // forgot password page
	router.HandleFunc("/forgot-password", app.forgotPasswordPost, http.MethodPost) // forgot password treatment route

	router.HandleFunc("/reset-password/:token", app.resetPassword, http.MethodGet) // reset password page
	router.HandleFunc("/reset-password", app.resetPasswordPost, http.MethodPost)   // reset password treatment route

	return router
}
