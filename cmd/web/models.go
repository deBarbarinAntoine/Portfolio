package main

import (
	"Portfolio/internal/data"
	"Portfolio/internal/validator"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	"html/template"
	"log/slog"
	"time"
)

type config struct {
	port int64
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}

	smtp struct {
		host     string
		port     int64
		username string
		password string
		sender   string
	}
}

type application struct {
	logger         *slog.Logger
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
	models         data.Models
	config         *config
}

type templateData struct {
	Title           string
	CurrentYear     int
	Form            any
	Flash           string
	IsAuthenticated bool
	Nonce           string
	CSRFToken       string
	ActivationToken string
	ResetToken      string
	Error           struct {
		Title   string
		Message string
	}
	FieldErrors    map[string]string
	NonFieldErrors []string
	User           data.User
	Search         string
	Post           data.Post
	Posts          struct {
		List     []data.Post
		Metadata data.Metadata
	}
}

type userLoginForm struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

type userUpdateForm struct {
	Username             *string `form:"username,omitempty"`
	Email                *string `form:"email,omitempty"`
	Password             *string `form:"password,omitempty"`
	NewPassword          *string `form:"new_password,omitempty"`
	ConfirmationPassword *string `form:"confirmation_password,omitempty"`
	Avatar               *string `form:"avatar,omitempty"`
	validator.Validator  `form:"-"`
}

type userRegisterForm struct {
	Username            string `form:"username"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	ConfirmPassword     string `form:"confirm_password"`
	validator.Validator `form:"-"`
}

type userConfirmForm struct {
	Token               string `form:"token"`
	validator.Validator `form:"-"`
}

type forgotPasswordForm struct {
	Email               string `form:"email"`
	validator.Validator `form:"-"`
}

type resetPasswordForm struct {
	Token               string `form:"token"`
	NewPassword         string `form:"new_password"`
	ConfirmPassword     string `form:"confirm_password"`
	validator.Validator `form:"-"`
}

type postForm struct {
	Title               *string `form:"title,omitempty"`
	Content             *string `form:"content,omitempty"`
	validator.Validator `form:"-"`
}
