package handler

import (
	"barrel-api/controller"

	"github.com/gorilla/mux"
)

type OAuthHandler struct {
	oauthController *controller.OAuthController
}

func NewOAuthHandler(oauthController *controller.OAuthController) *OAuthHandler {
	return &OAuthHandler{oauthController}
}

// RegisterRoutes registra os endpoints OAuth2 no subrouter /auth/v1.
// Não passam pelo AuthenticationMiddleware — são chamados sem token.
func (h *OAuthHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/oauth/authorize", h.oauthController.AuthorizeHandler).Methods("GET", "POST")
	r.HandleFunc("/oauth/token", h.oauthController.TokenHandler).Methods("POST")
}
