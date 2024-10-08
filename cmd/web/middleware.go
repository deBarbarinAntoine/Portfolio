package main

import (
	"context"
	"fmt"
	"github.com/justinas/nosurf"
	"log/slog"
	"net/http"
)

const (
	authenticatedUserIDSessionManager = "authenticated_user_id"
)

func commonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// generating a nonce for the script embedded in the templates
		nonce, err := newNonce()
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// putting the nonce in the context
		ctx := context.WithValue(r.Context(), nonceContextKey, nonce)
		r = r.WithContext(ctx)

		// setting the common headers
		w.Header().Set("Content-Security-Policy", fmt.Sprintf("script-src 'self' 'nonce-%s' https://cdn.jsdelivr.net", nonce))
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")

		w.Header().Set("Server", "Golang server")

		next.ServeHTTP(w, r)
	})
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var (
			ip     = r.RemoteAddr
			proto  = r.Proto
			method = r.Method
			uri    = r.URL.RequestURI()
		)

		app.logger.Debug("received request", slog.String("ip", ip), slog.String("protocol", proto), slog.String("method", method), slog.String("URI", uri))

		next.ServeHTTP(w, r)
	})
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverError(w, r, fmt.Errorf("recovering from panic: %s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func noSurf(next http.Handler) http.Handler {

	csrfHandler := nosurf.New(next)

	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
	})

	return csrfHandler
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// getting the userID from the session
		id := app.sessionManager.GetInt(r.Context(), authenticatedUserIDSessionManager)

		// if user is not authenticated
		if id == 0 {
			next.ServeHTTP(w, r)
			return
		}

		exists, err := app.models.UserModel.Exists(id)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		if exists {
			// setting the user as authenticated in the context
			ctx := context.WithValue(r.Context(), isAuthenticatedContextKey, true)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !app.isAuthenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		w.Header().Add("Cache-Control", "no-store")

		next.ServeHTTP(w, r)
	})
}
